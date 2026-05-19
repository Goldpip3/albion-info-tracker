package gamedata

import (
	"fmt"
	"os"
	"path/filepath"
)

// ServerType selects which Albion GameData subdirectory to read from.
type ServerType string

const (
	ServerLive       ServerType = "game"
	ServerStaging    ServerType = "staging"
	ServerPlayground ServerType = "playground"
)

// BinDir returns the directory containing items.bin/spells.bin/etc. inside
// an Albion install. Path layout matches SAT's ExtractorUtilities.
func BinDir(installRoot string, server ServerType) string {
	return filepath.Join(installRoot, string(server), "Albion-Online_Data", "StreamingAssets", "GameData")
}

// IsValidInstall checks that the canonical .bin files are present.
func IsValidInstall(installRoot string, server ServerType) error {
	bin := BinDir(installRoot, server)
	for _, name := range []string{"items.bin", "spells.bin", "mobs.bin"} {
		p := filepath.Join(bin, name)
		if _, err := os.Stat(p); err != nil {
			return fmt.Errorf("%s: %w", p, err)
		}
	}
	return nil
}
