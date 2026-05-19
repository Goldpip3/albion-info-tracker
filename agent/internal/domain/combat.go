package domain

import "time"

// CombatStats accumulates per-fight or per-session combat numbers.
// Damage and Heal are absolute totals; DamageTaken mirrors them from the
// receiving side. CombatStart is the first time the actor dealt damage in
// this window; CombatTime accumulates active time once they're engaged.
type CombatStats struct {
	DamageDealt int64
	HealDone    int64
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

func recordHeal(cur, overall *CombatStats, amount int64, now time.Time) {
	if cur.CombatStart.IsZero() {
		cur.CombatStart = now
	}
	cur.HealDone += amount
	cur.LastAction = now
	overall.HealDone += amount
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
