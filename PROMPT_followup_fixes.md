# Follow-up fixes — IP should show Average IP, mob target names, regressions

Read `CLAUDE.md`, `ARCHITECTURE.md`, and `REVIEW_post_overhaul.md` before starting. The previous overhaul (commit `dffd7ab`) shipped a working IPChip, loot logger, party panel, sessions archive, and dungeon scope — most of the original Part 1–3 work plus chunks of Part 4. This follow-up cleans up the remaining gaps the user flagged after using the live build.

Work order matters — Part A is the highest-impact, lowest-cost fix. Parts B and C are net-new features. Part D is housekeeping.

---

## Part A — IP chip must show Average Item Power, not Base Average

The chip currently shows a number that's close to Albion's "Base Average Item Power" (no quality contribution) rather than its "Average Item Power" (quality + spec). On the user's character Albion reports:

- Average Item Power: 1153
- Base Average Item Power: 1004

The 149-IP gap is overwhelmingly the quality bonus across the six core slots (and a small bit of mastery, which Albion doesn't broadcast and we cannot recover — accept that). The agent code in `gamedata/itempower.go::AverageItemPower` already includes quality in the math — the variable is `qualities[slot]`, it's plumbed through `applyEquipmentArray` from `parseEquipmentParams`, the formula adds `IPQualityBonus[quality]`. So in principle this is solved. In practice the chip is showing ~1004, which means the quality array is arriving as all zeros and the quality bonus is contributing nothing.

Three things to do:

**A1. Instrument the equipment parser and confirm what's actually arriving.** In `agent/internal/domain/engine.go::parseEquipmentParams`, when `ALBION_AGENT_VERBOSE=1` is set, log the raw equipment params *as received*: every key in the `map[byte]any`, the type of the value, and the parsed `equip` / `qualities` arrays. Add a `if verbose { log.Printf(...) }` block at the bottom of the function before returning. Then run the agent in admin PowerShell with verbose mode, re-zone in Albion once, and grep the output for `parseEquipmentParams`. Two outcomes are possible:

- The user's Albion build is shipping quality under a different param byte than 41 (NewCharacter) or 3 (CharacterEquipmentChanged). The reference C# SAT source in `src/StatisticsAnalysisTool/Network/PacketProvider.cs` and the event handlers under `src/StatisticsAnalysisTool/Network/Handler/` is the source of truth — search there for the quality param index, port it over.
- The quality array is shipping but as `[]byte` or `[]int8`, and `intsOfArray` isn't unpacking it. Inspect `intsOfArray` in `agent/internal/domain/engine.go` — if it doesn't cover the type Albion sends, add the case.

**A2. Once quality is being decoded correctly**, the chip's number should jump to within 5–15 IP of Albion's "Average Item Power" display. If it overshoots, the type-bonus table in `itempower.go::IPTypeBonus` is double-counting (Crystal League weapons are listed at +100 there, which may already be baked into the tier base depending on patch — measure before adjusting). If it undershoots, the missing piece is mastery / spec, which is not recoverable from packet capture. Accept undershooting by up to ~50 IP — that's the spec contribution.

**A3. Label the chip's tooltip header accurately.** Today it says `"Item Power · avg"`. Once A1 is fixed, leave the label. If A1 reveals quality is fundamentally unrecoverable on the current patch, change the label to `"Base Item Power · avg"` so the user knows the number reads low intentionally rather than a bug. Don't ship the misleading state.

Verification: hover the chip on the local player's row in `MeterTable.tsx`. The tooltip shows per-slot IP breakdown. With a weapon at T8.2 quality 3 the MainHand slot should read around 1300–1400. If it reads 1100–1200, quality isn't reaching the calc.

---

## Part B — Resolve enemy / mob target names (`#8087` → `Fox`)

In the drill-in "Targets" tab, mobs currently render as `#<objectId>` because the agent never learned the mob's name. The C# SAT codebase handles this end-to-end. Port the pattern.

**B1. Load `mobs.bin`.** The file path is already declared in `agent/internal/gamedata/paths.go:27`, but the loader doesn't exist. Add `agent/internal/gamedata/mobs.go` modeled on `items.go`:

- `MobEntry { Index int; UniqueName string; NameLocaTag string; Tier int }`
- `MobCatalog { byIndex map[int]MobEntry }`
- `func (c *MobCatalog) Lookup(index int) MobEntry`
- `func (c *MobCatalog) Name(index int) string` — returns the localized name via `localization.bin` when wired (the loca tag is typically `@MOB_<uniquename>`), otherwise the prettified uniquename.
- `func LoadMobCatalog(installRoot string, server ServerType) (*MobCatalog, error)` — DES+gzip+XML pipeline identical to `items.go::LoadItemCatalog`. Mirror SAT's `StatisticAnalysisTool.Extractor.MobData.cs` extraction algorithm for the index/uniquename pairing.

Wire it through `cmd/agent/main.go` next to `LoadItemCatalog` / `LoadSpellCatalog`, and add a `SetMobCatalog` method on `Engine` parallel to `SetItemCatalog`. Update the probe at `agent/cmd/probe/main.go` to print mob catalog size + a few sample lookups (e.g., `Fox`, `Bandit`, `Heretic`).

**B2. Handle the `NewMob` event.** `agent/internal/gamecodes/events.go:131` already maps the event code; the dispatcher in `engine.go::onEvent` (around line 252) just needs a `case gamecodes.EventNewMob: e.handleNewMob(ev.Parameters)` branch.

`handleNewMob` parses (consult the SAT handler at `src/StatisticsAnalysisTool/Network/Handler/NewMobEventHandler.cs` and `src/StatisticsAnalysisTool/Network/Events/NewMobEvent.cs` for the exact param indices — typical SAT shape is param 0 = ObjectId, param 1 = MobIndex, param 5 = current HP, param 6 = MaxHP). Store the mob in a new `MobTracker` registry on the engine, keyed by ObjectId, with `{ MobIndex, Name, MaxHP, Cluster }`.

**B3. Resolve target names in `snapshot.go::topTargets`.** Today the function looks up the affected ObjectId in `Store` (the player registry). After the player lookup misses, fall through to `MobTracker.ByObjectId(id)` and use its `Name`. If still empty, fall back to `#<id>` as today. Result: a fox in your damage breakdown reads `Fox` instead of `#8087`.

**B4. Mob name cache cleanup.** Mobs are zone-scoped; their ObjectId becomes stale when you re-zone. Wire the existing `JoinFinished` handler (`engine.go`, search for `EventJoinFinished`) to call `e.mobs.ResetForNewZone()` so the next zone's mob ObjectIds don't collide.

Verification: enter a zone with a known fox spawn (Forest biome anywhere), tag a fox once, open the drill-in Targets tab — the entry that was `#8087` should now read `Fox` (or whatever Albion's localized mob name is). Re-zone, tag a wolf, confirm the wolf gets its own name and the fox's stale ObjectId doesn't leak.

---

## Part C — UI cleanups in the live build

**C1. Remove the redundant `Item Power` column toggle in Settings.** The DISPLAY → Columns section of `SettingsPanel.tsx` exposes `Item Power` as an optional column alongside `DPS`, `HPS`, `Damage Taken`, `Healing`, `Crit %`. Since IP already lives in the IPChip on every row, the column is duplicative. Either delete the toggle entirely or rename it to "Show IP after name" (the latter only if you want a second numeric IP surface — I don't think you do). Recommendation: delete the toggle.

**C2. Hide the `Crit %` column toggle.** Albion does not expose a crit flag in HealthUpdate; the column will be all `—` forever. Either remove the toggle from `SettingsPanel.tsx` or grey it out and mark it `coming soon` the way `Mechanics` mode is marked in the header.

**C3. The footer's `v0.6.0 · stale` indicator.** "Stale" should automatically resolve to "live" once the agent's first snapshot arrives. Check `web/src/Footer.tsx` — the stale threshold is probably > 5 seconds since `lastMessageAt`. Confirm it flips correctly and doesn't get stuck. If `Footer.tsx` is rendering `stale` when the WebSocket is currently open and receiving, the state isn't being updated; trace from `useMeterSocket.ts`.

**C4. Window title and brand.** The site title currently reads `Combat Analytics`. The user's brand is `Goldpipe's Data Analytics` (visible in the top-left as `gda · Combat Meter`). Either align the document title to `Goldpipe's Data Analytics` or simplify the in-page header to `GDA · Combat Meter` so the two surfaces match. The user has expressed pride in the brand — keep it consistent.

---

## Part D — Above-board / safety documentation

Add a short `SAFETY.md` at the repo root capturing:

1. This tool performs **passive UDP packet capture** of the local Albion process. It does **not** modify the game client, inject DLLs, read game memory, automate gameplay, or render in-game overlays.
2. All data renders in a browser window, separate from the Albion process — matching the "website data" pattern explicitly permitted in Sandbox Interactive's [Feb 2020 third-party-software statement](https://forum.albiononline.com/index.php/Thread/124819-Regarding-3rd-Party-Software-and-Network-Traffic-aka-do-not-cheat-Update-16-45-U/).
3. No real-time tactical advantage is provided. The meter does not surface enemy positions, hidden-player intel, resource node locations, or anything that isn't already visible on the user's screen.
4. Multi-agent / guild views are post-fight analytics; the worker does not stream live enemy presence as scouting intel.
5. Users who want explicit confirmation may email `support@albiononline.com` describing the tool — Sandbox's policy invites this and they respond. Link the public site (`https://albion-meter-web.pages.dev/`) and the GitHub repo when asking.

Don't make `SAFETY.md` legalistic — keep it factual. The point is to give a player who's asked "is this allowed" a one-page answer that mirrors Sandbox's own published guidance.

---

## Verification checklist before declaring this done

- [ ] Agent build clean: `cd agent && go build ./...`
- [ ] Web build clean: `cd web && npm run build`
- [ ] Probe shows mob catalog size > 0 and resolves `Fox` / `Bandit` / similar
- [ ] Live agent run with verbose: `qualities[]` array logs as non-zero values
- [ ] IPChip reading on the user's local row is within 50 IP of Albion's "Average Item Power" character-sheet number
- [ ] Drill-in Targets tab shows mob names instead of `#<id>` for at least one common open-world mob
- [ ] Settings panel no longer shows the redundant `Item Power` column toggle
- [ ] Settings panel hides or labels `Crit %` correctly
- [ ] Footer flips from `stale` to `live` within 1 second of an agent push
- [ ] `SAFETY.md` exists at the repo root

When all eight check, ship.
