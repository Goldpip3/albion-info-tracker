package domain

import (
	"path/filepath"
	"testing"
	"time"
)

// newTestDailyStore builds a DailyStore writing into a temp file so the
// test never touches the real %LocalAppData%\GDA\daily.json.
func newTestDailyStore(t *testing.T) *DailyStore {
	t.Helper()
	return &DailyStore{
		path: filepath.Join(t.TempDir(), "daily.json"),
		days: map[string]*DailyStat{},
	}
}

func TestDailyStoreAccumulatesPerDay(t *testing.T) {
	s := newTestDailyStore(t)

	// Two distinct local dates via injected timestamps.
	day1 := time.Date(2026, 5, 18, 14, 0, 0, 0, time.Local)
	day2 := time.Date(2026, 5, 19, 9, 0, 0, 0, time.Local)

	s.AddFame(100, day1)
	s.AddFame(50, day1)
	s.AddSilver(7_0000, day1) // 7 silver in FixPoint
	s.AddDeath(day1)

	s.AddFame(200, day2)
	s.AddMight(3_0000, day2)
	s.AddRespec(40, day2)

	got := s.List(0)
	if len(got) != 2 {
		t.Fatalf("want 2 day buckets, got %d", len(got))
	}
	// List is ascending by date.
	if got[0].Date != "2026-05-18" || got[1].Date != "2026-05-19" {
		t.Fatalf("unexpected ordering/dates: %q, %q", got[0].Date, got[1].Date)
	}
	if got[0].FameTotal != 150 {
		t.Errorf("day1 fame: want 150, got %d", got[0].FameTotal)
	}
	if got[0].SilverTotal != 7_0000 {
		t.Errorf("day1 silver: want 70000, got %d", got[0].SilverTotal)
	}
	if got[0].DeathsTotal != 1 {
		t.Errorf("day1 deaths: want 1, got %d", got[0].DeathsTotal)
	}
	if got[1].FameTotal != 200 {
		t.Errorf("day2 fame: want 200, got %d", got[1].FameTotal)
	}
	if got[1].MightTotal != 3_0000 {
		t.Errorf("day2 might: want 30000, got %d", got[1].MightTotal)
	}
	if got[1].RespecTotal != 40 {
		t.Errorf("day2 respec: want 40, got %d", got[1].RespecTotal)
	}
}

func TestDailyStoreRoundTripsThroughDisk(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "daily.json")

	s1 := &DailyStore{path: path, days: map[string]*DailyStat{}}
	day := time.Date(2026, 5, 20, 12, 0, 0, 0, time.Local)
	s1.AddFame(1234, day)
	s1.AddSilver(56_0000, day)
	s1.Flush()

	// Fresh store over the same file should reload the persisted day.
	s2 := &DailyStore{path: path, days: map[string]*DailyStat{}}
	s2.load()
	got := s2.List(0)
	if len(got) != 1 {
		t.Fatalf("want 1 reloaded day, got %d", len(got))
	}
	if got[0].FameTotal != 1234 || got[0].SilverTotal != 56_0000 {
		t.Errorf("reloaded totals wrong: fame=%d silver=%d", got[0].FameTotal, got[0].SilverTotal)
	}
}

func TestDailyStoreListCapsToMaxDays(t *testing.T) {
	s := newTestDailyStore(t)
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local)
	for i := 0; i < 10; i++ {
		s.AddFame(int64(i+1), base.AddDate(0, 0, i))
	}
	got := s.List(3)
	if len(got) != 3 {
		t.Fatalf("want 3 (capped), got %d", len(got))
	}
	// Cap keeps the most recent days.
	if got[len(got)-1].Date != "2026-01-10" {
		t.Errorf("want last day 2026-01-10, got %q", got[len(got)-1].Date)
	}
}
