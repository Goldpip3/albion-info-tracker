package domain

import (
	"strings"
	"time"
)

// DungeonRun is the run-scoped session — a "what did I get for THIS
// dungeon" view layered on top of the broader session counters.
// Opens on JoinResponse into a dungeon-pattern zone, closes on the
// next JoinResponse out.
type DungeonRun struct {
	Id          string    `json:"id"`           // unix-millis of EnteredAt — stable across the run
	Zone        string    `json:"zone"`         // pretty-printed entrance zone
	Type        string    `json:"type"`         // "solo" | "group" | "avalonian" | "mists" | "hellgate" | "unknown"
	EnteredAt   time.Time `json:"enteredAt"`
	EndedAt     time.Time `json:"endedAt,omitempty"`
	DurationMs  int64     `json:"durationMs,omitempty"`
	// Deltas baselined when the run opens; the snapshot exposes the
	// difference between current and baseline so the run-scoped counts
	// always reflect "what happened since I entered."
	FameBaseline   int64 `json:"-"`
	SilverBaseline int64 `json:"-"`
	RespecBaseline int64 `json:"-"`
	MightBaseline  int64 `json:"-"`
	DeathsBaseline int   `json:"-"`
	// Live deltas (current - baseline). Refreshed by the snapshot
	// builder via DungeonRun.refresh.
	FameGained   int64 `json:"fameGained"`
	SilverGained int64 `json:"silverGained"`
	RespecGained int64 `json:"respecGained"`
	MightGained  int64 `json:"mightGained"`
	DeathsInRun  int   `json:"deathsInRun,omitempty"`
}

// classifyDungeon inspects an Albion map-index token (the raw param 8
// value, like "@RANDOMDUNGEON_SOLO_T6_AVALON" or "MISTS_HELLISH_HUB")
// and returns ("solo"/"group"/"avalonian"/"mists"/"hellgate"/"") +
// whether this looks like an actual dungeon entrance worth opening a
// run for. Open worlds / cities / hideouts return ("", false).
func classifyDungeon(rawMapIndex string) (string, bool) {
	u := strings.ToUpper(rawMapIndex)
	switch {
	case strings.Contains(u, "@HELLGATE"):
		return "hellgate", true
	case strings.Contains(u, "AVALON_ROAD"), strings.Contains(u, "@AVALONIAN"):
		return "avalonian", true
	case strings.Contains(u, "@RANDOMDUNGEON_SOLO"), strings.Contains(u, "SOLODUNGEON"):
		return "solo", true
	case strings.Contains(u, "@RANDOMDUNGEON_GROUP"), strings.Contains(u, "GROUPDUNGEON"):
		return "group", true
	case strings.Contains(u, "MISTS_"), strings.Contains(u, "@MISTS"):
		return "mists", true
	case strings.Contains(u, "@CORRUPTED"):
		return "solo", true
	}
	return "", false
}

// openDungeon stamps a new DungeonRun based on the current session
// counters as baseline. Replaces any in-flight run silently.
func (e *Engine) openDungeon(zone, kind string, at time.Time) {
	e.dungeonMu.Lock()
	defer e.dungeonMu.Unlock()
	// Close any prior run first so its endTime reflects this moment.
	if e.dungeon != nil && e.dungeon.EndedAt.IsZero() {
		e.dungeon.EndedAt = at
		e.dungeon.DurationMs = at.Sub(e.dungeon.EnteredAt).Milliseconds()
	}
	e.sessionMu.Lock()
	run := &DungeonRun{
		Id:             ts(at),
		Zone:           zone,
		Type:           kind,
		EnteredAt:      at,
		FameBaseline:   e.session.FameTotal,
		SilverBaseline: e.session.SilverTotal,
		RespecBaseline: e.session.RespecTotal,
		MightBaseline:  e.session.MightTotal,
		DeathsBaseline: e.session.DeathsTotal,
	}
	e.sessionMu.Unlock()
	e.dungeon = run
}

// closeDungeon marks the current run as ended. Snapshot continues to
// expose it (with its frozen deltas) until a new run opens or the user
// resets the session.
func (e *Engine) closeDungeon(at time.Time) {
	e.dungeonMu.Lock()
	defer e.dungeonMu.Unlock()
	if e.dungeon == nil || !e.dungeon.EndedAt.IsZero() {
		return
	}
	e.dungeon.EndedAt = at
	e.dungeon.DurationMs = at.Sub(e.dungeon.EnteredAt).Milliseconds()
}

// CurrentDungeon returns a snapshot-friendly copy of the active run,
// with live deltas computed against the current session counters.
// Returns nil when there's no run to surface.
func (e *Engine) CurrentDungeon() *DungeonRun {
	e.dungeonMu.Lock()
	defer e.dungeonMu.Unlock()
	if e.dungeon == nil {
		return nil
	}
	out := *e.dungeon
	e.sessionMu.Lock()
	out.FameGained = e.session.FameTotal - out.FameBaseline
	out.SilverGained = e.session.SilverTotal - out.SilverBaseline
	out.RespecGained = e.session.RespecTotal - out.RespecBaseline
	out.MightGained = e.session.MightTotal - out.MightBaseline
	out.DeathsInRun = e.session.DeathsTotal - out.DeathsBaseline
	e.sessionMu.Unlock()
	if out.EndedAt.IsZero() {
		out.DurationMs = e.now().Sub(out.EnteredAt).Milliseconds()
	}
	return &out
}

func ts(t time.Time) string {
	return t.UTC().Format("20060102T150405")
}
