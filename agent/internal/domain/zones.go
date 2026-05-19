package domain

import (
	"sync"
	"time"
)

// ZoneVisit is one row of the zone history — the prettified zone name
// plus when the local player entered (and left, if known).
type ZoneVisit struct {
	Name      string    `json:"name"`
	EnteredAt time.Time `json:"enteredAt"`
	LeftAt    time.Time `json:"leftAt,omitempty"`
	DurationMs int64     `json:"durationMs,omitempty"`
}

// zoneLog keeps the last N zone visits in chronological order. Each
// successful JoinResponse for the local player closes the previous
// visit (writing LeftAt + duration) and opens a new one. Capped to
// keep the snapshot wire size sane.
const zoneLogCap = 25

// noteZoneEntry records a new zone visit. Called from handleJoinResponse
// after the zone name is prettified. Concurrent-safe.
func (e *Engine) noteZoneEntry(prettyName string, at time.Time) {
	if prettyName == "" {
		return
	}
	e.zonesMu.Lock()
	defer e.zonesMu.Unlock()
	// Close the previous visit, if any.
	if n := len(e.zones); n > 0 && e.zones[n-1].LeftAt.IsZero() {
		e.zones[n-1].LeftAt = at
		e.zones[n-1].DurationMs = at.Sub(e.zones[n-1].EnteredAt).Milliseconds()
	}
	// De-dupe: don't write back-to-back entries for the same zone
	// (Albion fires JoinResponse on respawn into the same map).
	if n := len(e.zones); n > 0 && e.zones[n-1].Name == prettyName {
		// Re-open the most recent so duration ticks correctly.
		e.zones[n-1].LeftAt = time.Time{}
		e.zones[n-1].DurationMs = 0
		return
	}
	e.zones = append(e.zones, ZoneVisit{Name: prettyName, EnteredAt: at})
	if len(e.zones) > zoneLogCap {
		e.zones = e.zones[len(e.zones)-zoneLogCap:]
	}
}

// ZoneHistory returns a defensive copy of the zone visits with current
// durations filled in for the open one.
func (e *Engine) ZoneHistory() []ZoneVisit {
	e.zonesMu.Lock()
	defer e.zonesMu.Unlock()
	if len(e.zones) == 0 {
		return nil
	}
	now := e.now()
	out := make([]ZoneVisit, len(e.zones))
	copy(out, e.zones)
	// Fill duration for the still-open last visit.
	last := &out[len(out)-1]
	if last.LeftAt.IsZero() {
		last.DurationMs = now.Sub(last.EnteredAt).Milliseconds()
	}
	return out
}

// zonesMu / zones live on the Engine. We declare them in zones.go to
// keep the engine.go diff manageable, but the actual storage is here.
var _ = sync.Mutex{} // keep import alive
