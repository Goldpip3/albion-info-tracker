package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Goldpip3/albion-info-tracker/agent/internal/capture"
)

const version = "0.0.1"

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Printf("albion agent v%s starting", version)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	var count uint64
	sink := capture.SinkFunc(func(p capture.Packet) {
		count++
		preview := p.Payload
		if len(preview) > 32 {
			preview = preview[:32]
		}
		fmt.Printf("#%06d  %5d → %5d  len=%4d  %s\n",
			count, p.SrcPort, p.DstPort, len(p.Payload),
			hex.EncodeToString(preview),
		)
	})

	if err := capture.Run(ctx, sink); err != nil && err != context.Canceled {
		log.Fatalf("capture: %v", err)
	}
	log.Printf("stopped after %d packets", count)
}
