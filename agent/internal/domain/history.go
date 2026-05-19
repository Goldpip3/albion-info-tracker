package domain

import "time"

// maxFightHistory caps how many completed fights the engine retains in
// memory. Older fights drop off the front. 10 is enough for "scroll back
// the last few pulls" without bloating the snapshot.
const maxFightHistory = 10

// FightArchive is a frozen copy of one completed fight: who was there,
// what they did, and when. Sent in the snapshot's Recent list so the
// frontend can render past fights without phoning home.
type FightArchive struct {
	Number     int                    `json:"number"`
	StartedAt  time.Time              `json:"startedAt"`
	EndedAt    time.Time              `json:"endedAt"`
	DurationMs int64                  `json:"durationMs"`
	Players    []FightPlayerArchive   `json:"players"`
}

// FightPlayerArchive is what the meter needs to render one row of a past
// fight: identity + stats at fight end. No live deltas; numbers are final.
type FightPlayerArchive struct {
	UserGuid    string           `json:"userGuid"`
	Name        string           `json:"name"`
	ClassCode   string           `json:"classCode,omitempty"`
	Role        string           `json:"role,omitempty"`
	RoleLabel   string           `json:"roleLabel,omitempty"`
	IsLocal     bool             `json:"isLocal,omitempty"`
	Damage      int64            `json:"damage"`
	DPS         float64          `json:"dps"`
	Heal        int64            `json:"heal"`
	HPS         float64          `json:"hps"`
	Overheal    int64            `json:"overheal,omitempty"`
	Taken       int64            `json:"taken"`
	Deaths      int              `json:"deaths,omitempty"`
	Spells      []SpellBreakdown `json:"spells,omitempty"`
}
