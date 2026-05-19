package domain

import (
	"sync"
	"time"
)

// ActivityEvent is one entry in the kill-feed / activity log. Sent inside
// the periodic snapshot rather than as a separate message — the buffer is
// small and the frontend wants the latest N anyway.
type ActivityEvent struct {
	TS           int64  `json:"ts"`                     // unix millis
	Kind         string `json:"kind"`                   // "hit" | "heal" | "death"
	Actor        string `json:"actor,omitempty"`        // attacker / healer / "—"
	ActorIsLocal bool   `json:"actorIsLocal,omitempty"`
	Target       string `json:"target,omitempty"`       // victim
	Amount       int64  `json:"amount,omitempty"`       // damage / heal value
	SpellName    string `json:"spellName,omitempty"`    // resolved uniquename when known
	BigHit       bool   `json:"bigHit,omitempty"`       // > bigHitThreshold for damage
}

// bigHitThreshold marks a damage event for the "Big hits" tab. Tuned to
// roughly match the design's example feed where a 22K hit qualifies.
const bigHitThreshold = 5_000

// eventBufferSize caps the ring; older events are dropped on overflow.
const eventBufferSize = 96

// eventBuffer is a thread-safe ring of recent ActivityEvents.
type eventBuffer struct {
	mu   sync.Mutex
	ring []ActivityEvent
	head int // index of the next write position
	size int // number of valid entries (≤ eventBufferSize)
}

func newEventBuffer() *eventBuffer {
	return &eventBuffer{ring: make([]ActivityEvent, eventBufferSize)}
}

func (b *eventBuffer) Push(ev ActivityEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.ring[b.head] = ev
	b.head = (b.head + 1) % eventBufferSize
	if b.size < eventBufferSize {
		b.size++
	}
}

// SnapshotLatest returns the n most-recent events, newest first.
func (b *eventBuffer) SnapshotLatest(n int) []ActivityEvent {
	b.mu.Lock()
	defer b.mu.Unlock()
	if n > b.size {
		n = b.size
	}
	out := make([]ActivityEvent, n)
	// Walk backwards from head.
	idx := b.head
	for i := 0; i < n; i++ {
		idx = (idx - 1 + eventBufferSize) % eventBufferSize
		out[i] = b.ring[idx]
	}
	return out
}

// recordHit pushes a damage event. amount must be positive.
func (e *Engine) recordHit(actor, target *Entity, amount int64, spellIndex int, now time.Time) {
	if e.events == nil || amount <= 0 {
		return
	}
	e.events.Push(ActivityEvent{
		TS:           now.UnixMilli(),
		Kind:         "hit",
		Actor:        entityName(actor),
		ActorIsLocal: actor != nil && actor.IsLocal,
		Target:       entityName(target),
		Amount:       amount,
		SpellName:    e.spellName(spellIndex),
		BigHit:       amount >= bigHitThreshold,
	})
}

// recordHealEvent pushes a healing event.
func (e *Engine) recordHealEvent(actor, target *Entity, amount int64, spellIndex int, now time.Time) {
	if e.events == nil || amount <= 0 {
		return
	}
	e.events.Push(ActivityEvent{
		TS:           now.UnixMilli(),
		Kind:         "heal",
		Actor:        entityName(actor),
		ActorIsLocal: actor != nil && actor.IsLocal,
		Target:       entityName(target),
		Amount:       amount,
		SpellName:    e.spellName(spellIndex),
	})
}

// recordDeathEvent pushes a death event.
func (e *Engine) recordDeathEvent(victim, killer string, now time.Time) {
	if e.events == nil {
		return
	}
	e.events.Push(ActivityEvent{
		TS:     now.UnixMilli(),
		Kind:   "death",
		Actor:  killer,
		Target: victim,
	})
}

func entityName(e *Entity) string {
	if e == nil {
		return ""
	}
	if e.Name != "" {
		return e.Name
	}
	return ""
}
