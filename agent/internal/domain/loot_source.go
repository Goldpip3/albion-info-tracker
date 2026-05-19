package domain

import "strings"

// prettifyLootSource turns Albion's raw @MOB_<id>_<thing> /
// @CHEST_<id>_<thing> localisation key into something the user can
// read at a glance. It's a placeholder until mobs.bin localisation
// lookup ships — once that lands, the engine should resolve mob
// ObjectIds to the localised display name instead of leaning on the
// uniquename. Until then, this gets us out of "@MOB_T6_HARVESTER_
// PLAYERSPAWN" territory and into "T6 Harvester".
//
// Strategy:
//   1. Strip the leading "@" and the family prefix ("MOB_", "CHEST_",
//      "DUNGEON_", "ENCOUNTER_").
//   2. Drop spawn-mode suffixes that mean "this is the version used
//      for player-triggered spawns" — they carry no display value.
//   3. Replace underscores with spaces.
//   4. Title-case each token, but keep tier markers ("T4", "T6", "T8")
//      uppercase and short roman tokens like "II" / "III" intact.
//
// Empty inputs round-trip as "" so the loot row falls through to
// whatever the UI's fallback is.
//
// TODO: switch to a real MobCatalog.Name(ObjectId) lookup once the
// agent's NewMob handler records ObjectId → localised name (Part B of
// PROMPT_followup_fixes.md).
func prettifyLootSource(s string) string {
	if s == "" {
		return ""
	}
	work := strings.TrimPrefix(s, "@")
	work = strings.ToUpper(work)

	for _, p := range lootSourcePrefixes {
		if strings.HasPrefix(work, p) {
			work = work[len(p):]
			break
		}
	}
	for {
		stripped := false
		for _, suf := range lootSourceSuffixes {
			if strings.HasSuffix(work, suf) {
				work = work[:len(work)-len(suf)]
				stripped = true
				break
			}
		}
		if !stripped {
			break
		}
	}

	parts := strings.Split(work, "_")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p == "" {
			continue
		}
		out = append(out, titleCaseSourceToken(p))
	}
	if len(out) == 0 {
		return s
	}
	return strings.Join(out, " ")
}

// lootSourcePrefixes covers the family tokens Albion uses to namespace
// mob / chest / encounter ids. Sorted longest-first so longer prefixes
// win over shorter ones.
var lootSourcePrefixes = []string{
	"DUNGEON_", "ENCOUNTER_", "CHEST_", "MOB_",
}

// lootSourceSuffixes are spawn-mode markers Albion appends. None of
// them add display value — they tell the server which spawn variant
// emitted the entity.
var lootSourceSuffixes = []string{
	"_HARVESTER_PLAYERSPAWN",
	"_PLAYERMOBSPAWN",
	"_PLAYERSPAWN",
	"_HARVESTER",
	"_BOSS",
	"_MINIBOSS",
	"_ELITE",
	"_VETERAN",
}

func titleCaseSourceToken(s string) string {
	if s == "" {
		return s
	}
	// Keep tier markers (T1..T8) and short roman numerals uppercase.
	if isTierToken(s) || isRomanToken(s) {
		return s
	}
	b := []byte(strings.ToLower(s))
	if b[0] >= 'a' && b[0] <= 'z' {
		b[0] -= 32
	}
	return string(b)
}

func isTierToken(s string) bool {
	if len(s) != 2 {
		return false
	}
	if s[0] != 'T' {
		return false
	}
	return s[1] >= '0' && s[1] <= '9'
}

func isRomanToken(s string) bool {
	if len(s) == 0 || len(s) > 4 {
		return false
	}
	for _, r := range s {
		switch r {
		case 'I', 'V', 'X', 'L':
			// ok
		default:
			return false
		}
	}
	return true
}
