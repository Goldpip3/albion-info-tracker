package gamedata

import "testing"

// stubCatalog builds an ItemCatalog from index→uniquename pairs so the IP
// math can be exercised without loading items.bin.
func stubCatalog(names map[int]string) *ItemCatalog {
	by := make(map[int]ItemEntry, len(names))
	for idx, name := range names {
		by[idx] = ItemEntry{UniqueName: name}
	}
	return &ItemCatalog{byIndex: by}
}

func TestItemPowerOf(t *testing.T) {
	cases := []struct {
		name    string
		unique  string
		quality int
		want    int
	}{
		{"empty", "", 1, 0},
		{"t4 base", "T4_2H_BOW", 1, 700},
		{"t6 head", "T6_HEAD_PLATE_SET1", 1, 900},
		{"t8 base", "T8_2H_TESTWEAPON", 1, 1100},
		{"t8 enchant2", "T8_MAIN_SWORD@2", 1, 1300},          // 1100 + 200 enchant
		{"crystal artifact + enchant", "T8_2H_DUALCROSSBOW_CRYSTAL@2", 1, 1400}, // 1100 + 200 + 100
	}
	for _, c := range cases {
		if got := ItemPowerOf(c.unique, c.quality); got != c.want {
			t.Errorf("%s: ItemPowerOf(%q,%d) = %d, want %d", c.name, c.unique, c.quality, got, c.want)
		}
	}
}

func TestAverageItemPower(t *testing.T) {
	// Slot layout: 0 Main, 1 Off, 2 Head, 3 Chest, 4 Shoes, 5 Bag, 6 Cape.
	const (
		twoH    = 1
		head    = 2
		chest   = 3
		shoes   = 4
		cape    = 5
		mainOne = 6
		offOne  = 7
	)
	cat := stubCatalog(map[int]string{
		twoH:    "T8_2H_TESTWEAPON",   // 1100, two-handed
		head:    "T8_HEAD_PLATE_SET1", // 1100
		chest:   "T8_ARMOR_PLATE_SET1",
		shoes:   "T8_SHOES_PLATE_SET1",
		cape:    "T8_CAPE",
		mainOne: "T8_MAIN_SWORD", // 1100, one-handed
		offOne:  "T8_OFF_SHIELD", // 1100
	})

	mk := func(main, off, h, c, s, cp int) [10]int {
		return [10]int{main, off, h, c, s, 0, cp, 0, 0, 0}
	}
	var noQ [10]int

	cases := []struct {
		name  string
		equip [10]int
		want  int
	}{
		// Full T8.0 with a 2H weapon: (2*1100 + 4*1100) / 6 = 1100.
		{"full T8 2H", mk(twoH, 0, head, chest, shoes, cape), 1100},
		// 2H unequipped: only the 4 armor pieces, still divided by 6.
		{"2H unequipped", mk(0, 0, head, chest, shoes, cape), 4 * 1100 / 6},
		// 1H + offhand, no 2H: 6 filled slots / 6 = 1100 (divisor unchanged).
		{"1H + offhand", mk(mainOne, offOne, head, chest, shoes, cape), 1100},
	}
	for _, c := range cases {
		if got := AverageItemPower(cat, c.equip, noQ); got != c.want {
			t.Errorf("%s: AverageItemPower = %d, want %d", c.name, got, c.want)
		}
	}
}
