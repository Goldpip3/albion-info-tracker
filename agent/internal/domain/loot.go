package domain

import (
	"sync"
	"time"

	"github.com/Goldpip3/albion-info-tracker/agent/internal/aodp"
)

// LootEntry is one row of the loot log — somebody (you or a friend)
// picked up an item or silver pile. Silver entries have IsSilver=true
// and the Quantity field holds the raw silver units (FixPoint internal
// when needed; in practice silver loot events ship integer copper).
type LootEntry struct {
	At           time.Time `json:"at"`
	Looter       string    `json:"looter"`
	LooterIsLocal bool     `json:"looterIsLocal,omitempty"`
	LootedFrom   string    `json:"lootedFrom,omitempty"`
	ItemIndex    int       `json:"itemIndex,omitempty"`
	UniqueName   string    `json:"uniqueName,omitempty"`
	DisplayName  string    `json:"displayName,omitempty"`
	Quantity     int       `json:"quantity"`
	IsSilver     bool      `json:"isSilver,omitempty"`
	SilverValue  int64     `json:"silverValue,omitempty"` // AODP-estimated, 0 if unknown
	Zone         string    `json:"zone,omitempty"`
	DungeonId    string    `json:"dungeonId,omitempty"`
}

// LooterTotals is the per-player rollup the Loot panel renders: how
// many items, what's the estimated silver value, was that a friend or
// you. Computed on demand from the loot log.
type LooterTotals struct {
	Name        string `json:"name"`
	IsLocal     bool   `json:"isLocal,omitempty"`
	ItemCount   int    `json:"itemCount"`
	SilverTotal int64  `json:"silverTotal"`
	ValueTotal  int64  `json:"valueTotal"`
}

const lootLogCap = 500

// lootLog state lives on the Engine. Keeping the storage in this file
// to keep the engine.go diff small.

// noteLoot appends a loot entry to the ring buffer and triggers an
// AODP price fetch when a non-silver item lands. Caller holds no locks.
func (e *Engine) noteLoot(entry LootEntry) {
	e.lootMu.Lock()
	e.lootLog = append(e.lootLog, entry)
	if len(e.lootLog) > lootLogCap {
		e.lootLog = e.lootLog[len(e.lootLog)-lootLogCap:]
	}
	e.lootMu.Unlock()
	if e.prices != nil && !entry.IsSilver && entry.UniqueName != "" {
		e.prices.QueueFetch(entry.UniqueName)
	}
}

// LootLog returns a defensive copy of the loot ring buffer with the
// latest AODP price estimates applied for each non-silver entry.
func (e *Engine) LootLog() []LootEntry {
	e.lootMu.Lock()
	defer e.lootMu.Unlock()
	if len(e.lootLog) == 0 {
		return nil
	}
	out := make([]LootEntry, len(e.lootLog))
	copy(out, e.lootLog)
	if e.prices != nil {
		for i := range out {
			if out[i].IsSilver || out[i].UniqueName == "" {
				continue
			}
			if p, ok := e.prices.Lookup(out[i].UniqueName); ok && p.Silver > 0 {
				out[i].SilverValue = p.Silver * int64(out[i].Quantity)
			}
		}
	}
	return out
}

// LooterTotalsList rolls up loot entries by looter name. Used by the
// "who farmed how much" view. Local player floats to the top.
func (e *Engine) LooterTotalsList() []LooterTotals {
	entries := e.LootLog()
	if len(entries) == 0 {
		return nil
	}
	byName := make(map[string]*LooterTotals, 8)
	for _, e2 := range entries {
		t, ok := byName[e2.Looter]
		if !ok {
			t = &LooterTotals{Name: e2.Looter, IsLocal: e2.LooterIsLocal}
			byName[e2.Looter] = t
		}
		if e2.IsSilver {
			t.SilverTotal += int64(e2.Quantity)
			t.ValueTotal += int64(e2.Quantity)
		} else {
			t.ItemCount += e2.Quantity
			t.ValueTotal += e2.SilverValue
		}
	}
	out := make([]LooterTotals, 0, len(byName))
	for _, t := range byName {
		out = append(out, *t)
	}
	// Local first; otherwise highest value descending.
	sortLooterTotals(out)
	return out
}

func sortLooterTotals(rows []LooterTotals) {
	// Insertion sort is fine — there are typically 1-20 looters.
	for i := 1; i < len(rows); i++ {
		j := i
		for j > 0 && looterBefore(rows[j], rows[j-1]) {
			rows[j], rows[j-1] = rows[j-1], rows[j]
			j--
		}
	}
}

func looterBefore(a, b LooterTotals) bool {
	if a.IsLocal != b.IsLocal {
		return a.IsLocal
	}
	if a.ValueTotal != b.ValueTotal {
		return a.ValueTotal > b.ValueTotal
	}
	return a.Name < b.Name
}

// resetLootOnNewSession clears the loot log when a session resets.
func (e *Engine) resetLootOnNewSession() {
	e.lootMu.Lock()
	e.lootLog = nil
	e.lootMu.Unlock()
}

// Declare prices type alias so the engine can reference *aodp.Client
// without an import cycle / extra cognitive load when reading engine.go.
type _ = aodp.Client

// Re-export for engine.go's struct field declaration; placed here to
// keep the import locality clear.
var _ sync.Mutex
