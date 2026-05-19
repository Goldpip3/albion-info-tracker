package domain

import (
	"os"
	"time"
)

// showAll loosens the meter's party gate. When true, Snapshot returns every
// entity with combat activity, not just party members.
var showAll = os.Getenv("ALBION_AGENT_SHOW_ALL") != ""

// PlayerSnapshot is a serializable view of one party member's combat numbers
// at an instant. Used both for stdout debugging now and for WebSocket push
// to the website later.
type PlayerSnapshot struct {
	UserGuid string `json:"userGuid"`
	ObjectId int64  `json:"objectId,omitempty"`
	Name     string `json:"name"`
	Guild    string `json:"guild,omitempty"`
	IsLocal  bool   `json:"isLocal,omitempty"`

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
}

// Composition counts each role across the snapshot's players.
type Composition struct {
	Tank      int `json:"tank"`
	Healer    int `json:"healer"`
	Ranged    int `json:"ranged"`
	Melee     int `json:"melee"`
	Support   int `json:"support"`
	Unknown   int `json:"unknown"`
	Total     int `json:"total"`
}

// Snapshot is the full set of party-member states the UI needs to render.
type Snapshot struct {
	GeneratedAt time.Time        `json:"generatedAt"`
	Players     []PlayerSnapshot `json:"players"`
	Composition Composition      `json:"composition"`
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
	out := Snapshot{
		GeneratedAt: e.now(),
		Players:     make([]PlayerSnapshot, 0, len(members)),
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
		default:
			out.Composition.Unknown++
		}
		out.Composition.Total++
	}
	return out
}
