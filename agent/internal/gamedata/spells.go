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
	Index      int    // numeric ID matching Albion's CausingSpellIndex over the wire
	UniqueName string // e.g. "BOLTSHOT" → "Explosive Bolt" in localization.bin
	Kind       string // "passivespell" / "activespell" / "togglespell"
	NameLocatag string // @SPELLS_FOO key for localization lookup, if present
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

// LocaTag returns the @SPELLS_… localization key for a spell index. Useful
// when the uniquename itself isn't in localization.bin but the namelocatag
// attribute on the spell element points at a different key.
func (c *SpellCatalog) LocaTag(index int) string {
	if c == nil {
		return ""
	}
	if e, ok := c.byIndex[index]; ok {
		return e.NameLocatag
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

// LoadSpellCatalog decrypts spells.bin and assigns each top-level spell
// element a 0-based document-order index that matches Albion's protocol-
// level CausingSpellIndex.
//
// The algorithm mirrors SAT's GameFileData.SpellData.BuildSpells:
//
//  1. Walk ONLY top-level children of the root element. Nested elements
//     (the spell-shape variants like <targetedspell>, <projectilespell>
//     inside an <activespell>) do NOT consume indices.
//  2. <colortag> at the top of the file is skipped without claiming
//     an index.
//  3. <passivespell uniquename="…"> claims one index.
//  4. <activespell uniquename="…"> claims one index. If it has a
//     <channelingspell> child, ALSO claim a SECOND consecutive index
//     using the same activespell's uniquename. (Albion uses two indices
//     to represent the cast and the channeling phase.)
//  5. <togglespell uniquename="…"> claims one index.
//
// Our prior loader walked every nested element through the recursive
// xml.Decoder.Token() stream and treated them as top-level — which
// inflated index assignment and made spell-name lookup return wrong
// uniquenames for high-tier abilities. "BOLTSHOT" (Explosive Bolt) is
// at a specific index Albion sends; without this fix, that index
// resolved to a totally unrelated nested spell shape.
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

	// Skip to inside the root element.
	for {
		tok, err := dec.Token()
		if err != nil {
			return nil, fmt.Errorf("spells.bin: %w", err)
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
		kind := strings.ToLower(start.Name.Local)
		if kind == "colortag" {
			_ = dec.Skip()
			continue
		}
		un := attr(start, "uniquename")
		nameTag := attr(start, "namelocatag")
		if un == "" {
			_ = dec.Skip()
			continue
		}
		// Claim an index for this top-level spell.
		cat.byIndex[idx] = SpellEntry{Index: idx, UniqueName: un, Kind: kind, NameLocatag: nameTag}
		if _, exists := cat.byName[un]; !exists {
			cat.byName[un] = idx
		}
		idx++

		// activespell with <channelingspell> child gets a second index
		// with the SAME uniquename. Inspect the subtree to detect this,
		// without re-emitting all the nested-element noise we just
		// stopped counting.
		hasChanneling := false
		if kind == "activespell" {
			hasChanneling, err = scanForChild(dec, "channelingspell")
			if err != nil {
				break
			}
		} else {
			if err := dec.Skip(); err != nil {
				break
			}
		}
		if hasChanneling {
			cat.byIndex[idx] = SpellEntry{Index: idx, UniqueName: un, Kind: kind, NameLocatag: nameTag}
			idx++
		}
	}
	if cat.Len() == 0 {
		return nil, fmt.Errorf("spells.bin parsed but produced no entries (XML schema may have changed)")
	}
	return cat, nil
}

// scanForChild consumes the rest of the current element's subtree,
// returning true if any direct child has the given tag name. Matches
// SAT's `element.Element("channelingspell") != null` check.
func scanForChild(dec *xml.Decoder, childTag string) (bool, error) {
	depth := 1
	found := false
	for depth > 0 {
		tok, err := dec.Token()
		if err != nil {
			return found, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			depth++
			if depth == 2 && strings.EqualFold(t.Name.Local, childTag) {
				found = true
			}
		case xml.EndElement:
			depth--
		}
	}
	return found, nil
}
