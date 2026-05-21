package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// DailyStore accrues per-day economy totals (Fame / Silver / Combat Fame /
// Might / Deaths) into a single JSON file so day-over-day progress survives
// "New session" resets and multi-day runs — the per-session SessionStats
// alone can't show "what did I earn each day this week". Days are few, so
// unlike SessionsStore (one file per record) this keeps every day in one
// file and rewrites the whole thing on save.
//
// File: %LocalAppData%\GDA\daily.json on Windows ($XDG_CACHE_HOME/GDA on
// POSIX). Never leaves the machine. Same FixPoint convention as Session
// (10_000 internal units = 1 fame/silver/might); the UI divides for display.

// DailyStat is one calendar day's accumulated totals. Date is the LOCAL
// date ("2006-01-02") so a day boundary matches the player's wall clock.
type DailyStat struct {
	Date        string    `json:"date"`
	FameTotal   int64     `json:"fameTotal"`
	SilverTotal int64     `json:"silverTotal"`
	RespecTotal int64     `json:"respecTotal"`
	MightTotal  int64     `json:"mightTotal"`
	DeathsTotal int       `json:"deathsTotal"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// dailySaveThrottle bounds how often Add writes the file: a burst of gains
// coalesces into one write, but an idle period after a gain still flushes
// promptly (the first gain after a quiet stretch writes immediately).
const dailySaveThrottle = 10 * time.Second

// DailyStore reads/writes daily.json. Thread-safe.
type DailyStore struct {
	path     string
	mu       sync.Mutex
	days     map[string]*DailyStat
	dirty    bool
	lastSave time.Time
}

// NewDailyStore opens (or prepares) daily.json and loads any existing
// history. Falls back to an in-memory-only store when the OS exposes no
// cache/home dir.
func NewDailyStore() (*DailyStore, error) {
	path, err := defaultDailyPath()
	if err != nil {
		return &DailyStore{days: map[string]*DailyStat{}}, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return &DailyStore{days: map[string]*DailyStat{}}, fmt.Errorf("mkdir %s: %w", filepath.Dir(path), err)
	}
	s := &DailyStore{path: path, days: map[string]*DailyStat{}}
	s.load()
	return s, nil
}

func defaultDailyPath() (string, error) {
	if cache, err := os.UserCacheDir(); err == nil && cache != "" {
		return filepath.Join(cache, "GDA", "daily.json"), nil
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, ".gda", "daily.json"), nil
	}
	return "", errors.New("no cache/home dir available")
}

// Path returns the on-disk file path. Useful for the boot log.
func (s *DailyStore) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}

// AddFame credits fame to the current local day.
func (s *DailyStore) AddFame(n int64, now time.Time) {
	s.add(now, func(d *DailyStat) { d.FameTotal += n })
}

// AddSilver credits silver (FixPoint) to the current local day.
func (s *DailyStore) AddSilver(n int64, now time.Time) {
	s.add(now, func(d *DailyStat) { d.SilverTotal += n })
}

// AddRespec credits combat-fame ("respec") credits to the current day.
func (s *DailyStore) AddRespec(n int64, now time.Time) {
	s.add(now, func(d *DailyStat) { d.RespecTotal += n })
}

// AddMight credits might (FixPoint) to the current local day.
func (s *DailyStore) AddMight(n int64, now time.Time) {
	s.add(now, func(d *DailyStat) { d.MightTotal += n })
}

// AddDeath increments the death count for the current local day.
func (s *DailyStore) AddDeath(now time.Time) {
	s.add(now, func(d *DailyStat) { d.DeathsTotal++ })
}

// add locates (or creates) today's bucket, applies fn, and persists on a
// throttle. Ignores non-positive no-op fame/silver via the caller; here it
// just records whatever the closure changed.
func (s *DailyStore) add(now time.Time, fn func(*DailyStat)) {
	if s == nil || s.path == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	d := s.bucket(now)
	fn(d)
	d.UpdatedAt = now
	s.dirty = true
	if time.Since(s.lastSave) > dailySaveThrottle {
		s.save()
	}
}

// bucket returns today's stat, creating it on first gain. Caller holds mu.
func (s *DailyStore) bucket(now time.Time) *DailyStat {
	key := now.Local().Format("2006-01-02")
	d := s.days[key]
	if d == nil {
		d = &DailyStat{Date: key}
		s.days[key] = d
	}
	return d
}

// List returns the buckets sorted by date ascending, capped to the last
// maxDays (0 = all). Safe to call from the snapshot builder.
func (s *DailyStore) List(maxDays int) []DailyStat {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sortedLocked(maxDays)
}

// Flush forces a write if there are unsaved gains. Called on shutdown so
// the last throttle window isn't lost.
func (s *DailyStore) Flush() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.save()
}

// load reads daily.json into the in-memory map. A missing or malformed
// file yields an empty store (not an error) — daily tracking is best-effort.
func (s *DailyStore) load() {
	b, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	var list []DailyStat
	if err := json.Unmarshal(b, &list); err != nil {
		return
	}
	for i := range list {
		d := list[i]
		if d.Date == "" {
			continue
		}
		s.days[d.Date] = &d
	}
}

// save writes the whole history sorted by date. Caller holds mu. No-op
// when nothing changed.
func (s *DailyStore) save() {
	if s.path == "" || !s.dirty {
		return
	}
	list := s.sortedLocked(0)
	b, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return
	}
	if err := os.WriteFile(s.path, b, 0o644); err != nil {
		log.Printf("  daily save: %v", err)
		return
	}
	s.lastSave = time.Now()
	s.dirty = false
}

// sortedLocked materializes the map as a date-ascending slice, optionally
// capped to the last maxDays. Caller holds mu.
func (s *DailyStore) sortedLocked(maxDays int) []DailyStat {
	list := make([]DailyStat, 0, len(s.days))
	for _, d := range s.days {
		list = append(list, *d)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Date < list[j].Date })
	if maxDays > 0 && len(list) > maxDays {
		list = list[len(list)-maxDays:]
	}
	return list
}
