// probe is a tiny diagnostic that loads items.bin + spells.bin from the
// configured Albion install and prints the entry counts. Use it to verify
// the catalog wiring without launching the full agent (which needs admin).
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/Goldpip3/albion-info-tracker/agent/internal/config"
	"github.com/Goldpip3/albion-info-tracker/agent/internal/gamedata"
)

func contains(s, sub string) bool {
	return strings.Contains(strings.ToUpper(s), strings.ToUpper(sub))
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Println("config:", err)
		os.Exit(1)
	}
	fmt.Println("install root:", cfg.AlbionInstallRoot)
	fmt.Println("bin dir:     ", gamedata.BinDir(cfg.AlbionInstallRoot, gamedata.ServerLive))

	items, err := gamedata.LoadItemCatalog(cfg.AlbionInstallRoot, gamedata.ServerLive)
	if err != nil {
		fmt.Println("items.bin:", err)
	} else {
		fmt.Printf("items.bin OK — %d entries\n", items.Len())
		// Sample range + the index Goldpipe's MainHand resolved to in
		// the verbose log, to verify Tier 8 weapons resolve.
		for _, idx := range []int{1, 100, 500, 1000, 2000, 4000, 6620, 6621, 6622, 6623, 6624, 6625} {
			fmt.Printf("  [%d] %q\n", idx, items.Name(idx))
		}
	}

	spells, err := gamedata.LoadSpellCatalog(cfg.AlbionInstallRoot, gamedata.ServerLive)
	if err != nil {
		fmt.Println("spells.bin:", err)
	} else {
		fmt.Printf("spells.bin OK — %d entries\n", spells.Len())
		// Search for crossbow/explosive/bomb abilities — anything the
		// user might recognise.
		patterns := []string{"EXPLOSIVE", "CROSSBOW", "BOLT", "DUALCROSSBOW", "ARCLIGHT", "FLICKER"}
		for _, pat := range patterns {
			fmt.Printf("  --- containing %q ---\n", pat)
			n := 0
			for i := 0; i < spells.Len()*2 && n < 8; i++ {
				name := spells.Name(i)
				if name != "" && contains(name, pat) {
					fmt.Printf("  [%d] %s\n", i, name)
					n++
				}
			}
		}
	}

	loc, err := gamedata.LoadLocalization(cfg.AlbionInstallRoot, gamedata.ServerLive)
	if err != nil {
		fmt.Println("localization.bin:", err)
	} else {
		fmt.Printf("localization.bin OK — %d EN-US strings\n", loc.Len())
		// Reverse-lookup: find @SPELLS_ tu-ids whose English value
		// contains "explosive bolt" or similar so we can wire the
		// real spell uniquename to the right user-visible name.
		fmt.Println("  --- localization reverse-lookup ---")
		searchTerms := []string{"explosive bolt", "frost bolt", "flickershot", "chain slash", "auto attack"}
		hits := loc.Search(searchTerms, 12)
		for term, results := range hits {
			fmt.Printf("  %q matches:\n", term)
			for _, r := range results {
				fmt.Printf("    %-50s → %q\n", r.Tuid, r.Value)
			}
		}
	}
}
