package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// PartyStore persists the confirmed party roster to a single JSON file
// so it survives an agent restart. Albion only emits PartyJoined /
// PartyPlayerJoined at the moment a member joins — never retroactively
// — so if the agent is started or rebuilt-and-relaunched while already
// grouped, the roster is otherwise lost. Saving it here (and on every
// manual add/remove) lets RestoreParty rehydrate it on boot.
//
// File: %LocalAppData%\GDA\party.json on Windows ($XDG_CACHE_HOME/GDA
// on POSIX) — same dir family as the sessions store. Never leaves the
// machine.

// PartyRef is the minimal identity persisted per member. Name is a
// convenience for the UI before the entity rebinds to live combat.
type PartyRef struct {
	Guid string `json:"guid"`
	Name string `json:"name,omitempty"`
}

// partyFile is the persisted shape: a save timestamp + the roster. The
// timestamp drives the freshness gate so a roster from days ago doesn't
// resurrect a party the user has long since left.
type partyFile struct {
	SavedAt time.Time  `json:"savedAt"`
	Members []PartyRef `json:"members"`
}

// partyFreshness bounds how long a persisted roster is trusted. Kept
// short (30 min) so a quick rebuild-and-relaunch restores your group,
// but coming back later solo doesn't resurrect an old roster. The
// manual "Clear party" control wipes it immediately regardless.
const partyFreshness = 30 * time.Minute

// PartyStore reads/writes the single party.json file. Thread-safe.
type PartyStore struct {
	path string
	mu   sync.Mutex
}

// NewPartyStore opens (or prepares) the party-roster file location.
// Falls back to a no-op store when the OS exposes no cache/home dir.
func NewPartyStore() (*PartyStore, error) {
	dir, err := defaultPartyDir()
	if err != nil {
		return &PartyStore{}, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return &PartyStore{}, fmt.Errorf("mkdir %s: %w", dir, err)
	}
	return &PartyStore{path: filepath.Join(dir, "party.json")}, nil
}

func defaultPartyDir() (string, error) {
	if cache, err := os.UserCacheDir(); err == nil && cache != "" {
		return filepath.Join(cache, "GDA"), nil
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, ".gda"), nil
	}
	return "", errors.New("no cache/home dir available")
}

// Path returns the on-disk file path. Useful for the boot log.
func (p *PartyStore) Path() string { return p.path }

// Save writes the roster with a fresh timestamp. Empty rosters still
// write (so a disband clears via Save([]) just as well as Clear()).
func (p *PartyStore) Save(members []PartyRef) error {
	if p.path == "" {
		return errors.New("party store not initialised")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	b, err := json.MarshalIndent(partyFile{SavedAt: time.Now(), Members: members}, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal party: %w", err)
	}
	if err := os.WriteFile(p.path, b, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", p.path, err)
	}
	return nil
}

// Load returns the persisted roster, but only when it's within
// partyFreshness of now. A stale file yields an empty slice (and no
// error) so callers don't resurrect an old party.
func (p *PartyStore) Load() ([]PartyRef, error) {
	if p.path == "" {
		return nil, errors.New("party store not initialised")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	b, err := os.ReadFile(p.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", p.path, err)
	}
	var pf partyFile
	if err := json.Unmarshal(b, &pf); err != nil {
		return nil, fmt.Errorf("parse %s: %w", p.path, err)
	}
	if time.Since(pf.SavedAt) > partyFreshness {
		return nil, nil
	}
	return pf.Members, nil
}

// Clear removes the persisted roster. Best-effort — a missing file
// counts as success so callers (e.g. PartyDisbanded) need no guard.
func (p *PartyStore) Clear() error {
	if p.path == "" {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := os.Remove(p.path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove %s: %w", p.path, err)
	}
	return nil
}
