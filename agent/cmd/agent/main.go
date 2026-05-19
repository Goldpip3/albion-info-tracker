package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sort"
	"syscall"

	"github.com/Goldpip3/albion-info-tracker/agent/internal/capture"
	"github.com/Goldpip3/albion-info-tracker/agent/internal/gamecodes"
	"github.com/Goldpip3/albion-info-tracker/agent/internal/photon"
)

const version = "0.0.2"

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Printf("albion agent v%s starting", version)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	var (
		events, requests, responses uint64
	)

	parser := photon.New(photon.Handlers{
		OnEvent: func(e photon.EventData) {
			events++
			code := realCode(e.Parameters, e.Code)
			fmt.Printf("EVENT  [%3d %-30s]  %s\n",
				code, gamecodes.EventName(gamecodes.Event(code)), formatParams(e.Parameters))
		},
		OnRequest: func(r photon.OperationRequest) {
			requests++
			code := realCode(r.Parameters, r.OperationCode)
			fmt.Printf("REQ    [%3d %-30s]  %s\n",
				code, gamecodes.OpName(gamecodes.Op(code)), formatParams(r.Parameters))
		},
		OnResponse: func(r photon.OperationResponse) {
			responses++
			code := realCode(r.Parameters, r.OperationCode)
			fmt.Printf("RESP   [%3d %-30s]  rc=%d msg=%q %s\n",
				code, gamecodes.OpName(gamecodes.Op(code)),
				r.ReturnCode, r.DebugMessage, formatParams(r.Parameters))
		},
	})

	sink := capture.SinkFunc(func(pkt capture.Packet) {
		parser.Receive(pkt.Payload)
	})

	if err := capture.Run(ctx, sink); err != nil && err != context.Canceled {
		log.Fatalf("capture: %v", err)
	}
	log.Printf("stopped: %d events, %d requests, %d responses", events, requests, responses)
}

// realCode returns the authoritative Photon code. For events, the
// application-level code lives at parameter 252 when it can't fit in a single
// byte; for operations it's at 253. Falls back to the byte after messageType.
func realCode(p map[byte]any, fallback byte) int {
	for _, k := range []byte{252, 253} {
		if v, ok := p[k]; ok {
			if n, ok := toInt(v); ok {
				return n
			}
		}
	}
	return int(fallback)
}

func toInt(v any) (int, bool) {
	switch x := v.(type) {
	case byte:
		return int(x), true
	case int16:
		return int(x), true
	case int32:
		return int(x), true
	case int64:
		return int(x), true
	case uint16:
		return int(x), true
	case uint32:
		return int(x), true
	case uint64:
		return int(x), true
	default:
		return 0, false
	}
}

// formatParams renders a parameter table in deterministic key order, with a
// short preview of each value. Long byte arrays are abbreviated.
func formatParams(p map[byte]any) string {
	if len(p) == 0 {
		return "{}"
	}
	keys := make([]int, 0, len(p))
	for k := range p {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)

	out := "{"
	for i, k := range keys {
		if i > 0 {
			out += " "
		}
		out += fmt.Sprintf("%d:%s", k, formatValue(p[byte(k)]))
	}
	out += "}"
	return out
}

func formatValue(v any) string {
	switch x := v.(type) {
	case nil:
		return "null"
	case bool:
		return fmt.Sprintf("%v", x)
	case byte:
		return fmt.Sprintf("%d", x)
	case int16, int32, int64, uint16, uint32, uint64:
		return fmt.Sprintf("%d", x)
	case float32, float64:
		return fmt.Sprintf("%g", x)
	case string:
		if len(x) > 32 {
			return fmt.Sprintf("%q…", x[:32])
		}
		return fmt.Sprintf("%q", x)
	case []byte:
		if len(x) > 16 {
			return fmt.Sprintf("[%d bytes]", len(x))
		}
		return fmt.Sprintf("%v", x)
	case photon.CustomType:
		return fmt.Sprintf("custom(%d, %d bytes)", x.TypeCode, len(x.Data))
	case []any:
		return fmt.Sprintf("array[%d]", len(x))
	case map[any]any:
		return fmt.Sprintf("dict[%d]", len(x))
	default:
		return fmt.Sprintf("%v", v)
	}
}
