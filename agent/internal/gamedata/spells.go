package gamedata

import (
	"encoding/xml"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

// SpellEntry is a minimal view of one row from spells.bin — enough to render
// a spell's display name on a damage-meter sub-row.
type SpellEntry struct {
	Index      int    // numeric ID Photon HealthUpdate refers to
	UniqueName string // e.g. "FIREBALL_AOE"
	Category   string // "active", "passive", etc. (omitted if missing)
}

// SpellCatalog is an Index → SpellEntry lookup, built once at agent startup.
type SpellCatalog struct {
	byIndex map[int]SpellEntry
}

// Name returns the unique name for a spell index, or "" if unknown.
func (c *SpellCatalog) Name(index int) string {
	if c == nil {
		return ""
	}
	if e, ok := c.byIndex[index]; ok {
		return e.UniqueName
	}
	return ""
}

// LoadSpellCatalog reads spells.bin from the given Albion install and returns
// an indexed catalog. The .bin file is XML once decrypted; we walk every
// element looking for an `index` + `uniquename` attribute pair, which catches
// every spell-shaped row regardless of the wrapping tag name.
func LoadSpellCatalog(installRoot string, server ServerType) (*SpellCatalog, error) {
	binPath := filepath.Join(BinDir(installRoot, server), "spells.bin")
	xmlBytes, err := DecryptAndDecompress(binPath)
	if err != nil {
		return nil, err
	}
	cat := &SpellCatalog{byIndex: make(map[int]SpellEntry, 4096)}

	dec := xml.NewDecoder(strings.NewReader(string(xmlBytes)))
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		entry, ok := spellFromAttrs(start.Attr)
		if !ok {
			continue
		}
		if existing, dup := cat.byIndex[entry.Index]; dup && existing.UniqueName != "" {
			// Don't overwrite a real name with a sub-row's blank entry.
			if entry.UniqueName == "" {
				continue
			}
		}
		cat.byIndex[entry.Index] = entry
	}
	if len(cat.byIndex) == 0 {
		return nil, fmt.Errorf("spells.bin parsed but produced no entries (XML schema may have changed)")
	}
	return cat, nil
}

func spellFromAttrs(attrs []xml.Attr) (SpellEntry, bool) {
	var (
		e        SpellEntry
		gotIndex bool
		gotName  bool
	)
	for _, a := range attrs {
		switch strings.ToLower(a.Name.Local) {
		case "index":
			if n, err := strconv.Atoi(a.Value); err == nil {
				e.Index = n
				gotIndex = true
			}
		case "uniquename":
			e.UniqueName = a.Value
			gotName = true
		case "category":
			e.Category = a.Value
		}
	}
	return e, gotIndex && gotName
}
