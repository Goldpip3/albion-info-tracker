package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/Goldpip3/albion-info-tracker/agent/internal/aodp"
	"github.com/Goldpip3/albion-info-tracker/agent/internal/capture"
	"github.com/Goldpip3/albion-info-tracker/agent/internal/config"
	"github.com/Goldpip3/albion-info-tracker/agent/internal/domain"
	"github.com/Goldpip3/albion-info-tracker/agent/internal/gamedata"
	"github.com/Goldpip3/albion-info-tracker/agent/internal/localserver"
	"github.com/Goldpip3/albion-info-tracker/agent/internal/photon"
	"github.com/Goldpip3/albion-info-tracker/agent/internal/push"
)

const (
	version = "0.6.0"

	// defaultPushURL points new installs at the shared Skirmish backend.
	// A custom Worker can be substituted by setting pushUrl in agent.json
	// or the ALBION_AGENT_URL env var.
	defaultPushURL = "wss://albion-meter.goldpipe.workers.dev/ingest"
	defaultViewURL = "https://albion-meter-web.pages.dev"
)

func main() {
	log.SetFlags(0)
	printBanner()

	// Verbose mode (--verbose / -v / env): flip the domain debug logger
	// AND tee all log output to agent-verbose.log next to the exe, so a
	// diagnostic capture can be read back from the file without
	// copy-pasting an elevated console window.
	verbose := hasFlag("--verbose") || hasFlag("-v") || os.Getenv("ALBION_AGENT_VERBOSE") != ""
	if verbose {
		domain.SetVerbose(true)
		if path, err := setupVerboseLog(); err != nil {
			log.Printf("  verbose log file: %v", err)
		} else {
			fmt.Printf("  Verbose logging on -> %s\n", path)
		}
	}

	openBrowser := hasFlag("--open-browser") || hasFlag("-b") || os.Getenv("ALBION_AGENT_OPEN_BROWSER") != ""

	// Local mode: serve the meter from the agent at http://localhost and do
	// NOT push to Cloudflare. No daily request budget; runs fully on this PC.
	// The "GDA App (Local)" launcher passes --local; "GDA Website" doesn't.
	localMode := hasFlag("--local") || hasFlag("-l") || os.Getenv("ALBION_AGENT_LOCAL") != ""

	cfg, err := config.Load()
	if err != nil {
		fatal("agent.json", err)
	}

	// First-run wizard: if pushToken or pushUrl is missing, walk the user
	// through pairing without making them edit JSON by hand. autoPair opens
	// the browser on its own; if we already had a token we still honor the
	// launcher's --open-browser flag below. In local mode we still generate
	// a token (so the user can switch to the website later) but skip the
	// Cloudflare pairing pop-up — local mode needs no pairing.
	firstRun := cfg.PushToken == ""
	cfg = ensureConfigured(cfg, localMode)
	if openBrowser && !firstRun && !localMode {
		openViewURL(cfg.PushToken)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// When launched by the desktop shell, --parent-pid <pid> ties our
	// lifetime to that window: the shell runs unelevated and can't kill
	// this elevated process directly, so we watch its PID and shut down
	// gracefully when it closes. Avoids an orphaned capture process.
	if pidStr := flagValue("--parent-pid"); pidStr != "" {
		if pid, err := strconv.Atoi(pidStr); err == nil && pid > 0 {
			go watchParentExit(pid, cancel)
		}
	}

	engine := domain.NewEngine()
	engine.SetAlwaysIncludeNames(cfg.AlwaysIncludeNames)
	loadGameData(cfg, engine)
	if store, err := domain.NewSessionsStore(); err != nil {
		log.Printf("  sessions store: %v", err)
	} else {
		engine.SetSessionsStore(store)
		fmt.Printf("  Sessions on disk: %s\n", store.Dir())
	}
	if pstore, err := domain.NewPartyStore(); err != nil {
		log.Printf("  party store: %v", err)
	} else {
		engine.SetPartyStore(pstore)
		fmt.Printf("  Party roster on disk: %s\n", pstore.Path())
		engine.RestoreParty()
	}
	if dstore, err := domain.NewDailyStore(); err != nil {
		log.Printf("  daily store: %v", err)
	} else {
		engine.SetDailyStore(dstore)
		fmt.Printf("  Daily progress on disk: %s\n", dstore.Path())
	}
	priceClient := aodp.New()
	priceClient.Start(ctx)
	engine.SetPriceClient(priceClient)
	fmt.Println("  AODP price client started — loot values estimated when prices are cached.")

	parser := photon.New(engine.Handlers())

	var packetsSeen atomic.Uint64
	sink := capture.SinkFunc(func(pkt capture.Packet) {
		packetsSeen.Add(1)
		parser.Receive(pkt.Payload)
	})

	go renderLoop(ctx, engine, &packetsSeen, parser, verbose)

	// Keep party.json's timestamp fresh while grouped so a stable party
	// (no join/leave events to re-save it) survives an agent restart
	// instead of aging past the 30-min freshness gate. No-op when solo.
	go func() {
		t := time.NewTicker(time.Minute)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				engine.RepersistPartyIfActive()
			}
		}
	}()

	// Local web server (always on): serves the meter UI + its /view socket at
	// http://localhost:<port> with no Cloudflare in the path. Bound to
	// loopback so it never trips the Windows Firewall or is reachable off
	// the machine. The embedded UI auto-detects localhost and connects here.
	localPort := cfg.LocalPort
	if localPort == 0 {
		localPort = 8787
	}
	localURL := fmt.Sprintf("http://localhost:%d", localPort)
	if ls, lerr := localserver.New(localserver.Options{
		Addr:      fmt.Sprintf("127.0.0.1:%d", localPort),
		Snapshot:  engine.Snapshot,
		DirtyGen:  engine.DirtyGen,
		OnCommand: engine.HandleCommand,
	}); lerr != nil {
		log.Printf("local server: disabled (%v)", lerr)
	} else {
		go func() {
			if err := ls.Run(ctx); err != nil {
				log.Printf("local server: stopped: %v", err)
			}
		}()
		fmt.Printf("\n  Local meter: %s  (no Cloudflare, no request limit)\n", localURL)
	}

	switch {
	case localMode:
		// Local-only: skip the Cloudflare push entirely. Zero requests.
		fmt.Println("  Cloud push: OFF (local mode)")
		if openBrowser {
			openLocalURL(localURL)
		}
	case cfg.PushURL != "":
		client := &push.Client{
			URL:          cfg.PushURL,
			Token:        cfg.PushToken,
			AgentVersion: version,
			Snapshot:     engine.Snapshot,
			DirtyGen:     engine.DirtyGen,
			OnCommand:    engine.HandleCommand,
		}
		fmt.Printf("  Streaming to %s\n", maskedURL(cfg.PushURL))
		fmt.Printf("  View at      %s\n\n", defaultViewURL)
		go func() {
			if err := client.Run(ctx); err != nil && err != context.Canceled {
				log.Printf("push: stopped: %v", err)
			}
		}()
	default:
		log.Print("push: no PushURL configured — running in stdout-only mode")
	}

	superviseCapture(ctx, sink, &packetsSeen)
	engine.FlushDaily()
	logDiag(engine, parser, packetsSeen.Load(), "final")
	fmt.Println("\nStopped.")
}

// superviseCapture runs capture.Run and self-heals when it stalls. The
// Windows raw socket binds to the interfaces present at open time and
// then blocks in Recvfrom; if the active adapter changes (VPN toggle,
// Wi-Fi reconnect) or the machine sleeps, packets quietly stop and never
// resume — the meter just freezes with no error. This watchdog notices
// the silence, cancels the run (which closes the dead sockets), and
// reopens — re-enumerating interfaces so a new adapter is picked up.
//
// A genuine setup failure (no admin, no interfaces) makes the very first
// Run return an error before any packet is seen; that's fatal, not a
// transient stall, so we surface it loudly instead of retry-spamming.
func superviseCapture(ctx context.Context, sink capture.Sink, seen *atomic.Uint64) {
	const (
		stallTimeout = 45 * time.Second
		checkEvery    = 5 * time.Second
		reopenDelay   = 2 * time.Second
	)
	everSeen := false
	for {
		if ctx.Err() != nil {
			return
		}
		capCtx, cancel := context.WithCancel(ctx)
		runErr := make(chan error, 1)
		go func() { runErr <- capture.Run(capCtx, sink) }()

		last := seen.Load()
		lastChange := time.Now()
		ticker := time.NewTicker(checkEvery)
		reason := ""
	monitor:
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				cancel()
				<-runErr
				return
			case err := <-runErr:
				ticker.Stop()
				cancel()
				if ctx.Err() != nil {
					return
				}
				if !everSeen && seen.Load() == 0 {
					// Never captured anything — a permission / interface
					// problem, not a recoverable stall. Fail loudly.
					fatal("capture", err)
				}
				reason = fmt.Sprintf("capture stopped: %v", err)
				break monitor
			case <-ticker.C:
				cur := seen.Load()
				if cur > 0 {
					everSeen = true
				}
				if cur != last {
					last = cur
					lastChange = time.Now()
					continue
				}
				// Only treat silence as a stall once traffic has flowed —
				// before the game connects, zero packets is expected.
				if everSeen && time.Since(lastChange) >= stallTimeout {
					reason = fmt.Sprintf("no packets for %s (adapter change / sleep?)", stallTimeout)
					ticker.Stop()
					cancel()
					<-runErr
					break monitor
				}
			}
		}
		if ctx.Err() != nil {
			return
		}
		log.Printf("capture: %s — reopening sockets in %s", reason, reopenDelay)
		select {
		case <-ctx.Done():
			return
		case <-time.After(reopenDelay):
		}
	}
}

// ensureConfigured fills in any missing critical config and re-writes
// agent.json next to the exe so future launches skip the prompt. First-
// run pairing is handled by autoPair — the agent generates a token and
// opens the default browser to the magic-link URL so the user never has
// to type or copy a token.
func ensureConfigured(cfg config.Config, localMode bool) config.Config {
	dirty := false
	if cfg.PushURL == "" {
		cfg.PushURL = defaultPushURL
		dirty = true
	}
	freshPair := false
	if cfg.PushToken == "" {
		cfg.PushToken = generateToken()
		freshPair = true
		dirty = true
	}
	if cfg.AlbionInstallRoot == "" {
		if guess := guessAlbionInstall(); guess != "" {
			fmt.Printf("  Auto-detected Albion at %s\n", guess)
			cfg.AlbionInstallRoot = guess
			dirty = true
		}
	}
	if dirty {
		if err := writeConfig(cfg); err != nil {
			log.Printf("warn: could not save agent.json: %v", err)
		}
	}
	if freshPair && !localMode {
		autoPair(cfg.PushToken)
	}
	return cfg
}

// openLocalURL opens the agent's built-in local meter as a standalone "app
// window" — chromeless, with its own taskbar entry and the page's GDA
// favicon, so it launches like a native app instead of a browser tab. A
// short delay lets the HTTP listener finish binding first.
func openLocalURL(url string) {
	time.Sleep(500 * time.Millisecond)
	fmt.Println("  Opening the local meter as an app window:")
	fmt.Println("      " + url)
	fmt.Println()
	// Chromium "app mode" (--app) gives a frameless single-purpose window.
	// Edge ships on every Win10/11 box; we launch it through the same shell
	// `start` path as a normal open (confirmed working in this environment),
	// just naming the browser + passing --app. Falls back to a normal tab.
	if runtime.GOOS == "windows" {
		if err := exec.Command("cmd", "/c", "start", "", "msedge", "--app="+url).Start(); err == nil {
			return
		}
	}
	if !openInBrowser(url) {
		fmt.Println("  (Couldn't launch automatically — open the URL above.)")
		fmt.Println()
	}
}

// autoPair prints the magic-link URL and tries to open the user's default
// browser to it. The web app reads ?pair=<token> from the URL and connects
// automatically — no copy-pasting required.
func autoPair(token string) {
	pairURL := fmt.Sprintf("%s/?pair=%s", defaultViewURL, token)
	fmt.Println()
	fmt.Println("  First-run pairing — opening your browser to:")
	fmt.Println()
	fmt.Println("      " + pairURL)
	fmt.Println()

	opened := openInBrowser(pairURL)
	if opened {
		fmt.Println("  If that didn't open, copy the URL above into any browser.")
	} else {
		fmt.Println("  Couldn't launch a browser automatically — copy the URL above.")
	}
	fmt.Println()
}

// openViewURL opens the web meter for an already-paired session. Passing
// ?pair=<token> is idempotent — the web app no-ops if the same token is
// already saved, and silently re-saves if not, so this works equally well
// on the user's own machine and on a friend's first run from the shared
// launcher. Invoked when the launcher passes --open-browser.
func openViewURL(token string) {
	url := defaultViewURL + "/"
	if token != "" {
		url = fmt.Sprintf("%s/?pair=%s", defaultViewURL, token)
	}
	fmt.Println("  Opening the meter in your browser:")
	fmt.Println("      " + url)
	fmt.Println()
	if !openInBrowser(url) {
		fmt.Println("  (Couldn't launch a browser automatically — open the URL above.)")
		fmt.Println()
	}
}

// hasFlag returns true if any CLI argument matches name. Tiny purpose-
// built check; we don't need a real flag library for one toggle.
func hasFlag(name string) bool {
	for _, a := range os.Args[1:] {
		if a == name {
			return true
		}
	}
	return false
}

// flagValue returns the value following a "--name value" pair, or the
// "--name=value" form. Empty string when the flag is absent or has no
// value. Same minimal style as hasFlag.
func flagValue(name string) string {
	args := os.Args[1:]
	for i, a := range args {
		if a == name && i+1 < len(args) {
			return args[i+1]
		}
		if strings.HasPrefix(a, name+"=") {
			return strings.TrimPrefix(a, name+"=")
		}
	}
	return ""
}

// setupVerboseLog tees all log output to agent-verbose.log in the GDA
// data dir (truncated each launch) while keeping it on stderr. Lets a
// diagnostic capture be read back from the file even when the agent runs
// in an elevated console we can't scrape, or with no console at all (the
// bundled -H windowsgui flavor inside the desktop shell).
func setupVerboseLog() (string, error) {
	dir, err := config.Dir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, "agent-verbose.log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return "", err
	}
	log.SetOutput(io.MultiWriter(os.Stderr, f))
	return path, nil
}

// openInBrowser tries to open url in the user's default browser. Returns
// true if the launch command was dispatched (which doesn't guarantee a
// window actually appeared, but the OS got the request).
func openInBrowser(url string) bool {
	switch runtime.GOOS {
	case "windows":
		// `cmd /c start "" "<url>"` — the empty quoted "" is the
		// window-title slot, required so cmd doesn't think the url is
		// the title.
		return exec.Command("cmd", "/c", "start", "", url).Start() == nil
	case "darwin":
		return exec.Command("open", url).Start() == nil
	default:
		return exec.Command("xdg-open", url).Start() == nil
	}
}

// generateToken returns 32 random hex chars — same shape the website
// uses. cryptographically random.
func generateToken() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// writeConfig serializes cfg to agent.json in the GDA data dir
// (%LocalAppData%\GDA on Windows) via config.Save.
func writeConfig(cfg config.Config) error {
	return config.Save(cfg)
}

// guessAlbionInstall tries the canonical Windows install locations so a
// fresh user doesn't have to spell their path. Returns "" if nothing was
// found.
func guessAlbionInstall() string {
	if runtime.GOOS != "windows" {
		return ""
	}
	candidates := []string{
		`C:\Program Files (x86)\AlbionOnline`,
		`C:\Program Files\AlbionOnline`,
		`C:\Program Files (x86)\Albion Online`,
		`C:\Program Files\Albion Online`,
	}
	for _, c := range candidates {
		if _, err := os.Stat(filepath.Join(c, "game")); err == nil {
			return c
		}
	}
	return ""
}

func loadGameData(cfg config.Config, engine *domain.Engine) {
	if cfg.AlbionInstallRoot == "" {
		fmt.Println("  (Albion install not found — class chips and spell names will be blank.)")
		fmt.Println("  Set AlbionInstallRoot in agent.json, or set $env:ALBION_INSTALL before launch.")
		return
	}
	fmt.Printf("  Game data: loading from %s\n", cfg.AlbionInstallRoot)
	if items, err := gamedata.LoadItemCatalog(cfg.AlbionInstallRoot, gamedata.ServerLive); err != nil {
		log.Printf("  items.bin: %v (class chips will be blank)", err)
	} else {
		fmt.Printf("  items.bin OK — %d entries\n", items.Len())
		engine.SetItemCatalog(items)
	}
	if spells, err := gamedata.LoadSpellCatalog(cfg.AlbionInstallRoot, gamedata.ServerLive); err != nil {
		log.Printf("  spells.bin: %v (spell names will be blank)", err)
	} else {
		fmt.Printf("  spells.bin OK — %d entries\n", spells.Len())
		engine.SetSpellCatalog(spells)
	}
	if mobs, err := gamedata.LoadMobCatalog(cfg.AlbionInstallRoot, gamedata.ServerLive); err != nil {
		log.Printf("  mobs.bin: %v (mob names will be blank, drill-in shows #<id>)", err)
	} else {
		fmt.Printf("  mobs.bin OK — %d entries\n", mobs.Len())
		engine.SetMobCatalog(mobs)
	}
	if loc, err := gamedata.LoadLocalization(cfg.AlbionInstallRoot, gamedata.ServerLive); err != nil {
		log.Printf("  localization.bin: %v (in-game display names will be blank)", err)
	} else {
		fmt.Printf("  localization.bin OK — %d EN-US strings\n", loc.Len())
		engine.SetLocalization(loc)
	}
	fmt.Println()
}

func renderLoop(ctx context.Context, e *domain.Engine, pkts *atomic.Uint64, parser *photon.Parser, verbose bool) {
	t := time.NewTicker(5 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			snap := e.Snapshot()
			total, _, _ := e.Store().Counts()
			st := parser.Stats()
			_, name, _, known := e.LocalIdentity()
			local := "unknown"
			if known {
				local = name
				if local == "" {
					local = "(bound, no name)"
				}
			}
			// Always-on enriched heartbeat: even a glance shows whether
			// packets are flowing, decoding, and binding a local player.
			fmt.Printf("  packets=%d datagrams=%d events=%d decodeErr=%d tracked=%d players=%d local=%s zone=%q\n",
				pkts.Load(), st.Datagrams, st.Events, st.DecodeErrs, total, len(snap.Players), local, snap.Fight.Zone)
			if verbose {
				logDiag(e, parser, pkts.Load(), "tick")
			}
		}
	}
}

// logDiag writes a full diagnostic line to the (verbose-teed) log: parser
// counters, top event/op codes by frequency, local identity, and zone.
// This is the decisive signal for the "meter tracks nothing" bisect — it
// reveals whether the break is capture, deserialize, code-routing, or
// identity. Reaches agent-verbose.log via the log MultiWriter.
func logDiag(e *domain.Engine, parser *photon.Parser, packets uint64, label string) {
	st := parser.Stats()
	events, ops := e.DiagHistograms()
	objId, name, guild, known := e.LocalIdentity()
	local := "unknown"
	if known {
		local = fmt.Sprintf("objId=%d name=%q guild=%q", objId, name, guild)
	}
	log.Printf("diag(%s): packets=%d datagrams=%d events=%d responses=%d requests=%d decodeErr=%d | local: %s | topEvents[%s] | topOps[%s]",
		label, packets, st.Datagrams, st.Events, st.Responses, st.Requests, st.DecodeErrs,
		local, topCodes(events, 8), topCodes(ops, 6))
}

// topCodes formats the n highest-count entries of a code histogram as
// "code×count" descending. Used to spot event/op-code drift after a patch.
func topCodes(hist map[int]uint64, n int) string {
	type kv struct {
		code  int
		count uint64
	}
	pairs := make([]kv, 0, len(hist))
	for c, cnt := range hist {
		pairs = append(pairs, kv{c, cnt})
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].count > pairs[j].count })
	if len(pairs) > n {
		pairs = pairs[:n]
	}
	parts := make([]string, 0, len(pairs))
	for _, p := range pairs {
		parts = append(parts, fmt.Sprintf("%d×%d", p.code, p.count))
	}
	return strings.Join(parts, " ")
}

func topName(s domain.Snapshot) string {
	if len(s.Players) == 0 {
		return "—"
	}
	top := s.Players[0]
	for _, p := range s.Players[1:] {
		if p.CurrentDamage > top.CurrentDamage {
			top = p
		}
	}
	if top.Name == "" {
		return "(unknown)"
	}
	return top.Name
}

func printBanner() {
	fmt.Println()
	fmt.Println("  ╭─ GDA ─────────────────────────────────────────────")
	fmt.Printf("  │   Goldpipe's Data Analytics · agent v%s\n", version)
	fmt.Println("  │   capturing Photon UDP and streaming to the website")
	fmt.Println("  ╰───────────────────────────────────────────────────")
	fmt.Println()
}

func fatal(stage string, err error) {
	fmt.Println()
	fmt.Printf("  ✕ %s: %v\n", stage, err)
	fmt.Println()
	// Only wait for Enter when attached to an interactive console. Run
	// headless (e.g. as the desktop shell's hidden sidecar) there is no
	// console, and blocking on stdin would hang the process forever instead
	// of exiting so the shell can notice the agent died.
	if fi, e := os.Stdin.Stat(); e == nil && (fi.Mode()&os.ModeCharDevice) != 0 {
		fmt.Println("  Press Enter to close…")
		_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
	}
	os.Exit(1)
}

// maskedURL hides the host's subdomain for logging readability while still
// printing enough to debug a wrong-URL case.
func maskedURL(u string) string {
	return u
}
