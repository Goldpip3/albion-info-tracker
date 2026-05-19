package push

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"

	"github.com/Goldpip3/albion-info-tracker/agent/internal/domain"
)

// Client is a single-connection WebSocket sender with auto-reconnect. It
// keeps only the latest snapshot in flight — older snapshots are silently
// dropped, since each snapshot is a complete state and supersedes the prior.
type Client struct {
	URL          string
	Token        string
	AgentVersion string

	// SendInterval is how often Snapshot() is read and pushed. Defaults to
	// 500ms if zero.
	SendInterval time.Duration

	// Snapshot is called from the push goroutine to fetch the current state.
	Snapshot func() domain.Snapshot

	// LocalGuid is included in the Hello message when set.
	LocalGuid func() string

	// OnCommand is invoked when the backend forwards a viewer command
	// (e.g. {type:"command", action:"resetSession"}). Optional.
	OnCommand func(action string)

	// counters
	connects  atomic.Uint64
	sent      atomic.Uint64
	errors    atomic.Uint64
}

// Run blocks until ctx is cancelled, looping over connect → send → disconnect
// with exponential backoff on failure. Returns the last connect error only
// if it never connected once.
func (c *Client) Run(ctx context.Context) error {
	if c.SendInterval == 0 {
		c.SendInterval = 500 * time.Millisecond
	}

	backoff := newBackoff(time.Second, 30*time.Second)
	var firstErr error

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		err := c.connectAndPump(ctx)
		if err == nil || errors.Is(err, context.Canceled) {
			return nil
		}
		if firstErr == nil && c.connects.Load() == 0 {
			firstErr = err
		}
		c.errors.Add(1)

		wait := backoff.Next()
		log.Printf("push: %v; reconnecting in %s", err, wait)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
	}
}

func (c *Client) connectAndPump(ctx context.Context) error {
	dialCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	header := http.Header{}
	if c.Token != "" {
		header.Set("Authorization", "Bearer "+c.Token)
	}

	conn, _, err := websocket.Dial(dialCtx, c.URL, &websocket.DialOptions{
		HTTPHeader: header,
	})
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	c.connects.Add(1)
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Send Hello first.
	hello := Envelope{
		V:    ProtocolVersion,
		Type: "hello",
		TS:   time.Now().UnixMilli(),
		Hello: &HelloMessage{
			AgentVersion: c.AgentVersion,
			LocalGuid:    safeGuid(c.LocalGuid),
		},
	}
	if err := writeJSON(ctx, conn, hello); err != nil {
		return fmt.Errorf("send hello: %w", err)
	}

	// Start a goroutine that reads inbound messages so command envelopes
	// from viewers (forwarded by the Worker) can be dispatched. Errors
	// terminate the read loop but don't kill the writer — the writer's
	// next send will hit the same broken socket and reconnect.
	readCtx, readCancel := context.WithCancel(ctx)
	defer readCancel()
	go c.readLoop(readCtx, conn)

	// Periodic snapshot pump.
	t := time.NewTicker(c.SendInterval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
			if c.Snapshot == nil {
				continue
			}
			snap := c.Snapshot()
			env := Envelope{
				V:    ProtocolVersion,
				Type: "snapshot",
				TS:   time.Now().UnixMilli(),
				Snap: &snap,
			}
			if err := writeJSON(ctx, conn, env); err != nil {
				return fmt.Errorf("send snapshot: %w", err)
			}
			c.sent.Add(1)
		}
	}
}

func safeGuid(f func() string) string {
	if f == nil {
		return ""
	}
	return f()
}

// readLoop pulls inbound messages off the connection and routes command
// envelopes to c.OnCommand. Unrecognised types are silently dropped.
func (c *Client) readLoop(ctx context.Context, conn *websocket.Conn) {
	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		var env Envelope
		if jsonErr := json.Unmarshal(data, &env); jsonErr != nil {
			continue
		}
		if env.V != ProtocolVersion {
			continue
		}
		if env.Type == "command" && env.Command != nil && c.OnCommand != nil {
			c.OnCommand(env.Command.Action)
		}
	}
}

func writeJSON(ctx context.Context, conn *websocket.Conn, v any) error {
	wctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return conn.Write(wctx, websocket.MessageText, b)
}

// Stats reports cumulative counters since Run started. Safe to call from any
// goroutine.
func (c *Client) Stats() (connects, sent, errs uint64) {
	return c.connects.Load(), c.sent.Load(), c.errors.Load()
}

type backoff struct {
	base, max time.Duration
	current   time.Duration
}

func newBackoff(base, max time.Duration) *backoff {
	return &backoff{base: base, max: max, current: base}
}

func (b *backoff) Next() time.Duration {
	d := b.current
	// Cap.
	if d > b.max {
		d = b.max
	}
	// Add ±20% jitter so a fleet of agents doesn't herd the server.
	jitter := time.Duration(rand.Int64N(int64(d) / 5))
	if rand.IntN(2) == 0 {
		d += jitter
	} else {
		d -= jitter
	}
	// Double for next time.
	b.current *= 2
	if b.current > b.max {
		b.current = b.max
	}
	return d
}

// Reset zeroes the backoff after a successful round-trip.
func (b *backoff) Reset() { b.current = b.base }
