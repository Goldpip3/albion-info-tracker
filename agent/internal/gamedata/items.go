package gamedata

import (
	"encoding/xml"
	"fmt"
	"path/filepath"
	"strings"
)

// ItemEntry is one row from items.bin. We only care about the uniquename for
// role detection, but capture the slot too so the caller can sanity-check.
type ItemEntry struct {
	Index      int
	UniqueName string
	Tag        string // XML tag name (weapon / equipmentitem / consumableitem / …)
}

// ItemCatalog is an Index → ItemEntry lookup.
type ItemCatalog struct {
	byIndex map[int]ItemEntry
}

// Lookup returns the ItemEntry for an index, or the zero value if unknown.
func (c *ItemCatalog) Lookup(index int) ItemEntry {
	if c == nil {
		return ItemEntry{}
	}
	return c.byIndex[index]
}

// Name returns the uniquename for an item index, or "" if unknown.
func (c *ItemCatalog) Name(index int) string {
	if c == nil {
		return ""
	}
	if e, ok := c.byIndex[index]; ok {
		return e.UniqueName
	}
	return ""
}

// Len reports total entries loaded.
func (c *ItemCatalog) Len() int {
	if c == nil {
		return 0
	}
	return len(c.byIndex)
}

// itemKinds enumerates the XML tag names that count as "an item" — anything
// the server might assign an index to in the equipment slots we read.
// Order-derived indexing only walks these tags; the shopcategory metadata
// at the top of the file is skipped.
var itemKinds = map[string]bool{
	"weapon":               true,
	"equipmentitem":        true,
	"mount":                true,
	"consumableitem":       true,
	"consumablefrominventoryitem": true,
	"simpleitem":           true,
	"farmableitem":         true,
	"furnitureitem":        true,
	"journalitem":          true,
	"laboureritem":         true,
	"trackingitem":         true,
	"crystalleagueitem":    true,
	"siegebanneritem":      true,
	"killtrophyitem":       true,
	"mountskin":            true,
	"unlockable":           true,
}

// LoadItemCatalog decrypts items.bin and walks the XML, assigning a
// sequential index to every recognised item-kind element with a uniquename.
// The numeric ID in equipment slots of NewCharacter / CharacterEquipment
// events corresponds to this document-order index.
func LoadItemCatalog(installRoot string, server ServerType) (*ItemCatalog, error) {
	binPath := filepath.Join(BinDir(installRoot, server), "items.bin")
	xmlBytes, err := DecryptAndDecompress(binPath)
	if err != nil {
		return nil, err
	}
	cat := &ItemCatalog{byIndex: make(map[int]ItemEntry, 8192)}

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
		tag := strings.ToLower(start.Name.Local)
		if !itemKinds[tag] {
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
		cat.byIndex[idx] = ItemEntry{Index: idx, UniqueName: name, Tag: tag}
		idx++
	}
	if cat.Len() == 0 {
		return nil, fmt.Errorf("items.bin parsed but produced no entries")
	}
	return cat, nil
}
