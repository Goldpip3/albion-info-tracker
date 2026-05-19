package gamedata

import "strings"

// Role classifies a player's primary battlefield function. The 5-letter
// alphabet matches the composition strip in the UI (T·H·R·M·S).
type Role string

const (
	RoleTank      Role = "T"
	RoleHealer    Role = "H"
	RoleRangedDPS Role = "R"
	RoleMeleeDPS  Role = "M"
	RoleSupport   Role = "S"
	RoleControl   Role = "C"
	RoleUnknown   Role = "?"
)

// WeaponClassification is the (3-letter chip, role, full label) tuple
// rendered next to each player row.
type WeaponClassification struct {
	Code  string // 3-letter chip text (e.g. DGR)
	Role  Role
	Label string // "MELEE DPS · DAGGERS"
}

// ClassifyWeapon maps an items.bin uniquename to a chip + role + label.
// Albion's uniquenames follow the pattern Tn_SLOT_WEAPON_FAMILY (eventually
// with subtype suffixes for branded artifacts).
//
// Matching is most-specific-first: artifact variants (Boltcaster, Arlight
// Blaster, etc.) get their proper names; otherwise we fall through to the
// weapon family ("Crossbow"). Unknown weapons fall back to RoleUnknown so
// the meter still renders.
func ClassifyWeapon(uniqueName string) WeaponClassification {
	u := strings.ToUpper(uniqueName)
	switch {
	// ───── Artifact variants — match before the generic families ─────
	// Crossbow artifacts
	case contains(u, "HELLISH_CROSSBOW"), contains(u, "CROSSBOW_HELL"):
		return cls("XBW", RoleRangedDPS, "BOLTCASTER")
	case contains(u, "UNDEAD_CROSSBOW"), contains(u, "CROSSBOW_UNDEAD"):
		return cls("XBW", RoleRangedDPS, "WEEPING REPEATER")
	case contains(u, "MORGANA_CROSSBOW"), contains(u, "CROSSBOWLARGE_MORGANA"):
		return cls("XBW", RoleRangedDPS, "ENERGY SHAPER")
	case contains(u, "AVALON_CROSSBOW"), contains(u, "CROSSBOW_AVALON"):
		return cls("XBW", RoleRangedDPS, "ARLIGHT BLASTER")
	case contains(u, "CROSSBOWLARGE"):
		return cls("XBW", RoleRangedDPS, "HEAVY CROSSBOW")
	case contains(u, "CROSSBOWSMALL"):
		return cls("XBW", RoleRangedDPS, "LIGHT CROSSBOW")
	// Bow artifacts
	case contains(u, "HELLISH_BOW"), contains(u, "BOW_HELL"):
		return cls("BOW", RoleRangedDPS, "BOW OF BADON")
	case contains(u, "MORGANA_BOW"), contains(u, "BOW_MORGANA"):
		return cls("BOW", RoleRangedDPS, "MISTPIERCER")
	case contains(u, "UNDEAD_BOW"), contains(u, "BOW_UNDEAD"):
		return cls("BOW", RoleRangedDPS, "WAILING BOW")
	case contains(u, "AVALON_BOW"), contains(u, "BOW_AVALON"):
		return cls("BOW", RoleRangedDPS, "WHISPERING BOW")
	// Fire staff artifacts
	case contains(u, "MORGANA_FIRESTAFF"), contains(u, "FIRESTAFF_MORGANA"):
		return cls("FIR", RoleRangedDPS, "INFERNAL STAFF")
	case contains(u, "HELLISH_FIRESTAFF"), contains(u, "FIRESTAFF_HELL"):
		return cls("FIR", RoleRangedDPS, "WILDFIRE STAFF")
	case contains(u, "AVALON_FIRESTAFF"), contains(u, "FIRESTAFF_AVALON"):
		return cls("FIR", RoleRangedDPS, "DAWNSONG")
	// Frost staff artifacts
	case contains(u, "MORGANA_FROSTSTAFF"), contains(u, "FROSTSTAFF_MORGANA"):
		return cls("FRO", RoleControl, "PERMAFROST PRISM")
	case contains(u, "HELLISH_FROSTSTAFF"), contains(u, "FROSTSTAFF_HELL"):
		return cls("FRO", RoleControl, "CHILLHOWL")
	case contains(u, "AVALON_FROSTSTAFF"), contains(u, "FROSTSTAFF_AVALON"):
		return cls("FRO", RoleControl, "ICICLE STAFF")
	// Cursed staff artifacts
	case contains(u, "HELLISH_CURSEDSTAFF"), contains(u, "CURSEDSTAFF_HELL"):
		return cls("CRS", RoleRangedDPS, "DEMONIC STAFF")
	case contains(u, "UNDEAD_CURSEDSTAFF"), contains(u, "CURSEDSTAFF_UNDEAD"):
		return cls("CRS", RoleRangedDPS, "LIFECURSE STAFF")
	case contains(u, "AVALON_CURSEDSTAFF"), contains(u, "CURSEDSTAFF_AVALON"):
		return cls("CRS", RoleRangedDPS, "DAMNATION STAFF")
	// Holy staff artifacts
	case contains(u, "HELLISH_HOLYSTAFF"), contains(u, "HOLYSTAFF_HELL"):
		return cls("HLY", RoleHealer, "REDEMPTION STAFF")
	case contains(u, "MORGANA_HOLYSTAFF"), contains(u, "HOLYSTAFF_MORGANA"):
		return cls("HLY", RoleHealer, "FALLEN STAFF")
	case contains(u, "AVALON_HOLYSTAFF"), contains(u, "HOLYSTAFF_AVALON"):
		return cls("HLY", RoleHealer, "HALLOWFALL")
	// Nature staff artifacts
	case contains(u, "HELLISH_NATURESTAFF"), contains(u, "NATURESTAFF_HELL"):
		return cls("NTR", RoleHealer, "BLIGHT STAFF")
	case contains(u, "MORGANA_NATURESTAFF"), contains(u, "NATURESTAFF_MORGANA"):
		return cls("NTR", RoleHealer, "RAMPANT STAFF")
	case contains(u, "AVALON_NATURESTAFF"), contains(u, "NATURESTAFF_AVALON"):
		return cls("NTR", RoleHealer, "FORCE-STAFF OF BALANCE")
	// Arcane staff artifacts
	case contains(u, "HELLISH_ARCANESTAFF"), contains(u, "ARCANESTAFF_HELL"):
		return cls("ARC", RoleSupport, "ENIGMATIC STAFF")
	case contains(u, "MORGANA_ARCANESTAFF"), contains(u, "ARCANESTAFF_MORGANA"):
		return cls("ARC", RoleSupport, "MALEVOLENT LOCUS")
	case contains(u, "AVALON_ARCANESTAFF"), contains(u, "ARCANESTAFF_AVALON"):
		return cls("ARC", RoleSupport, "OCCULT STAFF")
	// Dagger artifacts
	case contains(u, "HELLISH_DAGGER"), contains(u, "DAGGER_HELL"):
		return cls("DGR", RoleMeleeDPS, "BLOODLETTER")
	case contains(u, "MORGANA_DAGGER"), contains(u, "DAGGER_MORGANA"):
		return cls("DGR", RoleMeleeDPS, "DEATHGIVERS")
	case contains(u, "AVALON_DAGGER"), contains(u, "DAGGER_AVALON"):
		return cls("DGR", RoleMeleeDPS, "BRIDLED FURY")
	// Sword artifacts
	case contains(u, "MORGANA_SWORD"), contains(u, "SWORD_MORGANA"):
		return cls("SWD", RoleMeleeDPS, "GALATINE PAIR")
	case contains(u, "HELLISH_SWORD"), contains(u, "SWORD_HELL"):
		return cls("SWD", RoleMeleeDPS, "CARVING SWORD")
	case contains(u, "AVALON_SWORD"), contains(u, "SWORD_AVALON"):
		return cls("SWD", RoleMeleeDPS, "KINGMAKER")
	// Axe artifacts
	case contains(u, "MORGANA_AXE"), contains(u, "AXE_MORGANA"):
		return cls("AXE", RoleMeleeDPS, "CARRIONCALLER")
	case contains(u, "HELLISH_AXE"), contains(u, "AXE_HELL"):
		return cls("AXE", RoleMeleeDPS, "INFERNAL SCYTHE")
	case contains(u, "AVALON_AXE"), contains(u, "AXE_AVALON"):
		return cls("AXE", RoleMeleeDPS, "REALMBREAKER")
	// Hammer artifacts
	case contains(u, "MORGANA_HAMMER"), contains(u, "HAMMER_MORGANA"):
		return cls("HAM", RoleTank, "CAMLANN MACE")
	case contains(u, "HELLISH_HAMMER"), contains(u, "HAMMER_HELL"):
		return cls("HAM", RoleTank, "GROVEKEEPER")
	case contains(u, "AVALON_HAMMER"), contains(u, "HAMMER_AVALON"):
		return cls("HAM", RoleTank, "FORGE HAMMERS")
	// Spear artifacts
	case contains(u, "MORGANA_SPEAR"), contains(u, "SPEAR_MORGANA"):
		return cls("SPR", RoleMeleeDPS, "TRINITY SPEAR")
	case contains(u, "HELLISH_SPEAR"), contains(u, "SPEAR_HELL"):
		return cls("SPR", RoleMeleeDPS, "INFERNO SPIKE")
	case contains(u, "AVALON_SPEAR"), contains(u, "SPEAR_AVALON"):
		return cls("SPR", RoleMeleeDPS, "SPIRITHUNTER")
	// Quarterstaff artifacts
	case contains(u, "MORGANA_QUARTERSTAFF"), contains(u, "QUARTERSTAFF_MORGANA"):
		return cls("QRT", RoleTank, "BLACK MONK STAVE")
	case contains(u, "HELLISH_QUARTERSTAFF"), contains(u, "QUARTERSTAFF_HELL"):
		return cls("QRT", RoleTank, "GRAILSEEKER")
	case contains(u, "AVALON_QUARTERSTAFF"), contains(u, "QUARTERSTAFF_AVALON"):
		return cls("QRT", RoleTank, "STAFF OF BALANCE")
	}
	// Fall through to generic family classification below.
	switch {
	// Healers
	case contains(u, "HOLYSTAFF"):
		return cls("HLY", RoleHealer, "HOLY STAFF")
	case contains(u, "NATURESTAFF"):
		return cls("NTR", RoleHealer, "NATURE STAFF")
	case contains(u, "DIVINESTAFF"):
		return cls("DVN", RoleHealer, "DIVINE STAFF")
	case contains(u, "WILDSTAFF"):
		return cls("WLD", RoleHealer, "WILD STAFF")
	case contains(u, "GREATHOLYSTAFF"):
		return cls("HLY", RoleHealer, "GREAT HOLY")
	case contains(u, "GREATNATURESTAFF"):
		return cls("NTR", RoleHealer, "GREAT NATURE")
	case contains(u, "FALLENSTAFF"):
		return cls("FAL", RoleHealer, "FALLEN STAFF")
	case contains(u, "REDEMPTIONSTAFF"):
		return cls("RED", RoleHealer, "REDEMPTION")

	// Tanks
	case contains(u, "HAMMER"):
		return cls("HAM", RoleTank, "HAMMER")
	case contains(u, "MACE"):
		return cls("MAC", RoleTank, "MACE")
	case contains(u, "QUARTERSTAFF"):
		return cls("QRT", RoleTank, "QUARTERSTAFF")
	case contains(u, "KNUCKLES"):
		return cls("KNK", RoleTank, "KNUCKLES")

	// Ranged DPS (bows, crossbows, magic staves)
	case contains(u, "LONGBOW"):
		return cls("LBW", RoleRangedDPS, "LONGBOW")
	case contains(u, "WARBOW"):
		return cls("WBW", RoleRangedDPS, "WARBOW")
	case contains(u, "BOW"):
		return cls("BOW", RoleRangedDPS, "BOW")
	case contains(u, "CROSSBOW"):
		return cls("XBW", RoleRangedDPS, "CROSSBOW")
	case contains(u, "FIRESTAFF"):
		return cls("FIR", RoleRangedDPS, "FIRE STAFF")
	case contains(u, "FROSTSTAFF"):
		return cls("FRO", RoleControl, "FROST STAFF")
	case contains(u, "CURSEDSTAFF"):
		return cls("CRS", RoleRangedDPS, "CURSED STAFF")
	case contains(u, "GREATFIRE"):
		return cls("FIR", RoleRangedDPS, "GREAT FIRE")
	case contains(u, "GREATFROST"):
		return cls("FRO", RoleControl, "GREAT FROST")
	case contains(u, "GREATCURSED"):
		return cls("CRS", RoleRangedDPS, "GREAT CURSED")

	// Support / utility magic
	case contains(u, "ARCANESTAFF"):
		return cls("ARC", RoleSupport, "ARCANE STAFF")
	case contains(u, "GREATARCANE"):
		return cls("ARC", RoleSupport, "GREAT ARCANE")

	// Melee DPS
	case contains(u, "DAGGERPAIR"), contains(u, "CLAWPAIR"), contains(u, "DAGGER"):
		return cls("DGR", RoleMeleeDPS, "DAGGERS")
	case contains(u, "GREATSWORD"):
		return cls("GRT", RoleMeleeDPS, "GREATSWORD")
	case contains(u, "BROADSWORD"), contains(u, "CLAYMORE"), contains(u, "SWORD"):
		return cls("SWD", RoleMeleeDPS, "SWORD")
	case contains(u, "AXE"), contains(u, "HALBERD"):
		return cls("AXE", RoleMeleeDPS, "AXE")
	case contains(u, "SPEAR"), contains(u, "PIKE"), contains(u, "GLAIVE"), contains(u, "TRINITYSPEAR"):
		return cls("SPR", RoleMeleeDPS, "SPEAR")
	}
	return cls("—", RoleUnknown, "")
}

func contains(s, sub string) bool {
	return strings.Contains(s, sub)
}

// InferClassFromSpell looks at a spell uniquename and guesses the caster's
// weapon class. Used as a fallback for the local player when Albion never
// fires a CharacterEquipmentChanged we can see — your own ability spells
// usually carry the weapon family in their uniquename (e.g. CROSSBOW_
// FLICKERSHOT, FROSTSTAFF_FROZENGROUND, NATURESTAFF_REJUVENATING_AURA).
// Returns the same classification ClassifyWeapon would for a matching
// weapon family. ok=false if nothing recognisable.
func InferClassFromSpell(spellName string) (WeaponClassification, bool) {
	if spellName == "" {
		return WeaponClassification{}, false
	}
	u := strings.ToUpper(spellName)
	c := ClassifyWeapon(u)
	if c.Role == RoleUnknown {
		return WeaponClassification{}, false
	}
	return c, true
}

func cls(code string, role Role, label string) WeaponClassification {
	return WeaponClassification{Code: code, Role: role, Label: label}
}
