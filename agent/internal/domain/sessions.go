package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Sessions persists completed sessions to local disk so the user can
// review what they did yesterday / last week without the agent staying
// up forever. Everything lives in %LocalAppData%\GDA\sessions on
// Windows (or $XDG_CACHE_HOME/gda/sessions on POSIX) — never leaves the
// machine, no cloud, no token-tied identity.
//
// File layout: one JSON file per session, named "<unix-millis>.json".
// The unix-millis filename makes lexical sort = chronological sort. We
// also expose deletion by filename so the user can prune the directory
// from the UI.

// ArchivedSession is the on-disk shape — a session's metadata + frozen
// final state. Big enough to render a "what was I doing last Tuesday"
// view; small enough that 1000 sessions = ~10 MB.
type ArchivedSession struct {
	// Id is the session's unix-millis filename token (no extension).
	// Stable forever — the web uses it as a delete key.
	Id          string    `json:"id"`
	StartedAt   time.Time `json:"startedAt"`
	EndedAt     time.Time `json:"endedAt"`
	DurationMs  int64     `json:"durationMs"`
	Zone        string    `json:"zone,omitempty"`
	LocalName   string    `json:"localName,omitempty"`
	FameTotal   int64     `json:"fameTotal"`
	SilverTotal int64     `json:"silverTotal"`
	RespecTotal int64     `json:"respecTotal"`
	MightTotal  int64     `json:"mightTotal"`
	DeathsTotal int       `json:"deathsTotal"`
	FightCount  int       `json:"fightCount"`
	Tag         string    `json:"tag,omitempty"`
}

// SessionsStore reads/writes ArchivedSession files in a fixed directory.
// Thread-safe; multiple agent goroutines can call concurrently. The
// process-wide cached list (cached) is the source of truth for the
// snapshot — files are scanned once on boot and updated incrementally.
type SessionsStore struct {
	dir    string
	mu     sync.Mutex
	cached []ArchivedSession
}

// NewSessionsStore opens (or creates) the sessions directory. Falls back
// to a no-op store when the OS doesn't expose a cache dir.
func NewSessionsStore() (*SessionsStore, error) {
	dir, err := defaultSessionsDir()
	if err != nil {
		return &SessionsStore{}, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return &SessionsStore{}, fmt.Errorf("mkdir %s: %w", dir, err)
	}
	s := &SessionsStore{dir: dir}
	if err := s.scan(); err != nil {
		return s, fmt.Errorf("scan %s: %w", dir, err)
	}
	return s, nil
}

func defaultSessionsDir() (string, error) {
	if cache, err := os.UserCacheDir(); err == nil && cache != "" {
		return filepath.Join(cache, "GDA", "sessions"), nil
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, ".gda", "sessions"), nil
	}
	return "", errors.New("no cache/home dir available")
}

// Dir returns the on-disk path. Useful for diagnostics + the UI footer.
func (s *SessionsStore) Dir() string {
	return s.dir
}

// List returns the cached metadata sorted newest-first. Safe to call
// from snapshot builders.
func (s *SessionsStore) List() []ArchivedSession {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]ArchivedSession, len(s.cached))
	copy(out, s.cached)
	return out
}

// Save writes a session to disk and updates the cache. Idempotent on Id.
func (s *SessionsStore) Save(sess ArchivedSession) error {
	if s.dir == "" {
		return errors.New("sessions store not initialised")
	}
	if sess.Id == "" {
		sess.Id = fmt.Sprintf("%d", sess.StartedAt.UnixMilli())
	}
	path := filepath.Join(s.dir, sess.Id+".json")
	b, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	// Replace existing or append.
	replaced := false
	for i, e := range s.cached {
		if e.Id == sess.Id {
			s.cached[i] = sess
			replaced = true
			break
		}
	}
	if !replaced {
		s.cached = append(s.cached, sess)
	}
	sort.Slice(s.cached, func(i, j int) bool { return s.cached[i].StartedAt.After(s.cached[j].StartedAt) })
	return nil
}

// Delete removes a session file by Id. Best-effort — missing files
// silently succeed so the UI can drive deletion without race checks.
func (s *SessionsStore) Delete(id string) error {
	if s.dir == "" {
		return errors.New("sessions store not initialised")
	}
	if id == "" || strings.ContainsAny(id, `/\.`) {
		return fmt.Errorf("invalid session id %q", id)
	}
	path := filepath.Join(s.dir, id+".json")
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove %s: %w", path, err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.cached[:0]
	for _, e := range s.cached {
		if e.Id != id {
			out = append(out, e)
		}
	}
	s.cached = out
	return nil
}

// scan rebuilds the in-memory cache from disk. Cheap — sessions are
// small JSON files and we usually have at most a few hundred of them.
func (s *SessionsStore) scan() error {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return err
	}
	out := make([]ArchivedSession, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		path := filepath.Join(s.dir, e.Name())
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var sess ArchivedSession
		if err := json.Unmarshal(b, &sess); err != nil {
			continue
		}
		if sess.Id == "" {
			sess.Id = strings.TrimSuffix(e.Name(), ".json")
		}
		out = append(out, sess)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StartedAt.After(out[j].StartedAt) })
	s.mu.Lock()
	s.cached = out
	s.mu.Unlock()
	return nil
}
