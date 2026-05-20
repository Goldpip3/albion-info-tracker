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
	// MobSilver is the subset of this looter's silver that came from mob
	// kills. Only populated for the local row (from the session's
	// authoritative MobSilverTotal); party/guild rows leave it zero.
	MobSilver       int64     `json:"mobSilver,omitempty"`
	TopItemName     string    `json:"topItemName,omitempty"`
	TopItemValue    int64     `json:"topItemValue,omitempty"`
	// RecentItemName is the display name of the most-recent non-silver
	// pickup for this looter — used by the UI when TopItemName is empty
	// (every priced lookup missed) so the row still says what was
	// looted rather than rendering a bare em-dash.
	RecentItemName string `json:"recentItemName,omitempty"`
	// OnlySilver is true when every entry for this looter was a silver
	// pile. Lets the UI render "silver only" copy honestly instead of
	// implying there's an item-side number to look at.
	OnlySilver   bool      `json:"onlySilver,omitempty"`
	LastPickupAt time.Time `json:"lastPickupAt"`
	// Source documents why this looter passes the membership filter:
	//   "local"  — the agent's local player
	//   "party"  — Albion told us they're in the party via PartyJoined
	//   "guild"  — same Guild tag as the local player
	//   "friend" — on the agent.json::alwaysIncludeNames allowlist
	// Empty means we couldn't classify, which the UI hides.
	Source string `json:"source,omitempty"`
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
// latest AODP price estimates applied. Entries are filtered to the
// same membership scope as the meter (local + same-guild + party +
// alwaysIncludeNames), so the loot panel doesn't drag in every pub
// looting in render range. ALBION_AGENT_SHOW_ALL bypasses the filter.
func (e *Engine) LootLog() []LootEntry {
	var allowed map[string]struct{}
	if !showAll && e.LootFilterMode() != "everyone" {
		allowed = e.allowedLooters()
	}

	e.lootMu.Lock()
	defer e.lootMu.Unlock()
	if len(e.lootLog) == 0 {
		return nil
	}
	out := make([]LootEntry, 0, len(e.lootLog))
	for _, entry := range e.lootLog {
		if allowed != nil {
			if _, ok := allowed[entry.Looter]; !ok {
				continue
			}
		}
		out = append(out, entry)
	}
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

// allowedLooters returns the set of player names the loot views should
// surface. Mirrors the meter's filter so the two stay consistent:
//   - the local player
//   - everyone Albion told us is in the party (PartyJoined succeeded)
//   - same-guild players seen in this zone (skipped when the user has
//     toggled "Include guildies" off in Settings — LootFilterMode
//     reads "party" in that mode)
//   - explicit allowlist from agent.json::alwaysIncludeNames
//
// Returns nil when the local entity isn't known yet — caller treats
// nil as "no filter" and falls through to whatever default they pick.
func (e *Engine) allowedLooters() map[string]struct{} {
	out := make(map[string]struct{}, 16)
	e.alwaysIncludeMu.RLock()
	for n := range e.alwaysInclude {
		out[n] = struct{}{}
	}
	e.alwaysIncludeMu.RUnlock()

	includeGuild := e.LootFilterMode() != "party"

	e.store.mu.RLock()
	defer e.store.mu.RUnlock()
	var localGuild string
	if !e.store.localGuid.IsZero() {
		if local := e.store.byGuid[e.store.localGuid]; local != nil {
			if local.Name != "" {
				out[local.Name] = struct{}{}
			}
			localGuild = local.Guild
		}
	}
	for _, ent := range e.store.byGuid {
		if ent.Name == "" {
			continue
		}
		if ent.IsInParty {
			out[ent.Name] = struct{}{}
			continue
		}
		if includeGuild && localGuild != "" && ent.Guild == localGuild {
			out[ent.Name] = struct{}{}
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
	sources := e.looterSources()
	byName := make(map[string]*LooterTotals, 8)
	// Track which looters have had at least one non-silver pickup so
	// OnlySilver can flip false on the first item we see.
	hasItem := make(map[string]bool, 8)
	for _, e2 := range entries {
		t, ok := byName[e2.Looter]
		if !ok {
			t = &LooterTotals{
				Name:       e2.Looter,
				IsLocal:    e2.LooterIsLocal,
				OnlySilver: true,
				Source:     sources[e2.Looter],
			}
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
		hasItem[e2.Looter] = true
		t.OnlySilver = false
		t.UnitsTotal += e2.Quantity
		t.SilverValueLoot += e2.SilverValue
		// Track the most-recent non-silver pickup name regardless of
		// price so the UI has something to display when no item priced.
		// "Most recent" walks the log forward so we end up with the
		// latest entry — the log itself is append-only chronological.
		nm := e2.DisplayName
		if nm == "" {
			nm = e2.UniqueName
		}
		if nm != "" {
			t.RecentItemName = nm
		}
		if e2.SilverValue > 0 && e2.SilverValue >= t.TopItemValue {
			t.TopItemValue = e2.SilverValue
			t.TopItemName = nm
		}
	}

	// Local player's silver is authoritative from the running session
	// total, not the loot ring buffer: SilverTotal climbs forever while
	// lootLog caps at 500 entries (a long farm would undercount). It
	// also captures TakeSilver chest/dungeon silver that never produces
	// an OtherGrabbedLoot row. Override (not add) the local row, and
	// surface the mob-only subtotal for the Loot-tab breakout.
	if local := e.store.localGuidEntity(); local != nil && local.Name != "" {
		e.sessionMu.Lock()
		silverTotal, mobSilver := e.session.SilverTotal, e.session.MobSilverTotal
		e.sessionMu.Unlock()
		if silverTotal > 0 || mobSilver > 0 {
			t, ok := byName[local.Name]
			if !ok {
				t = &LooterTotals{Name: local.Name, IsLocal: true, Source: "local"}
				byName[local.Name] = t
			}
			t.SilverPicked = silverTotal
			t.MobSilver = mobSilver
		}
	}

	out := make([]LooterTotals, 0, len(byName))
	for _, t := range byName {
		out = append(out, *t)
	}
	sortLooterTotals(out)
	return out
}

// looterSources mirrors allowedLooters but returns the reason each
// name is allowed instead of just the set membership. Used by the
// rollup to tag each LooterTotals row with a PARTY / GUILD / FRIEND
// badge so the UI can show why a player is on screen.
func (e *Engine) looterSources() map[string]string {
	out := make(map[string]string, 16)

	e.alwaysIncludeMu.RLock()
	for n := range e.alwaysInclude {
		out[n] = "friend"
	}
	e.alwaysIncludeMu.RUnlock()

	includeGuild := e.LootFilterMode() != "party"

	e.store.mu.RLock()
	defer e.store.mu.RUnlock()
	var localGuild string
	if !e.store.localGuid.IsZero() {
		if local := e.store.byGuid[e.store.localGuid]; local != nil {
			if local.Name != "" {
				out[local.Name] = "local"
			}
			localGuild = local.Guild
		}
	}
	for _, ent := range e.store.byGuid {
		if ent.Name == "" {
			continue
		}
		if _, ok := out[ent.Name]; ok && out[ent.Name] == "local" {
			continue
		}
		if ent.IsInParty {
			out[ent.Name] = "party"
			continue
		}
		if includeGuild && localGuild != "" && ent.Guild == localGuild {
			if _, taken := out[ent.Name]; !taken {
				out[ent.Name] = "guild"
			}
		}
	}
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
