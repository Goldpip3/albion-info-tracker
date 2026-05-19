package domain

import "time"

// SessionStats tracks running cumulative gains since the agent started or
// since the last ResetSession() call. Exposed in the snapshot so the
// frontend can render headline numbers (Fame, Silver, Respec) plus a
// per-hour rate.
type SessionStats struct {
	Start         time.Time
	FameTotal     int64 // raw fame points (TotalPlayerFame delta sum)
	SilverTotal   int64 // raw silver units (10_000 = 1 silver per SAT's FixPoint)
	RespecTotal   int64 // raw respec credits (FixPoint internal)
	MightTotal    int64 // raw might (FixPoint internal; 10_000 = 1 might)
	DeathsTotal   int   // total deaths recorded across all entities
	prevFame      int64 // running TotalPlayerFame for delta calc
	prevSilver    int64 // running CurrentPlayerSilver for delta calc
	prevFameKnown bool  // wait for second event before counting; first sets baseline
	prevSilverKnown bool
}

// Reset zeroes the session. Called by the engine when the operator clicks
// "New session" or when the agent starts.
func (s *SessionStats) Reset(now time.Time) {
	*s = SessionStats{Start: now}
}

// AccumulateFame folds a new TotalPlayerFame reading into the session,
// computing the delta against the prior reading. First reading sets the
// baseline so existing lifetime fame doesn't get counted.
func (s *SessionStats) AccumulateFame(totalPlayerFame int64) {
	if !s.prevFameKnown {
		s.prevFame = totalPlayerFame
		s.prevFameKnown = true
		return
	}
	delta := totalPlayerFame - s.prevFame
	s.prevFame = totalPlayerFame
	if delta > 0 {
		s.FameTotal += delta
	}
}

// AccumulateSilver folds a new CurrentPlayerSilver reading into the
// session. Only positive deltas count — spending reduces wallet but
// shouldn't credit the session.
func (s *SessionStats) AccumulateSilver(currentPlayerSilver int64) {
	if !s.prevSilverKnown {
		s.prevSilver = currentPlayerSilver
		s.prevSilverKnown = true
		return
	}
	delta := currentPlayerSilver - s.prevSilver
	s.prevSilver = currentPlayerSilver
	if delta > 0 {
		s.SilverTotal += delta
	}
}

// AccumulateRespec adds a gained delta directly (the UpdateReSpec event
// already carries the delta, not a cumulative).
func (s *SessionStats) AccumulateRespec(gained int64) {
	if gained > 0 {
		s.RespecTotal += gained
	}
}

// AccumulateMight adds a gained might delta. MightAndFavorReceivedEvent
// fires per-gain (e.g. each daily quest, mob kill streak); param 1 is
// the gained amount in FixPoint internal units, not a lifetime running
// total. Same pattern as Respec.
func (s *SessionStats) AccumulateMight(gained int64) {
	if gained > 0 {
		s.MightTotal += gained
	}
}

// ElapsedSeconds returns the wall-clock time since session start.
func (s *SessionStats) ElapsedSeconds(now time.Time) float64 {
	if s.Start.IsZero() {
		return 0
	}
	d := now.Sub(s.Start)
	if d <= 0 {
		return 0
	}
	return d.Seconds()
}
