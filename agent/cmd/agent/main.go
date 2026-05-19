package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/Goldpip3/albion-info-tracker/agent/internal/capture"
	"github.com/Goldpip3/albion-info-tracker/agent/internal/config"
	"github.com/Goldpip3/albion-info-tracker/agent/internal/domain"
	"github.com/Goldpip3/albion-info-tracker/agent/internal/gamedata"
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

	cfg, err := config.Load()
	if err != nil {
		fatal("agent.json", err)
	}

	// First-run wizard: if pushToken or pushUrl is missing, walk the user
	// through pairing without making them edit JSON by hand.
	cfg = ensureConfigured(cfg)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	engine := domain.NewEngine()
	loadGameData(cfg, engine)

	parser := photon.New(engine.Handlers())

	var packetsSeen atomic.Uint64
	sink := capture.SinkFunc(func(pkt capture.Packet) {
		packetsSeen.Add(1)
		parser.Receive(pkt.Payload)
	})

	go renderLoop(ctx, engine, &packetsSeen)

	if cfg.PushURL != "" {
		client := &push.Client{
			URL:          cfg.PushURL,
			Token:        cfg.PushToken,
			AgentVersion: version,
			Snapshot:     engine.Snapshot,
		}
		fmt.Printf("\n  Streaming to %s\n", maskedURL(cfg.PushURL))
		fmt.Printf("  View at      %s\n\n", defaultViewURL)
		go func() {
			if err := client.Run(ctx); err != nil && err != context.Canceled {
				log.Printf("push: stopped: %v", err)
			}
		}()
	} else {
		log.Print("push: no PushURL configured — running in stdout-only mode")
	}

	if err := capture.Run(ctx, sink); err != nil && err != context.Canceled {
		fatal("capture", err)
	}
	fmt.Println("\nStopped.")
}

// ensureConfigured fills in any missing critical config and re-writes
// agent.json next to the exe so future launches skip the prompt. First-
// run pairing is handled by autoPair — the agent generates a token and
// opens the default browser to the magic-link URL so the user never has
// to type or copy a token.
func ensureConfigured(cfg config.Config) config.Config {
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
	if freshPair {
		autoPair(cfg.PushToken)
	}
	return cfg
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

// writeConfig serializes cfg to agent.json next to the executable.
func writeConfig(cfg config.Config) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	path := filepath.Join(filepath.Dir(exe), "agent.json")
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
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
		return
	}
	if items, err := gamedata.LoadItemCatalog(cfg.AlbionInstallRoot, gamedata.ServerLive); err != nil {
		log.Printf("items.bin: %v", err)
	} else {
		engine.SetItemCatalog(items)
	}
	if spells, err := gamedata.LoadSpellCatalog(cfg.AlbionInstallRoot, gamedata.ServerLive); err != nil {
		log.Printf("spells.bin: %v", err)
	} else {
		engine.SetSpellCatalog(spells)
	}
}

func renderLoop(ctx context.Context, e *domain.Engine, pkts *atomic.Uint64) {
	t := time.NewTicker(5 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			snap := e.Snapshot()
			total, _, _ := e.Store().Counts()
			if len(snap.Players) == 0 {
				fmt.Printf("  waiting for combat… packets=%d tracked=%d\n", pkts.Load(), total)
				continue
			}
			fmt.Printf("  fight %02d · %d player(s) · top: %s\n",
				snap.Fight.Number, len(snap.Players), topName(snap))
		}
	}
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
	fmt.Println("  ╭─ SKIRMISH ────────────────────────────────────────")
	fmt.Printf("  │   agent v%s\n", version)
	fmt.Println("  │   capturing Photon UDP and streaming to the website")
	fmt.Println("  ╰───────────────────────────────────────────────────")
	fmt.Println()
}

func fatal(stage string, err error) {
	fmt.Println()
	fmt.Printf("  ✕ %s: %v\n", stage, err)
	fmt.Println()
	fmt.Println("  Press Enter to close…")
	bufio.NewReader(os.Stdin).ReadString('\n')
	os.Exit(1)
}

// maskedURL hides the host's subdomain for logging readability while still
// printing enough to debug a wrong-URL case.
func maskedURL(u string) string {
	return u
}
