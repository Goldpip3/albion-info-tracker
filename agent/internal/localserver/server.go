// Package localserver serves the meter web UI and a /view WebSocket directly
// from the agent, so the whole meter can run on the local machine with no
// Cloudflare in the data path — and therefore no daily request budget.
//
// The SAME React bundle that's deployed to Cloudflare Pages is embedded here
// (copied into webdist/ at build time). When that bundle is served from
// http://localhost it auto-detects "local mode" and opens its data
// WebSocket against /view on this same origin instead of the Cloudflare
// Worker. Serving the page and the socket from one origin keeps the browser
// happy (no mixed-content / cross-origin blocking).
package localserver

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/coder/websocket"

	"github.com/Goldpip3/albion-info-tracker/agent/internal/domain"
	"github.com/Goldpip3/albion-info-tracker/agent/internal/push"
)

// webFS holds the built web UI. "all:" includes dotfiles / underscore files
// (e.g. _redirects) so the tree matches the Pages deploy exactly.
//
//go:embed all:webdist
var webFS embed.FS

// Options configure a Server.
type Options struct {
	// Addr is the listen address. Bind to loopback (127.0.0.1:<port>) so the
	// listener never triggers a Windows Firewall prompt and isn't reachable
	// from the network.
	Addr string

	// Snapshot/DirtyGen/OnCommand mirror the push.Client hooks — the same
	// engine methods feed both the local server and the Cloudflare push.
	Snapshot  func() domain.Snapshot
	DirtyGen  func() uint64
	OnCommand func(action, arg string)

	// SendInterval is how often a connected viewer is checked for updates.
	// Local mode has no request budget, so it can be snappy. Default 300ms.
	SendInterval time.Duration

	// Heartbeat resends an unchanged snapshot at most this often (keeps the
	// viewer's live/stale indicator green while idle). Default 3s.
	Heartbeat time.Duration
}

// Server serves the embedded UI plus the /view WebSocket.
type Server struct {
	opt   Options
	fsys  fs.FS
	index []byte
	files http.Handler
}

// New builds a Server, failing if the embedded UI is missing its index.html
// (which would mean the webdist/ copy step didn't run before the build).
func New(opt Options) (*Server, error) {
	if opt.SendInterval == 0 {
		opt.SendInterval = 300 * time.Millisecond
	}
	if opt.Heartbeat == 0 {
		opt.Heartbeat = 3 * time.Second
	}
	sub, err := fs.Sub(webFS, "webdist")
	if err != nil {
		return nil, err
	}
	index, err := fs.ReadFile(sub, "index.html")
	if err != nil {
		return nil, fmt.Errorf("embedded web UI is missing index.html (was webdist/ populated before build?): %w", err)
	}
	return &Server{opt: opt, fsys: sub, index: index, files: http.FileServer(http.FS(sub))}, nil
}

// Handler builds the HTTP routes (UI + /view + /healthz). Exposed so tests
// can drive the server through httptest without binding a real port.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/view", s.handleView)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/", s.handleStatic)
	return mux
}

// Run serves until ctx is cancelled. Returns nil on a clean shutdown.
func (s *Server) Run(ctx context.Context) error {
	srv := &http.Server{Addr: s.opt.Addr, Handler: s.Handler()}
	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutCtx)
	}()
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// handleStatic serves embedded files, falling back to index.html for unknown
// paths so client-side routes (/loot, /party, /sessions) load the SPA —
// mirroring the Pages _redirects rule.
func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	p := strings.TrimPrefix(r.URL.Path, "/")
	if p == "" {
		s.serveIndex(w)
		return
	}
	if _, err := fs.Stat(s.fsys, p); err != nil {
		s.serveIndex(w)
		return
	}
	s.files.ServeHTTP(w, r)
}

func (s *Server) serveIndex(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(s.index)
}

// handleView upgrades to a WebSocket and streams snapshots, mirroring the
// Worker's /view endpoint: an immediate replay on connect, then a push on
// every state change (DirtyGen) or at least every Heartbeat. Inbound
// command envelopes are routed to OnCommand, same protocol as the cloud path.
func (s *Server) handleView(w http.ResponseWriter, r *http.Request) {
	// Loopback-only server, so skip the cross-origin check (the page and the
	// socket share the localhost origin anyway).
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		return
	}
	defer c.CloseNow()

	ctx := r.Context()
	go s.readCommands(ctx, c)

	send := func() error {
		if s.opt.Snapshot == nil {
			return nil
		}
		snap := s.opt.Snapshot()
		env := push.Envelope{V: push.ProtocolVersion, Type: "snapshot", TS: time.Now().UnixMilli(), Snap: &snap}
		b, err := json.Marshal(env)
		if err != nil {
			return err
		}
		wctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		return c.Write(wctx, websocket.MessageText, b)
	}

	if err := send(); err != nil {
		return
	}
	lastSent := time.Now()
	var lastGen uint64
	if s.opt.DirtyGen != nil {
		lastGen = s.opt.DirtyGen()
	}

	t := time.NewTicker(s.opt.SendInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if s.opt.DirtyGen != nil {
				gen := s.opt.DirtyGen()
				if gen == lastGen && time.Since(lastSent) < s.opt.Heartbeat {
					continue
				}
				lastGen = gen
			}
			if err := send(); err != nil {
				return
			}
			lastSent = time.Now()
		}
	}
}

// readCommands routes inbound command envelopes to OnCommand. Snapshots are
// large but only ever written here, so the modest default read limit (which
// applies to inbound frames — i.e. the tiny command messages) is fine.
func (s *Server) readCommands(ctx context.Context, c *websocket.Conn) {
	for {
		_, data, err := c.Read(ctx)
		if err != nil {
			return
		}
		var env push.Envelope
		if json.Unmarshal(data, &env) != nil {
			continue
		}
		if env.V != push.ProtocolVersion {
			continue
		}
		if env.Type == "command" && env.Command != nil && s.opt.OnCommand != nil {
			s.opt.OnCommand(env.Command.Action, env.Command.Arg)
		}
	}
}
