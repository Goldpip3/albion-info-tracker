package gamedata

import "strings"

// spellOverrides is a hand-curated lookup for spell uniquenames that
// don't have a usable localization.bin entry. Seeded from in-game user
// reports — abilities the meter rendered as garbage Title-Case strings
// like "Curse skeleton barf fdhr" because their tu-ids genuinely don't
// exist in Albion's English translation table.
//
// Keys are normalized: the part of the uniquename remaining AFTER
// stripWeaponPrefix + stripSlotSuffix has stripped the weapon-family
// prefix ("CROSSBOW_", "DAGGER_", "2H_HAMMER_", …) and the trailing
// slot/grade tokens ("_E", "_Q", "_W", "_PASSIVE", "_1", "_2", "_3").
//
// Values are the user-facing English names Albion uses in tooltips.
//
// If you spot a spell rendering as a Title-Case underscore string in
// the meter, add an entry here, rebuild the agent, restart, and verify.
var spellOverrides = map[string]string{
	"CALTROPS":      "Caltrops",
	"FLICKERSHOT":   "Flickershot",
	"FROST_SHOT":    "Frost Shot",
	"FROSTSHOT":     "Frost Shot",
	"ENERGY_BOLT":   "Energy Bolt",
	"ENERGYBOLT":    "Energy Bolt",
	"AIMED_SHOT":    "Aimed Shot",
	"AIMEDSHOT":     "Aimed Shot",
	"EXPLOSIVE_ARROWS": "Explosive Arrows",
	"BOLTSHOT":      "Explosive Bolt", // also resolved via @SPELLS_BOLTSHOT, but cheap belt-and-suspenders.
	"CHAINDASH":     "Chain Slash",
	"BLOODLUST":     "Bloodlust",
	"SILENCING_BOLT": "Silencing Bolt",
	"SILENCINGBOLT": "Silencing Bolt",
	"SNIPE_SHOT":    "Snipe Shot",
	"SNIPESHOT":     "Snipe Shot",
	"AUTO_FIRE":     "Auto Fire",
	"AUTOFIRE":      "Auto Fire",
	"ARMOR_PIERCER": "Armor Piercer",
	"ARMORPIERCER":  "Armor Piercer",
}

// SpellOverride returns the in-game English name for a spell uniquename
// when localization.bin doesn't carry an entry. Normalization mirrors
// the prettifier so adding entries is intuitive: just key on the
// "interesting" middle of the name.
//
// Returns "" when there's no override.
func SpellOverride(uniqueName string) string {
	if uniqueName == "" {
		return ""
	}
	normalized := stripSlotSuffix(stripWeaponPrefix(strings.ToUpper(uniqueName)))
	if name, ok := spellOverrides[normalized]; ok {
		return name
	}
	return ""
}

// PrettifySpell is the final-line-of-defence renderer used when the
// localization table and the override table both miss. It produces a
// Title-Case string like "Flickershot" from a uniquename like
// "CROSSBOW_FLICKERSHOT_E". Mirrors the format you'd expect a player
// to see for an unrecognised ability without the obvious underscore
// noise.
//
// The result is best-effort — it's a graceful degradation, not a
// canonical name. The override table catches the cases where the
// pretty form would be wrong.
func PrettifySpell(uniqueName string) string {
	if uniqueName == "" {
		return ""
	}
	work := strings.ToUpper(uniqueName)
	work = stripWeaponPrefix(work)
	work = stripSlotSuffix(work)
	if work == "" {
		return uniqueName // give up: rather show the raw than nothing
	}
	parts := strings.Split(work, "_")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p == "" {
			continue
		}
		// Compound-word substitutes — Albion concatenates pieces that
		// the prettifier can't split heuristically. Look the token up
		// in the compound map; if it's there, emit the pre-formatted
		// multi-word replacement instead of title-casing the slug.
		if replacement, ok := compoundWords[p]; ok {
			out = append(out, replacement)
			continue
		}
		out = append(out, titleCase(p))
	}
	return strings.Join(out, " ")
}

// compoundWords maps concatenated tokens Albion ships in uniquenames
// to their pre-formatted display form. Keep keys uppercase to match
// the prettifier's working casing. Values are the strings the user
// should actually see — already title-cased and space-separated.
//
// Bias toward the most common offenders from in-the-wild loadouts;
// extend as more passives surface that fall through to the prettifier.
//
// TODO: extend as more passives show up in the wild.
var compoundWords = map[string]string{
	// Armor & equipment compounds.
	"PLATEARMOR":   "Plate Armor",
	"LEATHERARMOR": "Leather Armor",
	"CLOTHARMOR":   "Cloth Armor",
	"HEALTHCHANCE": "Health Chance",
	"ARMORCHANCE":  "Armor Chance",
	"SPELLPOWER":   "Spell Power",
	"ATTACKPOWER":  "Attack Power",
	"ATTACKBUFF":   "Attack Buff",
	"ATTACKSPEED":  "Attack Speed",
	"HEALTHREDUCTION": "Health Reduction",
	"MOVESPEED":    "Move Speed",
	"CASTSPEED":    "Cast Speed",
	"CRITCHANCE":   "Crit Chance",
	"MAGICRESIST":  "Magic Resist",
	"PHYSICALRESIST": "Physical Resist",
	// Albion city / capital names — usually appear as cape passives.
	"BRIDGEWATCH":  "Bridgewatch",
	"MARTLOCK":     "Martlock",
	"THETFORD":     "Thetford",
	"FORTSTERLING": "Fort Sterling",
	"LYMHURST":     "Lymhurst",
	"CAERLEON":     "Caerleon",
	"BRECILIEN":    "Brecilien",
}

// weaponPrefixes is the sorted-by-length list of family tokens we
// recognise at the start of a spell uniquename. Sorted longest-first
// so "2H_HAMMER_" wins over "HAMMER_" and we don't accidentally chop
// a shorter prefix off a longer one.
//
// Keep in rough sync with the cases in role.go::ClassifyWeapon — when
// you add a new weapon there, add its family prefix here too.
var weaponPrefixes = []string{
	"2H_HAMMER_", "2H_DUALCROSSBOW_", "2H_CROSSBOW_",
	"HOLYSTAFF_", "NATURESTAFF_", "DIVINESTAFF_", "WILDSTAFF_",
	"GREATHOLYSTAFF_", "GREATNATURESTAFF_",
	"FIRESTAFF_", "FROSTSTAFF_", "CURSEDSTAFF_", "ARCANESTAFF_",
	"GREATFIRE_", "GREATFROST_", "GREATCURSED_", "GREATARCANE_",
	"BOLTCASTER_", "DUALCROSSBOW_", "CROSSBOW_",
	"LONGBOW_", "WARBOW_", "BOW_", "XBW_",
	"QUARTERSTAFF_", "HAMMER_", "MACE_", "KNUCKLES_",
	"DAGGERPAIR_", "CLAWPAIR_", "DAGGER_",
	"GREATSWORD_", "BROADSWORD_", "CLAYMORE_", "SWORD_",
	"HALBERD_", "AXE_",
	"GLAIVE_", "PIKE_", "TRINITYSPEAR_", "SPEAR_",
}

// slotSuffixes are the trailing tokens we trim off a uniquename
// before display. _E/_Q/_W/_R are ability slot keys, _PASSIVE is a
// trigger marker, _1/_2/_3/_4 are spell versions Albion ships
// alongside the canonical entry.
var slotSuffixes = []string{
	"_PASSIVE", "_E", "_Q", "_W", "_R",
	"_1", "_2", "_3", "_4", "_5",
}

func stripWeaponPrefix(u string) string {
	for _, p := range weaponPrefixes {
		if strings.HasPrefix(u, p) {
			return u[len(p):]
		}
	}
	return u
}

func stripSlotSuffix(u string) string {
	for {
		stripped := false
		for _, s := range slotSuffixes {
			if strings.HasSuffix(u, s) {
				u = u[:len(u)-len(s)]
				stripped = true
				break
			}
		}
		if !stripped {
			return u
		}
	}
}

// titleCase returns "Caltrops" for "CALTROPS". Only the first letter is
// capitalised — Albion's tooltip names are sentence case, not Camel.
func titleCase(s string) string {
	if s == "" {
		return s
	}
	b := []byte(strings.ToLower(s))
	if b[0] >= 'a' && b[0] <= 'z' {
		b[0] -= 32
	}
	return string(b)
}
