package gamedata

import (
	"strings"
)

// Item Power computation for an equipped item.
//
// Sources: Albion Online Wiki "Item Power" page and community tools.
// The exact formula varies by patch, so all the magic numbers live as
// package vars at the top of this file — easy to tune without rewriting
// the parser.
//
// Tier base climbs by 100 per tier from T2 (500) up to T8 (1100).
// Enchantment adds 100 per level (uniquename suffix "@1" / "@2" / "@3" /
// "@4"). Quality adds 0–100 depending on grade. Artifact / Avalonian
// items get a type bonus on top.
//
// We don't try to reproduce Albion's exact server formula — that
// considers diminishing returns and a few other curves — but the
// "average IP" number you see in your equipment screen tracks this to
// within a few percent across the gear range.

// IPBaseByTier maps the leading "T<N>_" tier digit in a uniquename to
// the base IP at that tier. Missing tiers yield 0.
var IPBaseByTier = map[int]int{
	2: 500, 3: 600, 4: 700, 5: 800, 6: 900, 7: 1000, 8: 1100,
}

// IPEnchantBonus is the per-level IP added for each "@<level>" suffix.
var IPEnchantBonus = 100

// IPQualityBonus maps Albion's quality byte (1..5) to its IP addend.
// 1=Normal, 2=Good, 3=Outstanding, 4=Excellent, 5=Masterpiece.
var IPQualityBonus = map[int]int{
	1: 0, 2: 10, 3: 20, 4: 50, 5: 100,
}

// IPTypeBonus catches artifact / Avalonian variants. We match on the
// uppercased uniquename: AVALON gets the highest bump, then named
// artifacts (HELL / UNDEAD / MORGANA / KEEPER) get a smaller one.
var IPTypeBonus = []struct {
	Substr string
	Bonus  int
}{
	{"_AVALON", 200},
	{"_HELL", 100},
	{"_UNDEAD", 100},
	{"_MORGANA", 100},
	{"_KEEPER", 100},
	{"_CRYSTAL", 100}, // Crystal League weapons sit in the same band as artifacts.
}

// ItemPowerOf returns the IP contribution of a single equipped item.
//
// uniqueName: the raw items.bin uniquename — e.g. "T6_2H_DUALCROSSBOW_AVALON@2".
// quality:    the 1..5 quality byte from Albion's protocol. Caller can
//             pass 1 (Normal) if quality isn't available — that
//             under-reports geared players by ~5-10% but the chip still
//             beats a 3-letter abbreviation.
//
// Returns 0 when the name doesn't parse — callers treat that as an empty
// slot and skip it in the average.
func ItemPowerOf(uniqueName string, quality int) int {
	if uniqueName == "" {
		return 0
	}
	u := strings.ToUpper(uniqueName)

	// Tier: "T<N>_..." — first two bytes give us the digit.
	if len(u) < 3 || u[0] != 'T' {
		return 0
	}
	tierDigit := int(u[1] - '0')
	base, ok := IPBaseByTier[tierDigit]
	if !ok {
		return 0
	}

	// Enchantment: "@<level>" suffix. Strip before further analysis.
	enchant := 0
	if at := strings.LastIndexByte(u, '@'); at >= 0 && at+1 < len(u) {
		lvl := int(u[at+1] - '0')
		if lvl >= 1 && lvl <= 4 {
			enchant = lvl * IPEnchantBonus
		}
		u = u[:at]
	}

	// Type bonus: first-match-wins.
	typeBonus := 0
	for _, t := range IPTypeBonus {
		if strings.Contains(u, t.Substr) {
			typeBonus = t.Bonus
			break
		}
	}

	// Quality bonus: 1..5 → addend.
	qBonus := IPQualityBonus[quality]

	return base + enchant + typeBonus + qBonus
}

// AverageItemPower computes the "average IP" Albion shows in the
// equipment screen. It averages over the six "core" slots (MainHand,
// OffHand, Head, Chest, Shoes, Cape) — Bag, Mount, Potion, Food are
// excluded.
//
// 2H weapons (uniquename contains "2H_") count as occupying BOTH the
// MainHand and OffHand slots, so a 2H IP contribution gets divided by 2
// and added twice. This mirrors Albion's own averaging.
//
// equip is the 10-slot array as it arrives over the wire. qualities is
// the parallel 10-slot quality array (1..5); pass nil to default to
// Normal across the board.
func AverageItemPower(c *ItemCatalog, equip [10]int, qualities [10]int) int {
	if c == nil {
		return 0
	}
	const (
		slotMain  = 0
		slotOff   = 1
		slotHead  = 2
		slotChest = 3
		slotShoes = 4
		// slotBag   = 5  — excluded
		slotCape = 6
		// slotMount = 7  — excluded
		// slotPotion = 8 — excluded
		// slotFood   = 9 — excluded
	)
	coreSlots := []int{slotMain, slotOff, slotHead, slotChest, slotShoes, slotCape}

	// IP of one slot; an empty slot (idx 0) or unknown item contributes 0.
	ipOf := func(slot int) int {
		idx := equip[slot]
		if idx == 0 {
			return 0
		}
		q := 1
		if qualities[slot] >= 1 {
			q = qualities[slot]
		}
		return ItemPowerOf(c.Name(idx), q)
	}

	// Divisor is the FIXED count of core slots (6) — empty slots still
	// count (contributing 0), so unequipping your weapon drops the average
	// the same way it does on the in-game character sheet.
	total := 0
	count := len(coreSlots)
	for _, slot := range coreSlots {
		total += ipOf(slot)
	}
	// A 2H weapon occupies BOTH hand slots, so its IP is counted twice in
	// the numerator (MainHand + the otherwise-empty OffHand it fills) while
	// the divisor stays 6. This matches the game's character-sheet average:
	// e.g. (2×weapon + 4×armor) / 6. NOTE: this intentionally diverges from
	// SAT, which increments the divisor to 7 and so under-reports armed 2H
	// builds relative to the game.
	if equip[slotMain] > 0 && strings.Contains(strings.ToUpper(c.Name(equip[slotMain])), "2H_") {
		total += ipOf(slotMain)
	}
	if count == 0 {
		return 0
	}
	return total / count
}
