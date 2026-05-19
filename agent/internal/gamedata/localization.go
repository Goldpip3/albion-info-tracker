package gamedata

import (
	"encoding/xml"
	"fmt"
	"path/filepath"
	"strings"
)

// Localization is Albion's user-facing string table. It maps TMX tu-ids
// (e.g. "@ITEMS_T6_2H_CROSSBOW", "@SPELLS_CROSSBOW_FLICKERSHOT_E") to
// the localized display string ("Flickershot" / "Heavy Crossbow" / etc.).
//
// localization.bin is the same DES-CBC + gzip encoded XML as items.bin /
// spells.bin, in TMX format:
//
//   <tu tuid="@SPELLS_FOO">
//     <tuv xml:lang="EN-US"><seg>The Display Name</seg></tuv>
//     <tuv xml:lang="DE-DE"><seg>Der Anzeigename</seg></tuv>
//     ...
//   </tu>
//
// We keep only the EN-US strings — the rest of the UI is English-only
// today; if multi-language support comes later this becomes a [lang]name
// map instead of a flat lookup.
type Localization struct {
	byTuid map[string]string
}

// Name returns the localized name for a tu-id, or "" if unknown.
func (l *Localization) Name(tuid string) string {
	if l == nil {
		return ""
	}
	return l.byTuid[tuid]
}

// Len reports total tu entries loaded.
func (l *Localization) Len() int {
	if l == nil {
		return 0
	}
	return len(l.byTuid)
}

// SearchHit is one row of a Localization.Search result.
type SearchHit struct {
	Tuid  string
	Value string
}

// Search does a case-insensitive substring scan over the EN-US values.
// Used by the probe to figure out which @SPELLS_* tu-id corresponds to
// an in-game ability name like "Explosive Bolt".
func (l *Localization) Search(terms []string, limitPerTerm int) map[string][]SearchHit {
	out := make(map[string][]SearchHit, len(terms))
	if l == nil {
		return out
	}
	lower := make([]string, len(terms))
	for i, t := range terms {
		lower[i] = strings.ToLower(t)
	}
	for tuid, val := range l.byTuid {
		v := strings.ToLower(val)
		for i, t := range lower {
			if !strings.Contains(v, t) {
				continue
			}
			if len(out[terms[i]]) < limitPerTerm {
				out[terms[i]] = append(out[terms[i]], SearchHit{Tuid: tuid, Value: val})
			}
		}
	}
	return out
}

// ItemName resolves an items.bin uniquename to its in-game display name.
// "T6_2H_CROSSBOWLARGE_HELL@2" → "Boltcaster" (or similar — depends on the
// patch's localization table). Strips the "@<level>" suffix before lookup
// because the localization table only carries one entry per base item.
func (l *Localization) ItemName(uniqueName string) string {
	if l == nil || uniqueName == "" {
		return ""
	}
	base := uniqueName
	if i := strings.IndexByte(base, '@'); i >= 0 {
		base = base[:i]
	}
	return l.byTuid["@ITEMS_"+base]
}

// SpellName resolves a spells.bin uniquename to its in-game display name.
// Walks a fallback chain of prefix variants because Albion sprinkles spell
// tu-ids across multiple keyspaces:
//
//   @SPELLS_<U>            — the canonical 90% case ("BOLTSHOT" → "Explosive Bolt")
//   @SPELL_<U>             — older or one-off spells use the singular form
//   @MOB_ABILITIES_<U>     — mob abilities ("MOB_HER_DUALXBOW_ATTACK")
//   @SPELLDESC_<U>         — a handful of artifact spells live under desc keys
//   @ITEMS_<U>_SPELL       — some weapon-bound passives ("BOLTCASTER...")
//
// For each prefix variant, we also try a trailing _<digit> strip — Albion
// versions abilities with _1/_2/_3 suffixes that don't get separate
// localization rows. Returns "" if NONE of the variants matched. The
// caller (engine.go::localizedSpellName) decides whether to fall back to
// an override table or the prettifier.
func (l *Localization) SpellName(uniqueName string) string {
	if l == nil || uniqueName == "" {
		return ""
	}
	candidates := []string{uniqueName}
	if stripped := stripTrailingDigit(uniqueName); stripped != uniqueName {
		candidates = append(candidates, stripped)
	}
	prefixes := []string{
		"@SPELLS_",
		"@SPELL_",
		"@MOB_ABILITIES_",
		"@SPELLDESC_",
		"@ITEMS_%s_SPELL", // %s placeholder — special-cased below
	}
	for _, cand := range candidates {
		for _, pfx := range prefixes {
			var key string
			if strings.Contains(pfx, "%s") {
				key = strings.Replace(pfx, "%s", cand, 1)
			} else {
				key = pfx + cand
			}
			if name := l.byTuid[key]; name != "" {
				return name
			}
		}
	}
	return ""
}

// stripTrailingDigit removes a trailing "_<digit>" suffix. Used so that
// CROSSBOW_FLICKERSHOT_E_3 falls back to CROSSBOW_FLICKERSHOT_E in the
// localization lookup.
func stripTrailingDigit(s string) string {
	if len(s) < 3 || s[len(s)-2] != '_' {
		return s
	}
	last := s[len(s)-1]
	if last < '0' || last > '9' {
		return s
	}
	return s[:len(s)-2]
}

// LoadLocalization decrypts localization.bin and builds the lookup table.
// Reads only the EN-US <tuv> for each <tu>.
func LoadLocalization(installRoot string, server ServerType) (*Localization, error) {
	binPath := filepath.Join(BinDir(installRoot, server), "localization.bin")
	xmlBytes, err := DecryptAndDecompress(binPath)
	if err != nil {
		return nil, err
	}
	loc := &Localization{byTuid: make(map[string]string, 32768)}
	dec := xml.NewDecoder(strings.NewReader(string(xmlBytes)))

	var (
		curTuid     string
		curLang     string
		inSeg       bool
		segText     strings.Builder
		captureNext bool // set when we just entered a <seg> under an EN-US <tuv>
	)
	const wantLang = "EN-US"

	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "tu":
				curTuid = attr(t, "tuid")
			case "tuv":
				curLang = ""
				for _, a := range t.Attr {
					if strings.EqualFold(a.Name.Local, "lang") {
						curLang = a.Value
						break
					}
				}
				captureNext = strings.EqualFold(curLang, wantLang)
			case "seg":
				inSeg = true
				segText.Reset()
			}
		case xml.CharData:
			if inSeg {
				segText.Write(t)
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "seg":
				if inSeg && captureNext && curTuid != "" {
					loc.byTuid[curTuid] = segText.String()
				}
				inSeg = false
				captureNext = false
			case "tu":
				curTuid = ""
			case "tuv":
				curLang = ""
				captureNext = false
			}
		}
	}

	if loc.Len() == 0 {
		return nil, fmt.Errorf("localization.bin parsed but produced no entries")
	}
	return loc, nil
}
