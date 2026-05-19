package domain

import "time"

// PlayerSnapshot is a serializable view of one party member's combat numbers
// at an instant. Used both for stdout debugging now and for WebSocket push
// to the website later.
type PlayerSnapshot struct {
	UserGuid string `json:"userGuid"`
	ObjectId int64  `json:"objectId,omitempty"`
	Name     string `json:"name"`
	Guild    string `json:"guild,omitempty"`
	IsLocal  bool   `json:"isLocal,omitempty"`

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

// Snapshot is the full set of party-member states the UI needs to render.
type Snapshot struct {
	GeneratedAt time.Time        `json:"generatedAt"`
	Players     []PlayerSnapshot `json:"players"`
}

// Snapshot reads current state into a flat, JSON-friendly value. Safe to
// call concurrently with engine event handlers.
func (e *Engine) Snapshot() Snapshot {
	members := e.store.PartyMembers()
	out := Snapshot{
		GeneratedAt: e.now(),
		Players:     make([]PlayerSnapshot, 0, len(members)),
	}
	e.store.mu.RLock()
	defer e.store.mu.RUnlock()
	for _, m := range members {
		out.Players = append(out.Players, PlayerSnapshot{
			UserGuid: m.UserGuid.String(),
			ObjectId: m.ObjectId,
			Name:     m.Name,
			Guild:    m.Guild,
			IsLocal:  m.IsLocal,

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
	}
	return out
}
