// Package config loads agent runtime settings from a JSON file next to
// the executable, with environment-variable overrides.
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

// Load reads agent.json from the executable's directory, then applies env
// overrides. A missing file is not an error — empty Config is returned.
func Load() (Config, error) {
	var cfg Config

	dir, err := exeDir()
	if err != nil {
		return cfg, err
	}
	path := filepath.Join(dir, "agent.json")
	b, err := os.ReadFile(path)
	switch {
	case err == nil:
		if err := json.Unmarshal(b, &cfg); err != nil {
			return cfg, fmt.Errorf("parse %s: %w", path, err)
		}
	case errors.Is(err, os.ErrNotExist):
		// no file — fine, env may still configure it
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

func exeDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(exe), nil
}
