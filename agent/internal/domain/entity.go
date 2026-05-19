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
	RoleLabel      string // "MELEE DPS · DAGGERS"

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

	// BySpell aggregates damage per spell index for the drill-in view.
	// Key is the CausingSpellIndex from HealthUpdate (param 7).
	BySpell map[int]*SpellTotals

	// ByTarget aggregates damage dealt to each affected entity. Key is the
	// target's ObjectId. Used in the drill-in to show "who did this player
	// focus" — tanks should show the boss, cleavers should show many adds.
	ByTarget map[int64]int64

	// ActiveEffects lists spell indices currently buffing/debuffing this
	// entity. Refreshed on ActiveSpellEffectsUpdate. May be nil.
	ActiveEffects []int

	LastSeen time.Time
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

	// pendingMountObjectId is filled by MountStart and consumed by the next
	// NewMountObject to bind ObjectId <-> UserGuid.
	pendingMountObjectId int64
	pendingMountAt       time.Time

	// localGuid is set by JoinFinished; non-zero once known.
	localGuid Guid
}

// NewStore constructs an empty store.
func NewStore() *Store {
	return &Store{byGuid: make(map[Guid]*Entity)}
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
	if objectId != 0 {
		e.ObjectId = objectId
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
// must already hold store.mu (read or write).
func (s *Store) byObjectIdLocked(id int64) *Entity {
	if id == 0 {
		return nil
	}
	for _, e := range s.byGuid {
		if e.ObjectId == id {
			return e
		}
	}
	return nil
}

// ByObjectId returns the entity with the given ObjectId, or nil if none.
func (s *Store) ByObjectId(id int64) *Entity {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, e := range s.byGuid {
		if e.ObjectId == id && id != 0 {
			return e
		}
	}
	return nil
}

// ByGuid returns the entity with the given Guid, or nil if none.
func (s *Store) ByGuid(g Guid) *Entity {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.byGuid[g]
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
		e.ObjectId = pending
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
