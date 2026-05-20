package domain

import (
	"os"
	"sort"
	"strings"
	"time"

	"github.com/Goldpip3/albion-info-tracker/agent/internal/gamedata"
)

// showAll loosens the meter's party gate. When true, Snapshot returns every
// entity with combat activity, not just party members.
var showAll = os.Getenv("ALBION_AGENT_SHOW_ALL") != ""

// topSpells returns the top n spells for an entity by total damage,
// resolving names via the engine's SpellCatalog when available. Caller
// must hold store.mu. Filters out passive/aura/armor/food entries — those
// are emitted by Albion as damage events too but they're noise in the
// "what abilities am I using" view.
//
// Sort is deterministic: total damage descending, then by spell index so
// rows with the same numeric value don't shuffle each tick.
func (e *Engine) topSpells(ent *Entity, n int) []SpellBreakdown {
	if len(ent.BySpell) == 0 {
		return nil
	}
	out := make([]SpellBreakdown, 0, len(ent.BySpell))
	for idx, s := range ent.BySpell {
		name := e.localizedSpellName(idx)
		if isPassiveSpell(name) {
			continue
		}
		out = append(out, SpellBreakdown{
			Index:       idx,
			Name:        name,
			TotalDamage: s.TotalDamage,
			MaxHit:      s.MaxHit,
			Hits:        s.Hits,
			Casts:       s.Casts,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].TotalDamage != out[j].TotalDamage {
			return out[i].TotalDamage > out[j].TotalDamage
		}
		return out[i].Index < out[j].Index
	})
	if len(out) > n {
		out = out[:n]
	}
	return out
}

// sessionSpells is the session-level analogue of topSpells. Same shape;
// reads from BySpellSession which never resets between fights. Sorted by
// cast count then total damage, deterministic tie-break by index.
func (e *Engine) sessionSpells(ent *Entity, n int) []SpellBreakdown {
	if len(ent.BySpellSession) == 0 {
		return nil
	}
	out := make([]SpellBreakdown, 0, len(ent.BySpellSession))
	for idx, s := range ent.BySpellSession {
		name := e.localizedSpellName(idx)
		if isPassiveSpell(name) {
			continue
		}
		out = append(out, SpellBreakdown{
			Index:       idx,
			Name:        name,
			TotalDamage: s.TotalDamage,
			MaxHit:      s.MaxHit,
			Hits:        s.Hits,
			Casts:       s.Casts,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Casts != out[j].Casts {
			return out[i].Casts > out[j].Casts
		}
		if out[i].TotalDamage != out[j].TotalDamage {
			return out[i].TotalDamage > out[j].TotalDamage
		}
		return out[i].Index < out[j].Index
	})
	if len(out) > n {
		out = out[:n]
	}
	return out
}

// isPassiveSpell returns true for spell uniquenames that represent armor
// procs, food buffs, or movement-skill sub-effects — things that fire
// damage events but aren't "abilities the player chose to cast." Filters
// out clutter like "Dynamic curse cloth", "Food crafting p1", and
// "Skillshot teleport end" that confuses the drill-in.
func isPassiveSpell(name string) bool {
	if name == "" {
		return false // keep unknown indices; might be the real ability
	}
	u := strings.ToUpper(name)
	patterns := []string{
		"DYNAMIC_",         // armor passives ("DYNAMIC_CURSE_CLOTH")
		"FOOD_",            // food buffs
		"_FOOD",
		"_EFFECT",          // sub-effect bookkeeping
		"_END",             // ability-end markers ("SKILLSHOT_TELEPORT_END")
		"_BUFF",            // self-buffs
		"_DEBUFF",          // debuffs handled in Assists tab
		"_PASSIVE",         // passive trigger
		"_AURA",            // aura ticks
		"_BLEED_",          // bleed dot label
		"_POISON_",         // poison dot label
		"_PROC",            // armor proc
		"SET_BONUS",        // armor set bonus
		"CONSUMABLE_",      // potions / food etc.
	}
	for _, p := range patterns {
		if strings.Contains(u, p) {
			return true
		}
	}
	return false
}

// topAssists returns the top n debuff windows by damage-during, resolving
// spell names where possible. Caller must hold store.mu.
func (e *Engine) topAssists(ent *Entity, n int) []AssistBreakdown {
	if len(ent.AssistsBySpell) == 0 {
		return nil
	}
	out := make([]AssistBreakdown, 0, len(ent.AssistsBySpell))
	for idx, a := range ent.AssistsBySpell {
		name := e.localizedSpellName(idx)
		out = append(out, AssistBreakdown{
			Index:        idx,
			Name:         name,
			UptimeMs:     a.UptimeMs,
			DamageDuring: a.DamageDuring,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].DamageDuring > out[j].DamageDuring })
	if len(out) > n {
		out = out[:n]
	}
	return out
}

// slotNames maps equipment slot indices to the labels we render. Order
// matches the array Albion ships.
var slotNames = [10]string{
	"MainHand", "OffHand", "Head", "Chest", "Shoes",
	"Bag", "Cape", "Mount", "Potion", "Food",
}

// equipmentSlots builds the per-slot loadout for a player. Each slot
// that has an item resolves to its localized English name (when known)
// + IP contribution. Empty slots are omitted entirely.
func (e *Engine) equipmentSlots(ent *Entity) []SlotInfo {
	if e.items == nil {
		return nil
	}
	out := make([]SlotInfo, 0, 10)
	for i, idx := range ent.Equipment {
		if idx == 0 {
			continue
		}
		uniqueName := e.items.Name(idx)
		display := ""
		if e.loc != nil {
			display = e.loc.ItemName(uniqueName)
		}
		if display == "" {
			display = uniqueName
		}
		q := 1
		if ent.Qualities[i] >= 1 {
			q = ent.Qualities[i]
		}
		out = append(out, SlotInfo{
			Slot:      slotNames[i],
			Name:      display,
			ItemPower: gamedata.ItemPowerOf(uniqueName, q),
		})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// spellSlotLabels maps ActiveSpells array indices to user-visible slot
// keys. Only the slots Albion populates with real player abilities are
// included; the rest collapse to "—". See SAT's
// CharacterEquipmentChangedEvent for the index→slot mapping.
var spellSlotLabels = map[int]string{
	0: "Q", 1: "W", 2: "E",
	3: "Armor", 4: "Head", 5: "Shoes",
	12: "Potion", 13: "Food",
}

// activeSpellSlots resolves each player ability into its slot key + the
// localized English name. Empty / unrecognised entries are skipped.
// Passive-style spell names (sub-effects, armor procs, food buffs) are
// also skipped — those leaked into Q/W/E rendering on at least one
// observed party loadout where the equipment array shifted; filtering
// here keeps the bound-ability row honest until we have a confirmed
// fresh index→slot map from a verbose-log capture.
func (e *Engine) activeSpellSlots(ent *Entity) []SpellSlotInfo {
	if e.spells == nil {
		return nil
	}
	out := make([]SpellSlotInfo, 0, 8)
	for i, idx := range ent.ActiveSpells {
		if idx <= 0 {
			continue
		}
		slot, ok := spellSlotLabels[i]
		if !ok {
			continue
		}
		name := e.localizedSpellName(idx)
		if name == "" {
			continue
		}
		if isPassiveSpell(name) {
			continue
		}
		out = append(out, SpellSlotInfo{Slot: slot, Name: name})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// topTargets returns the top n damage recipients for an entity,
// resolving names in this order:
//   1. tracked player entity by ObjectId (other party members hit by
//      friendly fire / training dummies)
//   2. mob name cache (populated from NewMob events)
//   3. "" — frontend renders "#<id>" as the final fallback
// Caller must hold store.mu.
func (e *Engine) topTargets(ent *Entity, n int) []TargetBreakdown {
	if len(ent.ByTarget) == 0 {
		return nil
	}
	out := make([]TargetBreakdown, 0, len(ent.ByTarget))
	for id, dmg := range ent.ByTarget {
		var name string
		if t := e.store.byObjectIdLocked(id); t != nil {
			name = t.Name
		}
		if name == "" {
			name = e.MobName(id)
		}
		out = append(out, TargetBreakdown{ObjectId: id, Name: name, Damage: dmg})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Damage > out[j].Damage })
	if len(out) > n {
		out = out[:n]
	}
	return out
}

// PlayerSnapshot is a serializable view of one party member's combat numbers
// at an instant. Used both for stdout debugging now and for WebSocket push
// to the website later.
type PlayerSnapshot struct {
	UserGuid string `json:"userGuid"`
	ObjectId int64  `json:"objectId,omitempty"`
	Name     string `json:"name"`
	Guild    string `json:"guild,omitempty"`
	IsLocal  bool   `json:"isLocal,omitempty"`
	Deaths   int    `json:"deaths,omitempty"`
	Overheal int64  `json:"overheal,omitempty"`

	// Weapon-derived role + class chip. ClassCode is a 3-letter token
	// the frontend renders in a chip; Role is one of T/H/R/M/S/? for
	// the composition strip; RoleLabel is the full subtitle.
	ClassCode string `json:"classCode,omitempty"`
	Role      string `json:"role,omitempty"`
	RoleLabel string `json:"roleLabel,omitempty"`

	// ItemPower is the averaged IP across the player's core gear slots
	// (MainHand, OffHand, Head, Chest, Shoes, Cape). The web renders
	// this as the IPChip in place of the 3-letter class code.
	ItemPower int `json:"itemPower,omitempty"`

	// EquipmentSlots is the resolved per-slot loadout — name + IP for
	// each of the 10 slots. Powers the IPChip hover tooltip and the
	// Party panel's gear strip. Empty slots are omitted by JSON.
	EquipmentSlots []SlotInfo `json:"equipmentSlots,omitempty"`

	// ActiveSpellSlots is the loadout of bound abilities. Each entry has
	// a slot key ("Q" / "W" / "E" / "Armor" / "Head" / "Shoes" / "Cape"
	// / "Potion" / "Food") and the resolved English spell name.
	ActiveSpellSlots []SpellSlotInfo `json:"activeSpellSlots,omitempty"`

	CurrentDamage int64   `json:"currentDamage"`
	CurrentDPS    float64 `json:"currentDps"`
	OverallDamage int64   `json:"overallDamage"`
	OverallDPS    float64 `json:"overallDps"`
	CurrentHeal   int64   `json:"currentHeal"`
	CurrentHPS    float64 `json:"currentHps"`
	OverallHeal   int64   `json:"overallHeal"`
	OverallHPS    float64 `json:"overallHps"`
	CurrentTaken  int64   `json:"currentTaken"`
	OverallTaken  int64   `json:"overallTaken"`

	// Top spells used during the current fight, sorted by total damage
	// descending. Only included when there's something to show.
	Spells []SpellBreakdown `json:"spells,omitempty"`

	// SessionSpells is the same shape as Spells but covers the whole
	// session (doesn't reset between fights). Powers the
	// "how many times have I cast X this session" view.
	SessionSpells []SpellBreakdown `json:"sessionSpells,omitempty"`

	// Targets — top recipients of damage from this player. Top 5,
	// descending. Tanks focus the boss; cleavers spread across mobs.
	Targets []TargetBreakdown `json:"targets,omitempty"`

	// ActiveEffects is the list of spell indices currently active on
	// this entity (buffs + debuffs). Latest snapshot from the most
	// recent ActiveSpellEffectsUpdate.
	ActiveEffects []int `json:"activeEffects,omitempty"`

	// Assists is the per-spell debuff-window attribution: how long this
	// player's debuff was active on enemies and how much damage the
	// party dealt during that window. Sorted by DamageDuring desc.
	Assists []AssistBreakdown `json:"assists,omitempty"`
}

// AssistBreakdown is the "Level 2" contribution surfaced in the drill-in
// row's right column: which of the player's spells made the biggest
// difference, by uptime + damage that flowed under it.
type AssistBreakdown struct {
	Index        int    `json:"index"`
	Name         string `json:"name,omitempty"`
	UptimeMs     int64  `json:"uptimeMs"`
	DamageDuring int64  `json:"damageDuring"`
}

// SlotInfo is one row of a player's equipment loadout — the resolved
// English item name plus its IP contribution. Empty slots are omitted
// from the snapshot entirely.
type SlotInfo struct {
	Slot      string `json:"slot"`           // "MainHand" / "OffHand" / "Head" / …
	Name      string `json:"name,omitempty"` // "Adept's Arclight Blasters"
	ItemPower int    `json:"itemPower,omitempty"`
}

// SpellSlotInfo is one bound ability the player has on their bar.
type SpellSlotInfo struct {
	Slot string `json:"slot"`           // "Q" / "W" / "E" / "Armor" / "Head" / "Shoes" / "Cape" / "Potion" / "Food"
	Name string `json:"name,omitempty"` // "Flickershot" / "Caltrops" / …
}

// SpellBreakdown is one row of the drill-in screen's ability table.
type SpellBreakdown struct {
	Index       int    `json:"index"`
	Name        string `json:"name,omitempty"` // spells.bin uniquename, when known
	TotalDamage int64  `json:"totalDamage"`
	MaxHit      int64  `json:"maxHit"`
	Hits        int    `json:"hits"`
	Casts       int    `json:"casts,omitempty"`
}

// TargetBreakdown is one row of the per-target table — how much damage
// the player dealt to a specific target. Name is best-effort; tracked
// entities resolve, raw mob ObjectIds fall through as #<id>.
type TargetBreakdown struct {
	ObjectId int64  `json:"objectId"`
	Name     string `json:"name,omitempty"`
	Damage   int64  `json:"damage"`
}

// Composition counts each role across the snapshot's players.
type Composition struct {
	Tank    int `json:"tank"`
	Healer  int `json:"healer"`
	Ranged  int `json:"ranged"`
	Melee   int `json:"melee"`
	Support int `json:"support"`
	Control int `json:"control"`
	Unknown int `json:"unknown"`
	Total   int `json:"total"`
}

// Fight is the running state of the current combat encounter.
type Fight struct {
	Number    int    `json:"number"`     // 1-indexed; 0 means no fight yet
	ElapsedMs int64  `json:"elapsedMs"`  // milliseconds since the fight started
	InCombat  bool   `json:"inCombat"`
	Zone      string `json:"zone,omitempty"`
}

// Session is the running session-economy block shown above the meter.
// Silver/respec values are FixPoint internal units (10_000 = 1 unit); the
// frontend divides for display. Fame is already a whole number.
type Session struct {
	StartedAt   time.Time `json:"startedAt"`
	ElapsedMs   int64     `json:"elapsedMs"`
	FameTotal   int64     `json:"fameTotal"`
	SilverTotal int64     `json:"silverTotal"`
	MobSilverTotal int64  `json:"mobSilverTotal,omitempty"`
	RespecTotal int64     `json:"respecTotal"`
	MightTotal  int64     `json:"mightTotal"`
	DeathsTotal int       `json:"deathsTotal"`
}

// VisiblePlayer is a lightweight "add to party" candidate — a tracked
// player who isn't currently in the party. The web renders these so the
// user can manually add teammates the agent never saw join (mixed-guild
// parties, agent started after grouping).
type VisiblePlayer struct {
	UserGuid  string `json:"userGuid"`
	Name      string `json:"name"`
	ItemPower int    `json:"itemPower,omitempty"`
	ClassCode string `json:"classCode,omitempty"`
	Role      string `json:"role,omitempty"`
	RoleLabel string `json:"roleLabel,omitempty"`
}

// Snapshot is the full set of party-member states the UI needs to render.
type Snapshot struct {
	GeneratedAt time.Time         `json:"generatedAt"`
	Players     []PlayerSnapshot  `json:"players"`
	Composition Composition       `json:"composition"`
	Fight       Fight             `json:"fight"`
	Session     Session           `json:"session"`
	Recent      []FightArchive    `json:"recent,omitempty"`
	Events      []ActivityEvent   `json:"events,omitempty"`
	Sessions    []ArchivedSession `json:"sessions,omitempty"`
	Zones       []ZoneVisit       `json:"zones,omitempty"`
	Dungeon     *DungeonRun       `json:"dungeon,omitempty"`
	Loot        []LootEntry       `json:"loot,omitempty"`
	LooterTotals []LooterTotals   `json:"looterTotals,omitempty"`
	// VisiblePlayers are tracked, named players not currently in the
	// party — the manual "add to party" candidates for the Party panel.
	VisiblePlayers []VisiblePlayer `json:"visiblePlayers,omitempty"`
}

// Snapshot reads current state into a flat, JSON-friendly value. Safe to
// call concurrently with engine event handlers.
//
// Membership rules, in order:
//
//  1. ALBION_AGENT_SHOW_ALL set → render every entity with combat
//     activity, no filtering. Power-user escape hatch.
//  2. Party detection reports more than just the local player → trust it.
//     Random players in the zone are filtered out as desired.
//  3. Party detection only knows about the local player → fall back to
//     activity, but only include entities whose Guild matches the local
//     player's. This catches the well-known mid-zone case (user joined
//     the party before the agent started, so PartyJoined never fired)
//     without dragging in unrelated players who happened to be hitting
//     things nearby. Re-zoning doesn't reliably re-broadcast
//     PartyJoined, so we fix it here instead of leaning on a manual
//     env-var workaround.
//
// When the local player is guildless the guild filter has nothing to
// pivot on, so we hold the line at strict party-only — better than
// flooding the meter with strangers.
// scopedMembers returns the entities the meter should display under the
// current scope. Shared by Snapshot (live view) and archiveCurrentFight
// (past fights) so the two never disagree — picking a past fight from
// the dropdown shows the same set of players the live meter showed at
// the time. Acquires store locks internally via PartyMembers /
// AllWithActivity, so the caller MUST NOT hold store.mu.
//
// Scope values (from the Settings → Meter scope control, plumbed via
// the setLootFilter command):
//   - "everyone"    every entity with combat activity (ZvZ / open world)
//   - "partyGuild"  party + same-guild + allowlist (default; dungeons
//                   with guildies)
//   - "party"       confirmed party + allowlist only (tight dungeon runs)
//
// ALBION_AGENT_SHOW_ALL still forces "everyone" regardless of the
// scope setting — the power-user escape hatch.
func (e *Engine) scopedMembers() []*Entity {
	mode := e.LootFilterMode()
	if showAll || mode == "everyone" {
		return e.store.AllWithActivity()
	}

	members := e.store.PartyMembers()
	if len(members) > 1 {
		// Party detection succeeded — trust it, drop random players.
		return members
	}

	// Party detection only knows the local player. Fall back to
	// activity, filtered by the scope: same-guild for "partyGuild",
	// allowlist-only for "party". Guildless locals can't pivot on
	// guild, so they collapse to allowlist-only too.
	local := e.store.localGuidEntity()
	includeGuild := mode == "partyGuild" && local != nil && local.Guild != ""
	all := e.store.AllWithActivity()
	filtered := make([]*Entity, 0, len(all))
	for _, ent := range all {
		switch {
		case ent.IsLocal, e.AlwaysIncludes(ent.Name):
			filtered = append(filtered, ent)
		case includeGuild && ent.Guild == local.Guild:
			filtered = append(filtered, ent)
		}
	}
	if len(filtered) > len(members) {
		return filtered
	}
	return members
}

func (e *Engine) Snapshot() Snapshot {
	members := e.scopedMembers()
	now := e.now()
	fightN, elapsed, inCombat := e.FightStatus(now)
	e.sessionMu.Lock()
	sess := Session{
		StartedAt:   e.session.Start,
		ElapsedMs:   int64(e.session.ElapsedSeconds(now) * 1000),
		FameTotal:   e.session.FameTotal,
		SilverTotal: e.session.SilverTotal,
		MobSilverTotal: e.session.MobSilverTotal,
		RespecTotal: e.session.RespecTotal,
		MightTotal:  e.session.MightTotal,
		DeathsTotal: e.session.DeathsTotal,
	}
	e.sessionMu.Unlock()

	e.fightMu.Lock()
	recent := make([]FightArchive, len(e.fightHistory))
	copy(recent, e.fightHistory)
	e.fightMu.Unlock()

	out := Snapshot{
		GeneratedAt: now,
		Players:     make([]PlayerSnapshot, 0, len(members)),
		Fight: Fight{
			Number:    fightN,
			ElapsedMs: elapsed.Milliseconds(),
			InCombat:  inCombat,
			Zone:      e.Zone(),
		},
		Recent:       recent,
		Session:      sess,
		Events:       e.events.SnapshotLatest(48),
		Zones:        e.ZoneHistory(),
		Dungeon:      e.CurrentDungeon(),
		Loot:         e.LootLog(),
		LooterTotals: e.LooterTotalsList(),
	}
	if e.sessions != nil {
		out.Sessions = e.sessions.List()
	}
	e.store.mu.RLock()
	defer e.store.mu.RUnlock()
	for _, m := range members {
		out.Players = append(out.Players, PlayerSnapshot{
			UserGuid:  m.UserGuid.String(),
			ObjectId:  m.ObjectId,
			Name:      m.Name,
			Guild:     m.Guild,
			IsLocal:   m.IsLocal,
			Deaths:    m.Deaths,
			Overheal:  m.Overall.Overhealing,
			ClassCode: m.ClassCode,
			Role:      m.Role,
			RoleLabel: m.RoleLabel,
			ItemPower: m.ItemPower,

			// Current values fall back to LastFight when this fight
			// hasn't produced damage yet so the row doesn't snap to
			// zero on each fight boundary. Overall always reflects the
			// session-running total.
			CurrentDamage: m.Current.Or(m.LastFight).DamageDealt,
			CurrentDPS:    m.Current.Or(m.LastFight).DPS(),
			OverallDamage: m.Overall.DamageDealt,
			OverallDPS:    m.Overall.DPS(),
			CurrentHeal:   m.Current.Or(m.LastFight).HealDone,
			CurrentHPS:    m.Current.Or(m.LastFight).HPS(),
			OverallHeal:   m.Overall.HealDone,
			OverallHPS:    m.Overall.HPS(),
			CurrentTaken:  m.Current.Or(m.LastFight).DamageTaken,
			OverallTaken:  m.Overall.DamageTaken,

			Spells:           e.topSpells(m, 10),
			SessionSpells:    e.sessionSpells(m, 20),
			Targets:          e.topTargets(m, 5),
			ActiveEffects:    append([]int(nil), m.ActiveEffects...),
			Assists:          e.topAssists(m, 10),
			EquipmentSlots:   e.equipmentSlots(m),
			ActiveSpellSlots: e.activeSpellSlots(m),
		})
		switch m.Role {
		case "T":
			out.Composition.Tank++
		case "H":
			out.Composition.Healer++
		case "R":
			out.Composition.Ranged++
		case "M":
			out.Composition.Melee++
		case "S":
			out.Composition.Support++
		case "C":
			out.Composition.Control++
		default:
			out.Composition.Unknown++
		}
		out.Composition.Total++
	}

	// Add-to-party candidates: every tracked named player who isn't
	// already displayed (not in members) and isn't the local player.
	// The web surfaces these so a mixed-guild party the agent never saw
	// form can be assembled by hand. Bounded by zone presence.
	shown := make(map[string]struct{}, len(out.Players))
	for _, pl := range out.Players {
		shown[pl.UserGuid] = struct{}{}
	}
	for _, ent := range e.store.AllPlayers() {
		if ent.IsLocal || ent.UserGuid.IsZero() {
			continue
		}
		gid := ent.UserGuid.String()
		if _, ok := shown[gid]; ok {
			continue
		}
		out.VisiblePlayers = append(out.VisiblePlayers, VisiblePlayer{
			UserGuid:  gid,
			Name:      ent.Name,
			ItemPower: ent.ItemPower,
			ClassCode: ent.ClassCode,
			Role:      ent.Role,
			RoleLabel: ent.RoleLabel,
		})
	}
	sort.Slice(out.VisiblePlayers, func(i, j int) bool {
		return out.VisiblePlayers[i].Name < out.VisiblePlayers[j].Name
	})

	// Deterministic player order at the source. members comes from a Go
	// map (randomized iteration), so without this the array arrives in a
	// different order every tick — and any client sort that omits a
	// tiebreak (the "Top" / "Carried by" leaders, when tied) would let
	// that randomness flicker the displayed name. A stable guid order
	// here means JS's stable sort preserves it on ties everywhere.
	sort.Slice(out.Players, func(i, j int) bool {
		return out.Players[i].UserGuid < out.Players[j].UserGuid
	})
	return out
}
