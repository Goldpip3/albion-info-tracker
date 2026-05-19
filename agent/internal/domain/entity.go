package domain

import (
	"sync"
	"time"
)

// Entity is one tracked player — local user, party member, or an outsider
// who happens to be visible. The damage meter only displays entities with
// IsInParty == true, but we track everyone in case party state changes.
type Entity struct {
	UserGuid Guid
	ObjectId int64 // 0 means unknown — set when NewCharacter or a Mount-pair binds it
	Name     string
	Guild    string

	// MainHandItemId is the item index from NewCharacter param 40[0]. Used
	// to classify the player's weapon into a role + 3-letter class chip
	// via gamedata.ClassifyWeapon.
	MainHandItemId int
	ClassCode      string // "DGR", "FIR", … or "—" if unknown
	Role           string // "T", "H", "R", "M", "S", or "?"
	RoleLabel      string // "DAGGERS" / "ADEPT'S ARCLIGHT BLASTERS"

	// Equipment is the 10-slot indexed-by-position array as Albion sends it.
	// Order: 0 MainHand, 1 OffHand, 2 Head, 3 Chest, 4 Shoes, 5 Bag,
	// 6 Cape, 7 Mount, 8 Potion, 9 Food.
	Equipment [10]int

	// Qualities mirrors Equipment, holding the 1..5 quality byte per slot.
	// Zero means "unknown / use Normal default".
	Qualities [10]int

	// ActiveSpells lists spell indices currently bound to ability slots.
	// CharacterEquipmentChanged param 7 ships a short[] of 14 entries:
	//   0..2: MainHand Q/W/E
	//   3:    Armor (chest active)
	//   4:    Head
	//   5:    Shoes / Cape (depending on event variant)
	//   12:   Potion
	//   13:   Food
	// Other indices may carry sub-spells. We store the whole array
	// unmodified for the Party panel to render.
	ActiveSpells [14]int

	// ItemPower is the averaged IP across core slots (MainHand, OffHand,
	// Head, Chest, Shoes, Cape). Computed once on every equipment-event
	// landing — see gamedata.AverageItemPower for the formula.
	ItemPower int

	IsLocal   bool
	IsInParty bool

	// MaxHealth is the entity's max HP, captured from NewCharacter param 22
	// at spawn time. Used to compute overheal in handleHealthUpdate.
	MaxHealth int64

	// Deaths is the running death count for this entity since the last
	// session reset.
	Deaths int

	Current CombatStats
	Overall CombatStats

	// BySpell aggregates damage per spell index for the current fight.
	// Resets between fights so the drill-in shows "what's being used right
	// now". Key is CausingSpellIndex from HealthUpdate (param 7).
	BySpell map[int]*SpellTotals

	// BySpellSession aggregates damage + casts per spell across the WHOLE
	// session. Doesn't reset between fights — only on ResetSession. Powers
	// the "spells cast this session" report.
	BySpellSession map[int]*SpellTotals

	// ByTarget aggregates damage dealt to each affected entity. Key is the
	// target's ObjectId. Used in the drill-in to show "who did this player
	// focus" — tanks should show the boss, cleavers should show many adds.
	ByTarget map[int64]int64

	// ActiveEffects lists spell indices currently buffing/debuffing this
	// entity. Refreshed on ActiveSpellEffectsUpdate. May be nil.
	ActiveEffects []int

	// AssistsBySpell tracks "while this player's debuff was up on a target,
	// how much total damage hit that target." Key is the spell index this
	// player cast. Sums across all (target, window) tuples for the session.
	AssistsBySpell map[int]*AssistTotals

	LastSeen time.Time
}

// AssistTotals is the "Level 2" debuff-window contribution: how long this
// player's debuff sat on targets (uptime) and how much damage the party
// dealt during that uptime. Doesn't claim *causation* — just attribution.
type AssistTotals struct {
	UptimeMs     int64
	DamageDuring int64
}

// SpellTotals is the per-(player, spell) breakdown shown on the drill-in
// screen.
//
// Hits is the HealthUpdate count (number of times this spell landed).
// Casts is the CastFinished count (number of times this spell was cast).
// In single-target spells Hits ≈ Casts; in AoE spells Hits ≫ Casts.
type SpellTotals struct {
	TotalDamage int64
	MaxHit      int64
	Hits        int
	Casts       int
}

// Store is the in-memory entity registry. All mutating methods take the write
// lock; readers (e.g. Snapshot) take the read lock.
type Store struct {
	mu     sync.RWMutex
	byGuid map[Guid]*Entity

	// byObject is an O(1) ObjectId → Entity index, maintained alongside
	// byGuid. Every place that mutates Entity.ObjectId must call rebind
	// to keep this map consistent. Hot-path callers (HealthUpdate runs
	// twice per damage event) used to scan byGuid linearly — at 5v5+ ZvZ
	// rates that's O(events × players) per second, easily 10k+ map
	// iterations per tick. The map lookup is constant.
	byObject map[int64]*Entity

	// pendingMountObjectId is filled by MountStart and consumed by the next
	// NewMountObject to bind ObjectId <-> UserGuid.
	pendingMountObjectId int64
	pendingMountAt       time.Time

	// localGuid is set by JoinFinished; non-zero once known.
	localGuid Guid
}

// NewStore constructs an empty store.
func NewStore() *Store {
	return &Store{
		byGuid:   make(map[Guid]*Entity),
		byObject: make(map[int64]*Entity),
	}
}

// rebind keeps byObject in sync when an entity's ObjectId changes.
// Caller must hold the write lock.
func (s *Store) rebind(e *Entity, newObjectId int64) {
	if e.ObjectId != 0 && s.byObject[e.ObjectId] == e {
		delete(s.byObject, e.ObjectId)
	}
	e.ObjectId = newObjectId
	if newObjectId != 0 {
		s.byObject[newObjectId] = e
	}
}

// SetLocalGuid records the local user's Guid. Any existing matching entity
// is flagged IsLocal=true and IsInParty=true (local is always in their own
// party, even when alone).
func (s *Store) SetLocalGuid(g Guid) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.localGuid = g
	if e, ok := s.byGuid[g]; ok {
		e.IsLocal = true
		e.IsInParty = true
	}
}

// UpsertByGuid creates or updates an entity keyed by UserGuid. Empty fields
// (zero ObjectId, empty Name/Guild) preserve the existing values.
func (s *Store) UpsertByGuid(g Guid, objectId int64, name, guild string) *Entity {
	if g.IsZero() {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.byGuid[g]
	if !ok {
		e = &Entity{UserGuid: g}
		s.byGuid[g] = e
	}
	if objectId != 0 && objectId != e.ObjectId {
		s.rebind(e, objectId)
	}
	if name != "" {
		e.Name = name
	}
	if guild != "" {
		e.Guild = guild
	}
	if !s.localGuid.IsZero() && g == s.localGuid {
		e.IsLocal = true
		e.IsInParty = true
	}
	e.LastSeen = time.Now()
	return e
}

// byObjectIdLocked returns the entity with the given ObjectId. Caller
// must already hold store.mu (read or write). O(1) via byObject map.
func (s *Store) byObjectIdLocked(id int64) *Entity {
	if id == 0 {
		return nil
	}
	return s.byObject[id]
}

// ByObjectId returns the entity with the given ObjectId, or nil if none.
// O(1) via byObject map.
func (s *Store) ByObjectId(id int64) *Entity {
	if id == 0 {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.byObject[id]
}

// ByGuid returns the entity with the given Guid, or nil if none.
func (s *Store) ByGuid(g Guid) *Entity {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.byGuid[g]
}

// localGuidEntity returns the local-player entity (if known). Convenience
// for callers that don't want to grab the lock themselves.
func (s *Store) localGuidEntity() *Entity {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.localGuid.IsZero() {
		return nil
	}
	return s.byGuid[s.localGuid]
}

// MarkInParty sets IsInParty on the entity for guid, creating it if absent.
func (s *Store) MarkInParty(g Guid, inParty bool) {
	if g.IsZero() {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.byGuid[g]
	if !ok {
		e = &Entity{UserGuid: g, LastSeen: time.Now()}
		s.byGuid[g] = e
	}
	e.IsInParty = inParty
}

// ResetParty clears IsInParty for every entity except the local player.
func (s *Store) ResetParty() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, e := range s.byGuid {
		if !e.IsLocal {
			e.IsInParty = false
		}
	}
}

// StashMountStart records the most recent rider ObjectId; consumed by the
// next NewMountObject within the binding window.
func (s *Store) StashMountStart(objectId int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pendingMountObjectId = objectId
	s.pendingMountAt = time.Now()
}

// BindRiderGuid pairs a UserGuid (from NewMountObject) with the most recent
// MountStart ObjectId, if the pair is fresh and the Guid isn't the local
// player's (the local player's NewMountObject has no preceding MountStart).
const mountBindingWindow = 2 * time.Second

func (s *Store) BindRiderGuid(g Guid) {
	if g.IsZero() {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	// Always upsert an entity for the rider. Even without a matching
	// MountStart (e.g. the local player who used a Mount REQUEST, or a
	// remote player who entered view already mounted), at least we know
	// their UserGuid. A future MountStart pair, or a NewCharacter event,
	// will fill in the ObjectId.
	e, ok := s.byGuid[g]
	if !ok {
		e = &Entity{UserGuid: g, LastSeen: time.Now()}
		s.byGuid[g] = e
	}
	e.LastSeen = time.Now()

	pending := s.pendingMountObjectId
	s.pendingMountObjectId = 0
	if pending == 0 {
		return
	}
	if time.Since(s.pendingMountAt) > mountBindingWindow {
		return
	}
	if !s.localGuid.IsZero() && g == s.localGuid {
		return
	}
	if e.ObjectId == 0 {
		s.rebind(e, pending)
	}
}

// Counts returns (total tracked entities, entities with both Guid and
// non-zero ObjectId, entities currently flagged IsInParty). Useful for
// diagnostic output.
func (s *Store) Counts() (total, bound, party int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, e := range s.byGuid {
		total++
		if e.ObjectId != 0 {
			bound++
		}
		if e.IsInParty {
			party++
		}
	}
	return
}

// AllWithActivity returns every entity that has dealt or taken any damage
// or done any healing. Used by "show all" mode to render a meter even
// when party membership isn't known yet.
func (s *Store) AllWithActivity() []*Entity {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Entity, 0)
	for _, e := range s.byGuid {
		if e.Overall.DamageDealt > 0 || e.Overall.HealDone > 0 || e.Overall.DamageTaken > 0 {
			out = append(out, e)
		}
	}
	return out
}

// PartyMembers returns entities currently flagged IsInParty.
func (s *Store) PartyMembers() []*Entity {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Entity, 0)
	for _, e := range s.byGuid {
		if e.IsInParty {
			out = append(out, e)
		}
	}
	return out
}
