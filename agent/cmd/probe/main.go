// probe is a tiny diagnostic that loads items.bin + spells.bin from the
// configured Albion install and prints the entry counts. Use it to verify
// the catalog wiring without launching the full agent (which needs admin).
package main

import (
	"fmt"
	"os"

	"github.com/Goldpip3/albion-info-tracker/agent/internal/config"
	"github.com/Goldpip3/albion-info-tracker/agent/internal/gamedata"
)

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
	}
}
