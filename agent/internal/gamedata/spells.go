package gamedata

import (
	"encoding/xml"
	"fmt"
	"path/filepath"
	"strings"
)

// SpellEntry is a minimal view of one row from spells.bin — enough to render
// a spell's display name on a damage-meter sub-row.
type SpellEntry struct {
	Index      int    // numeric ID (assigned by document order; see LoadSpellCatalog)
	UniqueName string // e.g. "FIREBALL_AOE" — Albion's canonical identifier
	Kind       string // XML tag name: activespell, passivespell, etc.
}

// SpellCatalog is an Index → SpellEntry lookup, built once at agent startup.
type SpellCatalog struct {
	byIndex map[int]SpellEntry
	byName  map[string]int
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

// IndexOf returns the assigned numeric index for a uniquename, or -1.
func (c *SpellCatalog) IndexOf(name string) int {
	if c == nil {
		return -1
	}
	if i, ok := c.byName[name]; ok {
		return i
	}
	return -1
}

// Len reports how many spells were loaded.
func (c *SpellCatalog) Len() int {
	if c == nil {
		return 0
	}
	return len(c.byIndex)
}

// spellKinds enumerates the XML element names that count as "a spell"
// in spells.bin. Order-derived indexing only walks these elements;
// the colortag / comment / schema-declaration nodes at the top of the
// file are skipped.
var spellKinds = map[string]bool{
	"activespell":             true,
	"passivespell":            true,
	"targetedspell":           true,
	"selfspell":               true,
	"areaspell":               true,
	"projectilespell":         true,
	"channelspell":            true,
	"backgroundspell":         true,
	"interactivespell":        true,
	"persistentspell":         true,
	"toggleablespell":         true,
	"targetedchanneledspell":  true,
	"vectorprojectilespell":   true,
	"summonspell":             true,
	"resurrectspell":          true,
	"chargedspell":            true,
	"targetedchargedspell":    true,
	"summontemporaryspell":    true,
	"meleeattackspell":        true,
	"meleeattackchainspell":   true,
	"targetedaoespell":        true,
	"targetedreactionspell":   true,
	"reactionspell":           true,
	"escapereactionspell":     true,
	"vectortargetedaoespell":  true,
	"areadropdebuffspell":     true,
	"trapspell":               true,
	"manualtriggerspell":      true,
	"chainspell":              true,
	"summondropspell":         true,
	"persistentaurawhilemovingspell": true,
}

// LoadSpellCatalog decrypts spells.bin and walks the XML, assigning a
// sequential index to every element whose tag name is a known spell kind
// and that carries a uniquename attribute. Albion's CausingSpellIndex over
// the wire is *not* this document-order index in general — to map between
// them we'd need a snapshot from ao-bin-dumps or similar — but UniqueName
// remains the canonical identifier for display.
func LoadSpellCatalog(installRoot string, server ServerType) (*SpellCatalog, error) {
	binPath := filepath.Join(BinDir(installRoot, server), "spells.bin")
	xmlBytes, err := DecryptAndDecompress(binPath)
	if err != nil {
		return nil, err
	}
	cat := &SpellCatalog{
		byIndex: make(map[int]SpellEntry, 4096),
		byName:  make(map[string]int, 4096),
	}

	dec := xml.NewDecoder(strings.NewReader(string(xmlBytes)))
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
		kind := strings.ToLower(start.Name.Local)
		if !spellKinds[kind] {
			continue
		}
		var name string
		for _, a := range start.Attr {
			if strings.EqualFold(a.Name.Local, "uniquename") {
				name = a.Value
				break
			}
		}
		if name == "" {
			continue
		}
		entry := SpellEntry{Index: idx, UniqueName: name, Kind: kind}
		cat.byIndex[idx] = entry
		if _, exists := cat.byName[name]; !exists {
			cat.byName[name] = idx
		}
		idx++
	}
	if cat.Len() == 0 {
		return nil, fmt.Errorf("spells.bin parsed but produced no entries (XML schema may have changed)")
	}
	return cat, nil
}
