package domain

import (
	"os"
	"sort"
	"time"
)

// showAll loosens the meter's party gate. When true, Snapshot returns every
// entity with combat activity, not just party members.
var showAll = os.Getenv("ALBION_AGENT_SHOW_ALL") != ""

// topSpells returns the top n spells for an entity by total damage,
// resolving names via the engine's SpellCatalog when available. Caller
// must hold store.mu.
func (e *Engine) topSpells(ent *Entity, n int) []SpellBreakdown {
	if len(ent.BySpell) == 0 {
		return nil
	}
	out := make([]SpellBreakdown, 0, len(ent.BySpell))
	for idx, s := range ent.BySpell {
		var name string
		if e.spells != nil {
			name = e.spells.Name(idx)
		}
		out = append(out, SpellBreakdown{
			Index:       idx,
			Name:        name,
			TotalDamage: s.TotalDamage,
			MaxHit:      s.MaxHit,
			Hits:        s.Hits,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TotalDamage > out[j].TotalDamage })
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
}

// SpellBreakdown is one row of the drill-in screen's ability table.
type SpellBreakdown struct {
	Index       int    `json:"index"`
	Name        string `json:"name,omitempty"` // spells.bin uniquename, when known
	TotalDamage int64  `json:"totalDamage"`
	MaxHit      int64  `json:"maxHit"`
	Hits        int    `json:"hits"`
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
}

// Session is the running session-economy block shown above the meter.
// Silver/respec values are FixPoint internal units (10_000 = 1 unit); the
// frontend divides for display. Fame is already a whole number.
type Session struct {
	StartedAt   time.Time `json:"startedAt"`
	ElapsedMs   int64     `json:"elapsedMs"`
	FameTotal   int64     `json:"fameTotal"`
	SilverTotal int64     `json:"silverTotal"`
	RespecTotal int64     `json:"respecTotal"`
	MightTotal  int64     `json:"mightTotal"`
	DeathsTotal int       `json:"deathsTotal"`
}

// Snapshot is the full set of party-member states the UI needs to render.
type Snapshot struct {
	GeneratedAt time.Time        `json:"generatedAt"`
	Players     []PlayerSnapshot `json:"players"`
	Composition Composition      `json:"composition"`
	Fight       Fight            `json:"fight"`
	Session     Session          `json:"session"`
	Recent      []FightArchive   `json:"recent,omitempty"`
	Events      []ActivityEvent  `json:"events,omitempty"`
}

// Snapshot reads current state into a flat, JSON-friendly value. Safe to
// call concurrently with engine event handlers.
//
// When ALBION_AGENT_SHOW_ALL is set, the snapshot includes every tracked
// entity with any combat activity, not just party members. Useful when
// party events haven't fired yet (mid-zone-start) and you still want to
// see your own damage.
func (e *Engine) Snapshot() Snapshot {
	var members []*Entity
	if showAll {
		members = e.store.AllWithActivity()
	} else {
		members = e.store.PartyMembers()
	}
	now := e.now()
	fightN, elapsed, inCombat := e.FightStatus(now)
	e.sessionMu.Lock()
	sess := Session{
		StartedAt:   e.session.Start,
		ElapsedMs:   int64(e.session.ElapsedSeconds(now) * 1000),
		FameTotal:   e.session.FameTotal,
		SilverTotal: e.session.SilverTotal,
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
		},
		Recent: recent,
		Session: sess,
		Events:  e.events.SnapshotLatest(48),
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

			CurrentDamage: m.Current.DamageDealt,
			CurrentDPS:    m.Current.DPS(),
			OverallDamage: m.Overall.DamageDealt,
			OverallDPS:    m.Overall.DPS(),
			CurrentHeal:   m.Current.HealDone,
			CurrentHPS:    m.Current.HPS(),
			OverallHeal:   m.Overall.HealDone,
			OverallHPS:    m.Overall.HPS(),
			CurrentTaken:  m.Current.DamageTaken,
			OverallTaken:  m.Overall.DamageTaken,

			Spells: e.topSpells(m, 10),
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
	return out
}
