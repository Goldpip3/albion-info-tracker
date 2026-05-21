// Package config loads agent runtime settings from a JSON file in the
// per-user GDA data directory (%LocalAppData%\GDA on Windows), with
// environment-variable overrides.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Config is the on-disk + env-overlaid agent configuration. All fields are
// optional; defaults apply when zero.
type Config struct {
	// PushURL is the WebSocket endpoint that receives snapshots, e.g.
	// "wss://albion-meter.example.workers.dev/ingest". Empty disables push
	// and keeps the agent in stdout-only mode.
	PushURL string `json:"pushUrl"`

	// PushToken is the bearer token included on the WebSocket upgrade.
	PushToken string `json:"pushToken"`

	// AlbionInstallRoot is the folder containing game/Albion-Online_Data/.
	// Used by the optional spell-name catalog loader.
	AlbionInstallRoot string `json:"albionInstallRoot,omitempty"`

	// AlwaysIncludeNames forces matching player names into the meter even
	// when they're not in the local player's guild and PartyJoined never
	// fired. Use for non-guild friends you party with regularly so they
	// survive the mid-zone party-detection fallback. Case-sensitive,
	// exact-match against the player Name as Albion ships it.
	AlwaysIncludeNames []string `json:"alwaysIncludeNames,omitempty"`
}

// Dir returns the per-user GDA data directory (%LocalAppData%\GDA on
// Windows, $XDG_CACHE_HOME/GDA or ~/.gda on POSIX). This is the same
// directory family the sessions and party stores use, so the token,
// sessions, and party roster all live together — and survive an in-place
// reinstall/update that wipes the program folder. The desktop shell reads
// the token from here too, so the dir must be the same whether the agent
// runs elevated or not (it is: same user → same %LocalAppData%).
func Dir() (string, error) {
	if cache, err := os.UserCacheDir(); err == nil && cache != "" {
		return filepath.Join(cache, "GDA"), nil
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, ".gda"), nil
	}
	return "", errors.New("no cache/home dir available")
}

// Path returns the agent.json location inside Dir().
func Path() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "agent.json"), nil
}

// Load reads agent.json from the GDA data dir, then applies env overrides.
// A missing file is not an error — an empty Config is returned. If no file
// exists at the new location but a legacy agent.json sits next to the
// executable (the pre-relocation layout), it is loaded and migrated to the
// new path so an existing token — and the browser pairing tied to it —
// survives the move.
func Load() (Config, error) {
	var cfg Config

	path, err := Path()
	if err != nil {
		return cfg, err
	}
	b, err := os.ReadFile(path)
	switch {
	case err == nil:
		if err := json.Unmarshal(b, &cfg); err != nil {
			return cfg, fmt.Errorf("parse %s: %w", path, err)
		}
	case errors.Is(err, os.ErrNotExist):
		// Nothing at the new location — try the legacy exe-dir file and
		// migrate it forward so we don't strand the user's token.
		if legacy, ok := loadLegacy(); ok {
			cfg = legacy
			if err := Save(cfg); err != nil {
				// Non-fatal: we still have the config in memory this run.
				fmt.Fprintf(os.Stderr, "warn: could not migrate agent.json to %s: %v\n", path, err)
			}
		}
	default:
		return cfg, fmt.Errorf("read %s: %w", path, err)
	}

	if v := os.Getenv("ALBION_AGENT_URL"); v != "" {
		cfg.PushURL = v
	}
	if v := os.Getenv("ALBION_AGENT_TOKEN"); v != "" {
		cfg.PushToken = v
	}
	if v := os.Getenv("ALBION_INSTALL"); v != "" {
		cfg.AlbionInstallRoot = v
	}
	return cfg, nil
}

// Save writes cfg to agent.json in the GDA data dir, creating the dir if
// needed.
func Save(cfg Config) error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "agent.json"), b, 0o644)
}

// loadLegacy reads agent.json from beside the executable (the layout used
// before the relocation to %LocalAppData%\GDA). Returns ok=false when the
// file is absent or unreadable.
func loadLegacy() (Config, bool) {
	var cfg Config
	exe, err := os.Executable()
	if err != nil {
		return cfg, false
	}
	b, err := os.ReadFile(filepath.Join(filepath.Dir(exe), "agent.json"))
	if err != nil {
		return cfg, false
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return cfg, false
	}
	return cfg, true
}
