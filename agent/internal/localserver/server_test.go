package localserver

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/Goldpip3/albion-info-tracker/agent/internal/domain"
	"github.com/Goldpip3/albion-info-tracker/agent/internal/push"
)

func newTestServer(t *testing.T, onCmd func(action, arg string)) *httptest.Server {
	t.Helper()
	s, err := New(Options{
		Snapshot:     func() domain.Snapshot { return domain.Snapshot{} },
		DirtyGen:     func() uint64 { return 0 },
		OnCommand:    onCmd,
		SendInterval: 20 * time.Millisecond,
		Heartbeat:    50 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)
	return ts
}

// TestServesUI confirms the embedded bundle is wired: "/" returns the SPA
// shell, and an unknown route (a client-side path) falls back to it too.
func TestServesUI(t *testing.T) {
	ts := newTestServer(t, nil)

	for _, path := range []string{"/", "/loot", "/party"} {
		resp, err := http.Get(ts.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("GET %s: status %d", path, resp.StatusCode)
		}
		if !strings.Contains(strings.ToLower(string(body)), "<div id=\"root\"") &&
			!strings.Contains(strings.ToLower(string(body)), "<!doctype html") {
			t.Fatalf("GET %s: body doesn't look like the SPA shell: %.80q", path, body)
		}
	}
}

// TestViewStreamsSnapshot confirms a /view client gets an immediate snapshot
// envelope in the exact shape the web UI parses.
func TestViewStreamsSnapshot(t *testing.T) {
	ts := newTestServer(t, nil)
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/view"

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	c, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("dial /view: %v", err)
	}
	defer c.CloseNow()

	_, data, err := c.Read(ctx)
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	var env push.Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	if env.V != push.ProtocolVersion {
		t.Fatalf("version = %d, want %d", env.V, push.ProtocolVersion)
	}
	if env.Type != "snapshot" || env.Snap == nil {
		t.Fatalf("type=%q snap=%v, want a snapshot envelope", env.Type, env.Snap)
	}
}

// TestViewForwardsCommand confirms a command envelope from a viewer reaches
// OnCommand — the path that powers New Session / Clear meter / scope changes
// in local mode.
func TestViewForwardsCommand(t *testing.T) {
	got := make(chan [2]string, 1)
	ts := newTestServer(t, func(action, arg string) { got <- [2]string{action, arg} })
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/view"

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	c, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("dial /view: %v", err)
	}
	defer c.CloseNow()

	cmd := push.Envelope{V: push.ProtocolVersion, Type: "command", Command: &push.CommandMessage{Action: "clearMeter"}}
	b, _ := json.Marshal(cmd)
	if err := c.Write(ctx, websocket.MessageText, b); err != nil {
		t.Fatalf("write command: %v", err)
	}

	select {
	case g := <-got:
		if g[0] != "clearMeter" {
			t.Fatalf("action = %q, want clearMeter", g[0])
		}
	case <-time.After(2 * time.Second):
		t.Fatal("OnCommand was not called within 2s")
	}
}
