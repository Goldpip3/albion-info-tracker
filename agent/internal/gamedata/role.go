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
// with subtype suffixes). We match on the WEAPON token plus a few overrides
// for legendary/branded variants.
//
// Unknown weapons fall back to RoleUnknown so the meter still renders.
func ClassifyWeapon(uniqueName string) WeaponClassification {
	u := strings.ToUpper(uniqueName)
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

func cls(code string, role Role, label string) WeaponClassification {
	return WeaponClassification{Code: code, Role: role, Label: label}
}
