package domain

import "time"

// CombatStats accumulates per-fight or per-session combat numbers.
// Damage and Heal are absolute totals; DamageTaken mirrors them from the
// receiving side. CombatStart is the first time the actor dealt damage in
// this window; CombatTime accumulates active time once they're engaged.
type CombatStats struct {
	DamageDealt int64
	HealDone    int64
	Overhealing int64 // healing that exceeded the target's max HP
	DamageTaken int64

	CombatStart time.Time
	LastAction  time.Time
	CombatTime  time.Duration
}

// Reset clears the stats back to zero. Used at fight boundaries when we
// roll the Current window into Overall.
func (c *CombatStats) Reset() {
	*c = CombatStats{}
}

// ElapsedSeconds returns the combat time in fractional seconds, with a
// 1-second floor to avoid divide-by-zero / spiky DPS for short fights.
func (c CombatStats) ElapsedSeconds() float64 {
	if c.CombatTime <= 0 {
		// Fall back to wall-clock since combat start if we have one.
		if !c.CombatStart.IsZero() && c.LastAction.After(c.CombatStart) {
			d := c.LastAction.Sub(c.CombatStart)
			if d < time.Second {
				return 1
			}
			return d.Seconds()
		}
		return 1
	}
	if c.CombatTime < time.Second {
		return 1
	}
	return c.CombatTime.Seconds()
}

// DPS returns damage-per-second for this window.
func (c CombatStats) DPS() float64 { return float64(c.DamageDealt) / c.ElapsedSeconds() }

// HPS returns healing-per-second for this window.
func (c CombatStats) HPS() float64 { return float64(c.HealDone) / c.ElapsedSeconds() }

// HasActivity reports whether this stats block carries any combat
// data. Used by Or() to decide if the current window should display
// or hand off to a fallback (typically LastFight).
func (c CombatStats) HasActivity() bool {
	return c.DamageDealt > 0 || c.HealDone > 0 || c.DamageTaken > 0
}

// Or returns this stats block when it has any combat activity, else
// fallback. Powers the "show the last fight's numbers until the new
// fight has produced something" behaviour requested for the meter so
// rows don't visibly snap to zero between fights.
func (c CombatStats) Or(fallback CombatStats) CombatStats {
	if c.HasActivity() {
		return c
	}
	return fallback
}

// recordHit folds a single hit into both stats windows.
func recordDamage(cur, overall *CombatStats, amount int64, now time.Time) {
	if cur.CombatStart.IsZero() {
		cur.CombatStart = now
	}
	cur.DamageDealt += amount
	cur.LastAction = now
	overall.DamageDealt += amount
	if overall.CombatStart.IsZero() {
		overall.CombatStart = now
	}
	overall.LastAction = now
}

func recordHeal(cur, overall *CombatStats, effective, overheal int64, now time.Time) {
	if effective <= 0 && overheal <= 0 {
		return
	}
	if cur.CombatStart.IsZero() {
		cur.CombatStart = now
	}
	cur.HealDone += effective
	cur.Overhealing += overheal
	cur.LastAction = now
	overall.HealDone += effective
	overall.Overhealing += overheal
	if overall.CombatStart.IsZero() {
		overall.CombatStart = now
	}
	overall.LastAction = now
}

func recordTakenDamage(cur, overall *CombatStats, amount int64, now time.Time) {
	cur.DamageTaken += amount
	cur.LastAction = now
	overall.DamageTaken += amount
	overall.LastAction = now
}
