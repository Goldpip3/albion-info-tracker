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

// LooterTotals is the per-player rollup the Loot panel renders. Each
// field answers a specific question; combining them into one "total" is
// the UI's job so we don't double-count silver picked up + AODP value
// of items.
//
//   Pickups          how many OtherGrabbedLoot rows did this looter
//                    produce (events, not stack sizes)
//   UnitsTotal       sum of stack quantities across non-silver picks —
//                    the "247 ore + 18 leather" feel
//   SilverPicked     direct silver pulled from corpses / chests
//   SilverValueLoot  AODP-estimated market value of the looted items
//   TopItemName      display name of this looter's most-valuable single
//                    drop this session (display name from items.bin loc)
//   TopItemValue     silver value of that top item (0 if unpriced)
//   LastPickupAt     timestamp of the most-recent entry — drives the
//                    "active 3s ago" indicator
type LooterTotals struct {
	Name            string    `json:"name"`
	IsLocal         bool      `json:"isLocal,omitempty"`
	Pickups         int       `json:"pickups"`
	UnitsTotal      int       `json:"unitsTotal"`
	SilverPicked    int64     `json:"silverPicked"`
	SilverValueLoot int64     `json:"silverValueLoot"`
	TopItemName     string    `json:"topItemName,omitempty"`
	TopItemValue    int64     `json:"topItemValue,omitempty"`
	LastPickupAt    time.Time `json:"lastPickupAt"`
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

// LooterTotalsList rolls up loot entries by looter name in a single
// pass. Pickups counts events (one OtherGrabbedLoot row = +1) so it
// matches the "47 pickups" feel the UI promises; UnitsTotal tracks the
// stack-size view for cases where it's interesting. Silver pickups and
// AODP-priced item value live in separate buckets so the UI can show
// both honestly without double-counting in any one cell.
//
// TopItemName / TopItemValue track the single highest-priced non-silver
// entry seen for this looter this session. Ties go to the most recent
// pickup so a fresh equally-good drop displaces the old one (matches
// the way a player thinks about "what's the best thing I got").
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
		t.Pickups++
		if e2.At.After(t.LastPickupAt) {
			t.LastPickupAt = e2.At
		}
		if e2.IsSilver {
			t.SilverPicked += int64(e2.Quantity)
			continue
		}
		t.UnitsTotal += e2.Quantity
		t.SilverValueLoot += e2.SilverValue
		if e2.SilverValue > 0 && e2.SilverValue >= t.TopItemValue {
			t.TopItemValue = e2.SilverValue
			if e2.DisplayName != "" {
				t.TopItemName = e2.DisplayName
			} else {
				t.TopItemName = e2.UniqueName
			}
		}
	}
	out := make([]LooterTotals, 0, len(byName))
	for _, t := range byName {
		out = append(out, *t)
	}
	sortLooterTotals(out)
	return out
}

// sortLooterTotals orders rows by total session value (silver picked
// plus AODP item value) descending, with a stable name tiebreak so
// adjacent ticks don't shuffle equal rows. Insertion sort fits the
// typical 1-20 looter count.
func sortLooterTotals(rows []LooterTotals) {
	for i := 1; i < len(rows); i++ {
		j := i
		for j > 0 && looterBefore(rows[j], rows[j-1]) {
			rows[j], rows[j-1] = rows[j-1], rows[j]
			j--
		}
	}
}

func looterBefore(a, b LooterTotals) bool {
	at := a.SilverPicked + a.SilverValueLoot
	bt := b.SilverPicked + b.SilverValueLoot
	if at != bt {
		return at > bt
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
