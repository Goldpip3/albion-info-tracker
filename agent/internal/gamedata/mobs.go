package gamedata

import (
	"encoding/xml"
	"fmt"
	"path/filepath"
	"strings"
)

// MobEntry is one row from mobs.bin — enough to put a friendly name on
// a target in the drill-in.
type MobEntry struct {
	Index      int
	UniqueName string
	NameTag    string // localization key (typically @MOB_<uniquename> when present)
	Tier       int    // 1..8 when parseable, 0 otherwise
}

// MobCatalog is an Index → MobEntry lookup. Same shape as ItemCatalog.
type MobCatalog struct {
	byIndex map[int]MobEntry
}

// Lookup returns the MobEntry for a protocol-level mob index, or the
// zero value if unknown.
//
// IMPORTANT: SAT subtracts 15 from the in-game MobIndex before doing
// the lookup ("From July 18, 2025, the in-game index will start
// counting from 15. The IDs were decreased by 15."). We match that
// shift here so callers can pass the raw param value from NewMob.
func (c *MobCatalog) Lookup(rawIndex int) MobEntry {
	if c == nil {
		return MobEntry{}
	}
	adjusted := rawIndex - 15
	if adjusted < 0 {
		return MobEntry{}
	}
	return c.byIndex[adjusted]
}

// Name returns the localized English display name (via the provided
// Localization). Falls back to a Title-Cased uniquename when localization
// is missing or doesn't carry the @MOB_<uniquename> key.
//
// Pass loc=nil if localization isn't wired yet — you'll get the
// prettified uniquename in that case.
func (c *MobCatalog) Name(rawIndex int, loc *Localization) string {
	entry := c.Lookup(rawIndex)
	if entry.UniqueName == "" {
		return ""
	}
	if loc != nil {
		key := entry.NameTag
		if key == "" {
			key = "@MOB_" + entry.UniqueName
		}
		if v := loc.byTuid[key]; v != "" {
			return v
		}
	}
	return prettifyMobName(entry.UniqueName)
}

// Len reports the number of entries.
func (c *MobCatalog) Len() int {
	if c == nil {
		return 0
	}
	return len(c.byIndex)
}

// LoadMobCatalog decrypts mobs.bin and walks the XML, assigning a
// 0-based document-order index to every top-level element that carries a
// uniquename attribute. Mirrors the simple structure of items.bin's
// top level (no enchantment / journal post-processing required here —
// each mob is one entry).
func LoadMobCatalog(installRoot string, server ServerType) (*MobCatalog, error) {
	binPath := filepath.Join(BinDir(installRoot, server), "mobs.bin")
	xmlBytes, err := DecryptAndDecompress(binPath)
	if err != nil {
		return nil, err
	}
	cat := &MobCatalog{byIndex: make(map[int]MobEntry, 4096)}
	dec := xml.NewDecoder(strings.NewReader(string(xmlBytes)))

	// Skip to inside the root element.
	for {
		tok, err := dec.Token()
		if err != nil {
			return nil, fmt.Errorf("mobs.bin: %w", err)
		}
		if _, ok := tok.(xml.StartElement); ok {
			break
		}
	}

	idx := 0
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		un := attr(start, "uniquename")
		if un == "" {
			_ = dec.Skip()
			continue
		}
		tier := parseTier(un)
		nameTag := attr(start, "namelocatag")
		cat.byIndex[idx] = MobEntry{
			Index:      idx,
			UniqueName: un,
			NameTag:    nameTag,
			Tier:       tier,
		}
		idx++
		// Mobs may carry nested children (spells, drops, …) — skip them;
		// we only need the top-level metadata for naming.
		_ = dec.Skip()
	}

	if cat.Len() == 0 {
		return nil, fmt.Errorf("mobs.bin parsed but produced no entries")
	}
	return cat, nil
}

// parseTier extracts the leading T<N>_ tier digit from a uniquename
// when present (e.g. "T5_MOB_FOREST_FOX" → 5). Returns 0 if absent.
func parseTier(uniqueName string) int {
	if len(uniqueName) < 3 || uniqueName[0] != 'T' {
		return 0
	}
	d := int(uniqueName[1] - '0')
	if d < 1 || d > 9 {
		return 0
	}
	return d
}

// prettifyMobName turns "T5_MOB_FOREST_GREYWOLF_FARMED" into
// "Greywolf" — strips T<N>_, MOB_, common biome / state suffixes, and
// Title-Cases the surviving token. Loses fidelity vs the localized
// name but never produces "underscore garbage" output.
func prettifyMobName(uniqueName string) string {
	if uniqueName == "" {
		return ""
	}
	u := strings.ToUpper(uniqueName)
	// Strip leading tier marker.
	if len(u) >= 3 && u[0] == 'T' && u[2] == '_' {
		u = u[3:]
	}
	// Strip MOB_ prefix.
	u = strings.TrimPrefix(u, "MOB_")
	// Strip common biome / variant tokens — they're noise for naming.
	stripTokens := []string{
		"FOREST_", "STEPPE_", "MOUNTAIN_", "HIGHLAND_", "SWAMP_",
		"BLACK_", "AVALON_", "MIST_", "MISTS_",
		"_FARMED", "_BABY", "_PLAYER", "_BOSS", "_MINIBOSS",
		"_T1", "_T2", "_T3", "_T4", "_T5", "_T6", "_T7", "_T8",
	}
	for _, t := range stripTokens {
		u = strings.ReplaceAll(u, t, "")
	}
	// Collapse double underscores left behind by the strips.
	for strings.Contains(u, "__") {
		u = strings.ReplaceAll(u, "__", "_")
	}
	u = strings.Trim(u, "_")
	if u == "" {
		return uniqueName
	}
	parts := strings.Split(u, "_")
	for i, p := range parts {
		parts[i] = titleCase(p)
	}
	return strings.Join(parts, " ")
}
