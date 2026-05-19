package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sort"
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

const version = "0.0.4"

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Printf("albion agent v%s starting", version)

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	engine := domain.NewEngine()
	if cfg.AlbionInstallRoot != "" {
		if items, err := gamedata.LoadItemCatalog(cfg.AlbionInstallRoot, gamedata.ServerLive); err != nil {
			log.Printf("items.bin: %v (class chips will be blank)", err)
		} else {
			engine.SetItemCatalog(items)
			log.Printf("items.bin: %d entries loaded for weapon classification", items.Len())
		}
	} else {
		log.Print("AlbionInstallRoot not configured — class chips will be blank. Set ALBION_INSTALL or albionInstallRoot in agent.json.")
	}
	parser := photon.New(engine.Handlers())

	var packetsSeen atomic.Uint64
	sink := capture.SinkFunc(func(pkt capture.Packet) {
		packetsSeen.Add(1)
		parser.Receive(pkt.Payload)
	})

	go renderLoop(ctx, engine, &packetsSeen)

	// Push to remote backend if configured. Runs concurrently with capture.
	if cfg.PushURL != "" {
		client := &push.Client{
			URL:          cfg.PushURL,
			Token:        cfg.PushToken,
			AgentVersion: version,
			Snapshot:     engine.Snapshot,
		}
		log.Printf("push: connecting to %s", cfg.PushURL)
		go func() {
			if err := client.Run(ctx); err != nil && err != context.Canceled {
				log.Printf("push: stopped: %v", err)
			}
		}()
	} else {
		log.Printf("push: no PushURL configured — running in stdout-only mode")
	}

	if err := capture.Run(ctx, sink); err != nil && err != context.Canceled {
		log.Fatalf("capture: %v", err)
	}
	printSnapshot(engine.Snapshot(), true)
	log.Print("stopped")
}

func renderLoop(ctx context.Context, e *domain.Engine, pkts *atomic.Uint64) {
	t := time.NewTicker(2 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			snap := e.Snapshot()
			total, bound, _ := e.Store().Counts()
			if len(snap.Players) == 0 {
				log.Printf("heartbeat  packets=%d  tracked=%d  bound=%d  party=0", pkts.Load(), total, bound)
				continue
			}
			printSnapshot(snap, false)
		}
	}
}

func printSnapshot(s domain.Snapshot, final bool) {
	if len(s.Players) == 0 {
		if final {
			fmt.Println("--- no party members tracked ---")
		}
		return
	}
	sort.Slice(s.Players, func(i, j int) bool {
		return s.Players[i].CurrentDamage > s.Players[j].CurrentDamage
	})

	header := "--- damage meter --- "
	if final {
		header = "=== final snapshot ==="
	}
	fmt.Printf("%s  %s\n", header, s.GeneratedAt.Format("15:04:05"))
	fmt.Printf("  %-20s %12s %10s %12s %10s %12s %12s\n",
		"NAME", "DMG CUR", "DPS CUR", "DMG OVR", "DPS OVR", "TAKEN CUR", "HEAL CUR")
	for _, p := range s.Players {
		name := p.Name
		if name == "" {
			name = p.UserGuid[:8] + "…"
		}
		marker := "  "
		if p.IsLocal {
			marker = "* "
		}
		fmt.Printf("%s%-20s %12d %10.0f %12d %10.0f %12d %12d\n",
			marker, name,
			p.CurrentDamage, p.CurrentDPS,
			p.OverallDamage, p.OverallDPS,
			p.CurrentTaken, p.CurrentHeal,
		)
	}
	fmt.Println()
}
