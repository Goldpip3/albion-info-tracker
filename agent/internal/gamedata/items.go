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

// LoadItemCatalog decrypts items.bin and assigns each item element a 1-based
// document-order index that matches Albion's protocol-level item IDs.
//
// The algorithm mirrors SAT's StatisticAnalysisTool.Extractor.ItemData:
//
//  1. Start idx = 1 (NOT 0).
//  2. Walk every top-level child of the root element. If it carries a
//     uniquename attribute, claim the current idx and increment.
//  3. If it has an enchantmentlevel attribute > 0, the stored name is
//     "<uniquename>@<level>".
//  4. If the element has an <enchantments> child, walk each <enchantment>
//     sub-element and claim a fresh idx for "<parent uniquename>@<level>".
//     This is where the bulk of the indices live — every T6+ weapon
//     produces 4 entries (base + 3 enchant levels).
//  5. After the main pass, every journalitem name claims two more indices
//     for "<name>_EMPTY" and "<name>_FULL", in that order.
//
// Without enchantment expansion our catalog topped out at ~5800 entries
// while Albion's actual indices reach into the 6500-7000 range. That's
// why item 6623 (a Tier 8 enchanted weapon) failed to resolve.
func LoadItemCatalog(installRoot string, server ServerType) (*ItemCatalog, error) {
	binPath := filepath.Join(BinDir(installRoot, server), "items.bin")
	xmlBytes, err := DecryptAndDecompress(binPath)
	if err != nil {
		return nil, err
	}
	cat := &ItemCatalog{byIndex: make(map[int]ItemEntry, 8192)}
	dec := xml.NewDecoder(strings.NewReader(string(xmlBytes)))

	// Skip until we land inside the root element. After this loop the next
	// Token() call returns the FIRST top-level item start tag.
	for {
		tok, err := dec.Token()
		if err != nil {
			return nil, fmt.Errorf("items.bin: %w", err)
		}
		if _, ok := tok.(xml.StartElement); ok {
			break
		}
	}

	idx := 1
	var journals []string

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
			// Either a non-item element or one without a uniquename. Skip
			// its subtree so we stay at top-level on the next Token().
			_ = dec.Skip()
			continue
		}
		tag := strings.ToLower(start.Name.Local)
		level := attr(start, "enchantmentlevel")
		name := un
		if level != "" && level != "0" {
			name = un + "@" + level
		}
		cat.byIndex[idx] = ItemEntry{Index: idx, UniqueName: name, Tag: tag}
		idx++
		if tag == "journalitem" {
			journals = append(journals, un)
		}
		// Walk this item's subtree looking for <enchantments>; index each
		// <enchantment> sub-element as "<parent uniquename>@<level>".
		idx = consumeItemSubtree(dec, un, idx, cat)
	}

	for _, un := range journals {
		cat.byIndex[idx] = ItemEntry{Index: idx, UniqueName: un + "_EMPTY", Tag: "journalitem"}
		idx++
		cat.byIndex[idx] = ItemEntry{Index: idx, UniqueName: un + "_FULL", Tag: "journalitem"}
		idx++
	}

	if cat.Len() == 0 {
		return nil, fmt.Errorf("items.bin parsed but produced no entries")
	}
	return cat, nil
}

// consumeItemSubtree drains the rest of the current top-level item's
// subtree, indexing any <enchantment> grandchildren of <enchantments> as
// "<parentUniqueName>@<level>". Returns the next idx to claim.
//
// The decoder enters this function having just consumed a top-level
// StartElement; we must consume tokens until that element's matching
// EndElement so the outer loop is back at top-level depth.
func consumeItemSubtree(dec *xml.Decoder, parentName string, idx int, cat *ItemCatalog) int {
	depth := 1 // inside the top-level item
	inEnchantments := false
	enchantDepth := 0
	for depth > 0 {
		tok, err := dec.Token()
		if err != nil {
			return idx
		}
		switch t := tok.(type) {
		case xml.StartElement:
			depth++
			tag := strings.ToLower(t.Name.Local)
			if inEnchantments {
				enchantDepth++
				// Direct child of <enchantments> — every such element is
				// one enchantment level. Level comes from the attribute.
				if enchantDepth == 1 {
					level := attr(t, "enchantmentlevel")
					if level == "" {
						level = "0"
					}
					cat.byIndex[idx] = ItemEntry{
						Index:      idx,
						UniqueName: parentName + "@" + level,
						Tag:        "enchantment",
					}
					idx++
				}
			} else if tag == "enchantments" {
				inEnchantments = true
				enchantDepth = 0
			}
		case xml.EndElement:
			if inEnchantments {
				if enchantDepth > 0 {
					enchantDepth--
				} else {
					inEnchantments = false
				}
			}
			depth--
		}
	}
	return idx
}

// attr returns the value of the named attribute (case-insensitive), or "".
func attr(start xml.StartElement, name string) string {
	for _, a := range start.Attr {
		if strings.EqualFold(a.Name.Local, name) {
			return a.Value
		}
	}
	return ""
}
