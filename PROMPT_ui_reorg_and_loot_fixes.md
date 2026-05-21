# UI Reorganization + Loot Cleanup — make the main page the dashboard, kill the bottom-corner buttons, fix the Loot view's "what does this number mean" problem

Read `CLAUDE.md`, `ARCHITECTURE.md`, `REVIEW_post_overhaul.md`, and `PROMPT_followup_fixes.md` (especially Part B on mob name resolution — that work intersects with the Loot Items tab here).

I walked the live site with real data (5-player party, 144 loot pickups, 32m elapsed, agent paired and live). Three structural problems and a handful of rendering bugs. The structural ones drive the user experience; the rendering ones make the existing screens read as broken even when the underlying data is right.

The order below is the order to ship in — Part 1 changes the navigation shape, Parts 2–4 are fixes the user will see immediately afterward.

---

## Part 1 — Top-level tabs replace the bottom-right corner buttons

Today the secondary screens (`Party`, `Loot`, `Sessions`) live in `web/src/Footer.tsx` as small buttons in the bottom-right inset strip. The user finds this silly (their word). The intent is now: the **main page is the dashboard** — combat meter, stat cards, activity log — and adjacent screens hang off a **tab bar at the top**, alongside or below the brand header. This puts Loot and the others one click away at the natural spot in the navigation, not tucked under the version stamp.

### 1a. Add a `<TabBar>` to `web/src/App.tsx` directly under the existing header (above the stat strip)

Tabs, left-to-right:

1. **Meter** — the current main view (damage / healing / tank panes + stat strip + activity log). Default route `/`.
2. **Loot** — the existing `/loot` page (`LootPage.tsx`), no more modal.
3. **Party** — promote `PartyPanel.tsx` to `PartyPage.tsx`, route `/party`.
4. **Sessions** — promote `SessionsPanel.tsx` to `SessionsPage.tsx`, route `/sessions`.

Each tab is a top-level route. Don't pull in a router library — keep the existing pattern of reading `window.location.pathname` (or extend the `?page=loot` query-param if that's already in use) and conditionally rendering the page component. The page bodies for Loot, Party, Sessions already exist as modal-wrapped components; for each, extract the body into a shared component (`LootBody.tsx`, `PartyBody.tsx`, `SessionsBody.tsx` — `LootBody.tsx` already exists) and have both the route page AND the legacy modal use it. Then delete the modal mounting code from `App.tsx` (the `showLoot` / `showParty` / `showSessions` state and the JSX that uses them).

### 1b. Visual treatment of the tab bar

Keep it spare so it doesn't compete with the meter:

- 36-40px tall row directly under the existing brand strip.
- Each tab a flat text-only button at ~12.5px, semibold, all-caps with `letterSpacing: 0.08em`. No pill, no fill. Hover lightens text from `--sk-fg-2` to `--sk-fg-0`. Active tab gets a 2px bottom border in `--sk-damage` (matches the existing tab pattern in `DrillIn.tsx::DrillTabs`).
- Tab badges where relevant: `Loot · 144`, `Sessions · 1`, `Party · 5`. Use the existing snapshot counts (`loot.length`, `recent.length`, `players.length`). Hide the badge when zero.
- Mobile / narrow viewports — collapse to a single-row scrollable strip; don't try to fit into a hamburger menu for four tabs.

### 1c. Delete the bottom-right Party / Loot / Sessions buttons from `Footer.tsx`

The footer goes back to being chrome only: tick rate, last-update timestamp, version, live/stale indicator. Remove the `onOpenParty`, `onOpenLoot`, `onOpenSessions`, `sessionsCount`, `lootCount` props from `FooterProps`. Remove the matching state and handlers in `App.tsx`. The buttons leave no trace.

### 1d. The mode tabs (`Damage / Healing / Tank / Mechanics`) on the meter view stay where they are

They're scoped to the meter view itself — when the user is on the `Meter` top-tab they pick which combat metric to look at. Leaving them in place keeps the existing muscle memory.

### 1e. Tab order on mobile / page-title sync

When you switch tabs, update `document.title` to `GDA · <Tab name>` (e.g. `GDA · Loot`). When the user opens the site fresh, default to the Meter tab.

---

## Part 2 — Loot view rendering bugs

The user opened the Loot tab with real session data (11 looters, 144 pickups, 32m elapsed). Several rows look broken or unreadable. Each issue below is a concrete UX failure, not a design preference.

### 2a. The "34" and "473K" columns need headers (or inline units)

In `LootBody.tsx`, `LooterRow` (around line 277) renders the pickup count and silver total as bare mono numbers with no column header above them. The user looked at `Ichigoniggasaki · 34 · 473K` and reasonably asked "what is 34?". The fix:

- Add a header row above the rows with column titles: blank | `PLAYER` | `RELATIVE TO TOP` | `PICKUPS` | `SILVER` | blank. Use the same grid template as the data rows (`44px minmax(180px, 1.4fr) minmax(0, 2fr) 70px 100px 16px`), same uppercase-tiny font as the existing `MeterTable.tsx` header (around line 75 of MeterTable). Sticky-top inside the scroll container so the headers stay visible while scrolling long lists.
- OR: change the row format so the unit travels with the number: `34 pickups · 473K silver`. This matches the format already used in the Top Farmer card (`"22 PICKUPS"` sub-label). Either is acceptable; the header-row option keeps the rows scannable, which is what the user actually asked for. **Pick the header-row option.**

### 2b. The Items tab's silver-pile rows print the raw FixPoint quantity

In the Items tab (`LootBody.tsx::ItemsTab`, around line 522-540), each silver-pile row displays `{l.quantity}` directly in the QTY column. For silver, `l.quantity` is the raw silver units (e.g. `100000000` for 100K silver) — that's a wall of zeros that's meaningless to the user. The SILVER column already renders `fmt(toSilver(l.quantity))` correctly (showing "100K"). Fix the QTY rendering:

- For silver-pile rows (`l.isSilver === true`), render the QTY cell as `"—"` or `"1"` (one pile event). Don't repeat the silver amount.
- For item rows, keep `{l.quantity}` as-is (`1`, `2`, `26`, etc — the stack size).
- Confirm the same fix in the per-player drill view, which uses the same pattern (around line 449).

### 2c. The Items tab's loot-source field shows raw `@MOB_T6_HARVESTER_PLAYERSPAWN` localization tags

Look at the drill view's sub-line for each entry (`LootBody.tsx`, around line 525-527): `from ${l.lootedFrom}`. The `lootedFrom` field today carries Albion's raw `@MOB_<id>_<thing>` localization key. This is the mob-name-resolution work already specified in `PROMPT_followup_fixes.md` Part B (load `mobs.bin`, wire the `NewMob` handler, resolve ObjectId → localized name).

**Don't duplicate that work here.** Instead:

- Until Part B lands, sanitize `lootedFrom` in the agent before it goes on the wire: strip the leading `@MOB_`, drop the trailing `_PLAYERSPAWN` / `_PLAYERMOBSPAWN` / `_HARVESTER_PLAYERSPAWN` suffixes, replace underscores with spaces, title-case the rest. Output for `@MOB_T6_HARVESTER_PLAYERSPAWN` becomes `T6 Harvester`. Same for chests (`@CHEST_<id>` etc.). This is a 20-line function in `agent/internal/domain/loot.go` or a new `agent/internal/domain/loot_source.go`.
- When Part B lands and `mobs.bin` is loaded, replace the prettifier with a real localized name lookup. Leave a `// TODO: switch to MobCatalog.Name once mobs.bin loader lands` marker.

### 2d. Rows with no top-priced item show "—" with no context

Several rows in the per-player view (e.g. Goldpipe with 9 pickups but no top item, Tz03 with 1 pickup, mathsil with 1, yonder5 with 1, DoctorPorro with 1) print `—` in the top-item slot. The cause: their entries are silver piles or items AODP hasn't priced yet. The fix:

- When a looter has pickups but no priced items, display the most-recent item name (regardless of price) with `(unpriced)` after it. The looter at least sees what they picked up — `recent: Master's Boots (unpriced)` reads better than a naked `—`.
- When the looter has only silver pickups, display `silver only · X K` where X is the SilverPicked formatted. This is honest — they didn't loot items, just coin.
- Truly empty rows (no entries at all) shouldn't reach this view; if they do, hide them entirely.

Implement by extending `LooterTotals` in `agent/internal/domain/loot.go` with two more optional fields:

- `RecentItemName string` — the name of the most-recent non-silver pickup, regardless of price.
- `OnlySilver bool` — true when every entry for this looter was a silver pile.

Populate in the same rollup loop (`LooterTotalsList`, around line 168). Mirror on `web/src/types.ts::LooterTotals`. Use in the sub-line rendering in `LootBody.tsx::LooterRow` (around line 327-331) and the Top Farmer card (around line 253-260).

### 2e. Empty IPChip slots render as a bordered ghost

Half the looter rows have no `player` entry in the snapshot (`playerByName.get(l.name)` returns undefined). The IPChip slot then renders `<Placeholder size={30} />` — an empty grey rectangle that looks like a broken chip. Better:

- Have `<Placeholder>` render a small muted `—` glyph centered inside the slot instead of a blank rectangle. Use `var(--sk-fg-3)` text color and a transparent background. The viewer sees "no IP data" instead of "broken element".
- Add a `title` attribute (`title="IP unknown — agent hasn't seen this looter's equipment yet"`) for hover context.

### 2f. The looter-filter is actually working; the *result* still looks wrong because guildies look like strangers

The agent's `allowedLooters` (in `loot.go`, around line 123) correctly limits visible looters to: local player + party members + same-guild players + the explicit `alwaysIncludeNames` allowlist. That's the filter the user asked for. Why the user still feels "random people" are showing up: of the 11 looters in the live screenshot, 5 were in their active party (per the damage meter) and the other 6 were same-guild members who weren't in the active party.

Two cleanups make this honest:

- Add a small badge next to each looter's name in `LootBody.tsx::LooterRow` showing why they're included: `PARTY` (orange-tinted), `GUILD` (purple-tinted), `FRIEND` (blue-tinted, from the allowlist). Mirrors how the meter colors local-player rows. Source the badge from a new `Source string` field on `LooterTotals` populated by `allowedLooters` (extend the function to return the reason too, not just the name set).
- Add a Settings toggle: `Loot view → Include guildies (default: on)`. When off, only party members and explicit-allowlist names appear. Surface the toggle in `web/src/SettingsPanel.tsx` next to the existing `Pin local user`. Pipe the setting through to the agent via a new `command` envelope (`{action: "setLootFilter", scope: "party" | "partyGuild"}`) — same pattern as the existing `resetSession` command. Defaults: `partyGuild`.

---

## Part 3 — Party panel stability + spell-label bugs

### 3a. Party rows reshuffle each tick

`PartyPanel.tsx` (line 74) renders `{players.map((p) => <PartyRow key={p.userGuid} p={p} />)}`. The `players` array comes from `snapshot.players`, which the agent builds from a Go `map[Guid]*Entity` — Go's map iteration is randomized, so every snapshot tick the order changes and the rows visibly swap positions. The user described it as "keeps switching back and forth up and down, switch positions."

Fix: sort the array before rendering. Stable by `itemPower` desc, then by `userGuid` asc as the tie-break (same pattern as `MeterTable.tsx::sort` around line 38). Memoize on `snapshot.generatedAt` so the sort doesn't re-run every render. The Damage meter table already has this stabilization; copy the pattern verbatim into the Party page.

### 3b. Bound-slot abilities are showing passive effects

Looking at the live PARTY panel screenshot, AnimeNutz (Spear weapon, Dreadstorm Monarch) shows `Q Passive Spellpower Chance Spear Effect` in their Q binding. wIlDDYY (Redemption Staff healer) shows passives from unrelated weapons (`Sword Condition`, `Knuckle Rushdown`, `Spellpower Caster Crossbow`). Zilthic (Incubus Mace tank) has the same — `Spear`, `Quarterstaff`, `Crossbow` passives leaking into bound slots.

These are real bugs in the spell-slot indexing. The agent reads `param 7` from `CharacterEquipmentChanged` into a 14-int array per the SAT-documented layout `(0..2 MainHand, 3 Armor, 4 Head, 5 Shoes, 12 Potion, 13 Food)`. Indices 6–11 are equipment-passive slots that today get mislabeled.

To fix:

- In `agent/internal/domain/engine.go`, find the `activeSpellSlots` builder (search for `ActiveSpellSlots` or `equipmentSlots`) and confirm the index→slot-label mapping. The current code is treating every entry the same way — it should:
  - Index 0 → `Q` (active ability 1)
  - Index 1 → `W` (active ability 2)
  - Index 2 → `E` (active ability 3)
  - Index 3 → `R` (active ability 4 — for some weapons; missing for 1H)
  - Indices 4–11 → equipment passives; do NOT label these `Q/W/E/R`. Either drop them from the active-spells row entirely (they belong on the equipment row), or label them by their actual slot (`Armor passive`, `Head passive`, `Cape passive`, etc).
  - Index 12 → `Potion`
  - Index 13 → `Food`
- Cross-check with `src/StatisticsAnalysisTool/Network/Events/CharacterEquipmentChangedEvent.cs` and the corresponding `MainSpells` handler in SAT's C# source — they're the reference for the exact mapping.
- The "Passive ..." names in the screenshot are coming from passive equipment effects bound to slots 4-11; once those stop being labeled as `Q`, they'll render in the equipment row where they belong.

### 3c. Spell names like "Plateamor" (misspelled) need extending the override table

Some passive effect uniquenames aren't in `localization.bin` and fall through `prettifySpellName`. `PLATEARMOR_HEALTHREDUCTION_EFFECT` becomes `Plateamor Healthreduction Effect` because the prettifier doesn't know `PLATEARMOR` is one word. Same kind of bug for `BRIDGEWATCH`, `THETFORD`, `HEALTHCHANCE`, `ARMORCHANCE`, `SPELLPOWER`, etc. — multi-word concatenations that need to be broken back up.

Extend `agent/internal/gamedata/spells_override.go`:

- Add a "compound word splitter" map that turns `PLATEARMOR` → `Plate Armor`, `HEALTHCHANCE` → `Health Chance`, `ARMORCHANCE` → `Armor Chance`, `SPELLPOWER` → `Spell Power`, `ATTACKBUFF` → `Attack Buff`. Run this BEFORE the title-case step.
- Add a small geography map for city names: `BRIDGEWATCH`, `MARTLOCK`, `THETFORD`, `FORTSTERLING`, `LYMHURST`, `CAERLEON` → already-titled forms. So `PASSIVE_CAPE_BRIDGEWATCH_DEBUFF` → `Bridgewatch Cape Debuff`, not `Cape Bridgewatch Debuff` (note the word order swap — city goes first when it qualifies a noun, but if reordering is hard, just title-casing each token correctly is acceptable; the user's complaint is the misspelling, not the order).
- These additions need to NOT break existing override entries (`Caltrops`, `Flickershot`, etc.). Test with the existing probe.

This work is genuinely tedious because there are a lot of compound words in Albion's uniquenames. Bias toward the 20 most common; leave a `// TODO: extend as more passives show up in the wild` comment.

---

## Part 4 — Verification

Before declaring this done:

- [ ] Top of page has a `MUTER | LOOT · N | PARTY · 5 | SESSIONS · K` tab row; tabs route correctly and update `document.title`
- [ ] Bottom-right `Party / Loot / Sessions` buttons are gone from the Footer
- [ ] Loot per-player view has visible column headers (`PLAYER · RELATIVE TO TOP · PICKUPS · SILVER`); a viewer can tell what `34` and `473K` mean within 1 second of looking
- [ ] Loot Items tab: silver piles show `—` in QTY (not `100000000`); item rows still show stack size
- [ ] Loot Items tab: source field shows `T6 Harvester` (prettified) instead of `@MOB_T6_HARVESTER_PLAYERSPAWN`. Once Part B from `PROMPT_followup_fixes.md` lands, it'll show the localized name without changes here.
- [ ] Loot per-player rows that have pickups but no priced items show `recent: <item name> (unpriced)` instead of `—`
- [ ] Loot per-player rows that picked up only silver show `silver only · X K`
- [ ] Empty IP chips render `—` centered with a hover tooltip, not a blank rectangle
- [ ] Each looter row carries a `PARTY` / `GUILD` / `FRIEND` badge; Settings has an "Include guildies" toggle that defaults on
- [ ] Party page rows don't reshuffle each tick — sort is stable by IP desc, then guid asc
- [ ] Party page spell-slot labels: Q/W/E (and R when applicable) only show active abilities, not passive equipment effects
- [ ] Party page passive names like `Plateamor` are gone; reads as `Plate Armor Health Reduction Effect` or better

Ship when all twelve check. Don't rewrite the loot capture or AODP pricing — those work. The agent-side work in this entire prompt totals < 200 lines; the rest is web restructuring and component cleanup. Estimated 1–1.5 days of focused work.
