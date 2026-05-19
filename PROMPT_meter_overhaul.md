# Meter Overhaul — Naming, IP Display, Resource Trim, Feature Backlog

You are working on the `go-port` branch of the Albion DPS repo. Read `CLAUDE.md` and `ARCHITECTURE.md` first — they describe the data flow (Photon UDP → Go agent → Cloudflare Worker → React/Vite web UI) and the file layout. This prompt rolls four bodies of work into one: **naming fixes**, **IP display**, **resource utilization trims**, and a **prioritized feature backlog**. Treat parts 1–3 as the immediate sprint and part 4 as the roadmap you propose at the end.

The user observed three concrete naming defects on https://albion-meter-web.pages.dev/ while playing Adept's Arclight Blasters (Avalonian dual crossbow):

- The ability "1" / left-click slot reads as `1` instead of `Auto Attack`.
- The drill-in / role-label sometimes prints `CRYSTAL DUAL CROSSBOWS` even though the actual weapon is an Arclight Blaster.
- Caltrops prints as something resembling `Curse skeleton barf fdhr` — i.e. `prettySpell()` ran an unlocalized uniquename through underscore-titlecase and produced garbage.
- The player row prints `XBW` next to the name. The user wants the player's **IP (Item Power)** there instead.

Work is gated on each section: do not skip ahead. After parts 1–3, run `go build ./...`, `npm run build` in `web/`, and propose the part-4 backlog as a written plan before touching it.

---

## Part 1 — Naming fixes (agent + web)

All three name bugs share one root cause: the agent emits a uniquename string and the web UI prettifies it via `prettySpell()` (`web/src/format.ts`). When localization.bin has no entry, the prettifier hides the bug behind a Title-Case mangling. Fix this in the **agent**, not the web side — keep the wire format as resolved English where possible, and have the web only fall back to `prettySpell()` when the agent admits it has no name.

### 1a. Auto-attack normalization

Albion's auto-attack ability uniquenames follow patterns like `CROSSBOW_AUTO_ATTACK_JUMP`, `DAGGER_AUTO_ATTACK`, `2H_HAMMER_AUTO_ATTACK_1`, etc. Their localization entries (when they exist) often say `"1"`, `"Light Attack"`, or are missing entirely — none of which are useful. Treat all of them as "Auto Attack" before the snapshot is built.

- Edit `agent/internal/domain/engine.go::localizedSpellName(idx int)` so that after resolving the uniquename it checks `strings.Contains(strings.ToUpper(uniqueName), "AUTO_ATTACK")` (and `AUTOATTACK` as a defensive variant) and returns the string `"Auto Attack"` early.
- Don't put this in `web/src/format.ts` — every consumer of `localizedSpellName` (Fight tab, Session tab, Assists tab, ActivityLog) needs the same answer, and centralizing in Go keeps the snapshot self-describing.
- Verify: in admin PowerShell run `.\agent.exe` with `ALBION_AGENT_VERBOSE=1`, fire a few left-clicks in a training dummy zone, watch the Fight tab in the meter — the entry that used to read `1` or `Crossbow auto attack jump` should now read `Auto Attack`.

### 1b. Spell-name localization fallback chain

`Localization.SpellName` currently tries `@SPELLS_<U>` then `@MOB_ABILITIES_<U>`. Several real abilities live under different prefixes — e.g. some passive sub-spells under `@SPELLS_<U>_DESC`, a handful of artifact-only abilities under `@ITEMS_<U>_SPELLDESC`, and crossbow Caltrops in particular is keyed off the parent item's spell, not the bare uniquename.

Do the following in `agent/internal/gamedata/localization.go`:

1. Add a small ordered list of prefixes to try in `SpellName`. Today: `@SPELLS_`, `@MOB_ABILITIES_`. Add: `@SPELL_`, `@SPELLDESC_`, and `@ITEMS_<uniquename>_SPELL` (used by some artifact crossbow/dagger abilities). For each prefix variant, also try stripping a trailing `_<digit>` (Albion sometimes versions abilities with `_1`, `_2`, `_3` suffixes that don't appear in localization).
2. If every prefix misses, return `""` — do **not** return the prettified uniquename here. Let the call site decide.
3. In `agent/internal/domain/engine.go::localizedSpellName`, when `SpellName` returns `""`, run a small in-Go prettifier that:
   - Strips a leading weapon family prefix (`CROSSBOW_`, `DAGGER_`, `2H_HAMMER_`, …) using a sorted list of known prefixes (copy the keys from `gamedata/role.go::ClassifyWeapon`'s match tokens).
   - Strips trailing slot/grade tokens (`_E`, `_Q`, `_W`, `_1`, `_2`, `_3`, `_PASSIVE`).
   - Title-cases each remaining `_`-separated token.
   - Returns the result. The output of `BOLTCASTER_CALTROPS_E` should be `Caltrops`, of `CROSSBOW_FLICKERSHOT_E` should be `Flickershot`.
4. Keep a hard-coded override table in `gamedata/spells_override.go` (new file) for known stubborn cases. Seed it with at least: Caltrops, Flickershot, Frost Shot, Energy Bolt — abilities the user already named as broken. Each override key is a normalized uniquename (post-prefix-strip, upper-cased), the value is the display name.

### 1c. Weapon label (`roleLabel`) — fix the Arclight Blasters / Crystal Dual Crossbows confusion

Two real bugs:

- `agent/internal/gamedata/role.go` line ~47 prints `"ARLIGHT BLASTER"` (missing "C", missing trailing "S"). Fix to `"ARCLIGHT BLASTERS"`.
- The classification fallback in `ClassifyWeapon` matches `DUALCROSSBOW_CRYSTAL` before `CROSSBOW_AVALON`/`AVALON_CROSSBOW`. The Avalonian dual crossbow's uniquename is `T<N>_2H_DUALCROSSBOW_AVALON` — it contains both `DUALCROSSBOW` and `AVALON`. Because Crystal is matched first, an Arclight Blasters item ends up labelled `CRYSTAL DUAL CROSSBOWS`. Add a dedicated `DUALCROSSBOW_AVALON` case at the same level as `DUALCROSSBOW_CRYSTAL`, returning `"ARCLIGHT BLASTERS"`. Then verify the entire artifact block is ordered specific → generic, with no `<family>_<variant>` substring collision before its sibling. Audit Dagger Pair, Claw Pair, Quarterstaff, and Sword variants the same way — those have the same `<family><suffix>` overlap risk.

The label only matters when `localization.ItemName(uniqueName)` returns `""`. When localization works (it usually does), the live page already shows `ADEPT'S ARCLIGHT BLASTERS` correctly — confirmed in browser. So the label fix is the safety net. Don't remove it; tighten it.

Do **not** change the wire field name `roleLabel` or `classCode` — that ripples into history, archives, and demo data.

### 1d. Web side cleanup

`web/src/format.ts::prettySpell` should keep working but become a last-resort: only used when `s.name` is empty. The agent's new behavior guarantees `s.name` is either the localized English (preferred), an override (Caltrops/Flickershot/etc.), or a clean Title-Case ("Auto Attack"). If the web still receives a raw `CROSSBOW_<...>` uniquename from an old agent version, `prettySpell` keeps it from looking like garbage.

---

## Part 2 — Replace the XBW chip with IP (Item Power)

The user wants the spot occupied by the 3-letter weapon abbreviation (`XBW`, `DGR`, `HLY`, …) replaced with the player's **average Item Power**, which in Albion is computed across equipped slots.

### 2a. Read full equipment, not just MainHand

`agent/internal/domain/engine.go::applyEquipment` currently calls `firstIntOfArray(equip)` and only uses `mainHand`. The equipment array is 10 slots (MainHand, OffHand, Head, Chest, Shoes, Bag, Cape, Mount, Potion, Food — order per SAT). Capture **all ten** indices into `Entity.Equipment [10]int`.

- Extend `Entity` with `Equipment [10]int` and `ItemPower int` (the computed value).
- Generalize `firstIntOfArray` into `intsOfArray(v any, n int) []int` returning up to `n` ints (Albion's array may be short for newly-spawned players — pad with zeros).
- Continue to use slot 0 (MainHand) for classification — don't change that behavior.
- The `pendingEquip` cache today stores a single `mainHand int` keyed by `ObjectId` (the local-player race fix). It needs to store the full `[10]int` array so IP can be computed when `handleJoinResponse` replays the cached equipment. Update `cachePendingEquip` / `takePendingEquip` / `applyCachedEquipment` to carry an array, not an int.

### 2b. Item Power calculation

Per the Albion wiki and community tools (sources at the end of this prompt), IP = tier base + enchantment + quality + type bonus, averaged across "real" equipped slots (excluding consumables/mount/bag).

Implement in `agent/internal/gamedata/itempower.go` (new file):

```go
// ItemPowerOf parses a uniquename like "T6_2H_DUALCROSSBOW_AVALON@2" and
// returns its IP contribution. Returns 0 when the name doesn't parse —
// callers treat that as "empty slot, skip in the average".
func ItemPowerOf(uniqueName string, quality int) int
```

Tier base (each tier adds 100 IP from T4=700 upward — confirmed against the [Albion Online Wiki](https://wiki.albiononline.com/wiki/Item_Power)):
- T2 = 500, T3 = 600, T4 = 700, T5 = 800, T6 = 900, T7 = 1000, T8 = 1100

Enchantment bonus (`@1`, `@2`, `@3`, `@4` suffix): +100 IP per level.

Quality bonus: Normal=0, Good=+10, Outstanding=+20, Excellent=+50, Masterpiece=+100.

Type bonus (substring match against the uniquename, upper-cased): Artifact and Avalonian items are higher base — the wiki's exact numbers vary by patch, so:
- If the uniquename ends in `_AVALON` → +200
- `_HELL` / `_UNDEAD` / `_MORGANA` / `_KEEPER` → +100 (artifact)
- otherwise → 0

These are starting values; expose them as `var IPBaseByTier`, `var IPEnchantBonus`, `var IPQualityBonus`, `var IPTypeBonus` so they're easy to tune without recompiling logic.

**Quality**: Albion sends the item Quality byte in NewCharacter/CharacterEquipmentChanged but you haven't been decoding it. Look in SAT's source (already in this repo at `src/StatisticsAnalysisTool/`, search for `Quality` in equipment handlers) for the exact param index. If you can't find it cleanly, ship with `quality=1` (Normal) as a fallback — IP will under-report by ~5–10% on geared players but the chip still beats the 3-letter abbreviation.

### 2c. Aggregating IP

In `engine.go::applyEquipment`, after caching the full equipment array, compute average IP over the 6 "core" slots: MainHand, OffHand, Head, Chest, Shoes, Cape. Skip slots with `index == 0` and skip Bag/Mount/Potion/Food. Store in `ent.ItemPower`. Albion's own "average IP" matches this slot subset.

When MainHand and OffHand are both filled but the MainHand is a 2-handed weapon (uniquename contains `2H_`), Albion counts the 2H as occupying both slots — divide the 2H's IP by 2 and count it twice, or equivalently leave the OffHand at 0 and include it as a 0-contributing slot. Do whichever is simpler to reason about — document the choice in a comment.

### 2d. Wire + render

- Add `ItemPower int json:"itemPower,omitempty"` to `PlayerSnapshot` in `agent/internal/domain/snapshot.go`. Populate from `m.ItemPower` in the snapshot builder.
- Add `itemPower?: number` to the `PlayerSnapshot` interface in `web/src/types.ts` and `FightPlayerArchive`.
- Replace the `<ClassChip>` in `web/src/MeterTable.tsx` (the small 22-28px square that prints `XBW`) with a new `<IPChip>` component (new file `web/src/IPChip.tsx`) that:
  - Renders the numeric IP in mono, right-aligned, with the same per-weapon accent color as the chip used to have (call `classAccent(player.classCode, roleKey)` to keep the color tie).
  - Falls back to `"—"` when `itemPower` is 0 or missing.
  - Keeps the same width footprint so the grid template doesn't shift.
- Keep `classCode` in the data — `classAccent()` still uses it for bar/DPS tinting and the row's color identity. You're just not rendering the three-letter chip text anymore.
- The DrillIn header also renders `ClassChip` — replace there too.

The expected before/after is the player row going from `01 [XBW] Goldpipe — ADEPT'S ARCLIGHT BLASTERS` to `01 [1320] Goldpipe — ADEPT'S ARCLIGHT BLASTERS`, where 1320 is the numeric IP.

---

## Part 3 — Resource utilization

Read first, then file a short plan inline at the top of each PR describing what you measured. Don't blanket-apply micro-optimizations; pick the wins that matter on a 5v5+ ZvZ.

### 3a. Hot-path lookup

`Store.byObjectIdLocked(id)` is a linear scan over `s.byGuid`. It's called on every `HealthUpdate` (twice — for causer + affected), every `MountStart`, and every `CastFinished`. On a ZvZ where you might see 1000+ HealthUpdates per second across all visible players, that's `O(N×events)` where N is "every player ever seen in this session." Add `byObject map[int64]*Entity` to `Store`, maintain it alongside `byGuid` in every entity-mutating operation (UpsertByGuid, MountStart binding, etc.), and switch lookups to `O(1)`. Cross-check every `e.byGuid[g]` site for missed updates; the existing tests (if any) should still pass.

### 3b. Snapshot pacing

The push client ticks at 250 ms and rebuilds the **entire** snapshot — players, spells, sessionSpells, targets, assists, activeEffects copies, fight history copy, events copy — even when nothing changed. Two concrete wins:

1. **Skip the push when there's nothing new**: track a "dirty" generation counter in `Engine`. Every Photon handler that mutates state increments it. The push client remembers the last sent generation. When current == last and there's no fight-elapsed change visible to the user, skip the send (the worker will keep last state and re-serve it to viewers).
2. **Pre-compute the heavy per-entity tables once per dirty generation**, not per push. `topSpells`, `sessionSpells`, `topTargets`, `topAssists` all read maps and sort. Cache the result on the Entity (e.g. `cachedSpells []SpellBreakdown` with a `cachedSpellsGen uint64`); invalidate when the corresponding map mutates. The snapshot builder then just reads slices.

Both changes are invisible to the worker and the web — the wire format is unchanged.

### 3c. JSON marshal cost

`json.Marshal(env)` runs on every send. The snapshot is a big nested struct with `omitempty` on lots of slices. Worth one experiment: switch the snapshot send to `bytes.Buffer` + `json.NewEncoder(buf).Encode(env)` reused across iterations (cheaper than fresh `Marshal` for chatty pumps), or feed the worker a pre-marshaled message directly when nothing's changed. **Measure first** — a `time.Since` log around the marshal call for 30s of agent runtime is enough to know if this matters.

### 3d. Worker fan-out

`MeterRoom.webSocketMessage` rebroadcasts each snapshot string to every viewer in a tight `for` loop. Cloudflare's WebSocket Hibernation is cheap, but if a single viewer's socket is slow you're holding the agent's incoming `webSocketMessage` until every viewer finishes its `.send()`. Wrap each viewer send in a fire-and-forget via `this.ctx.waitUntil(Promise.resolve())`-style — or at minimum, swallow individual exceptions cleanly (already does) and don't await sequentially. This is a low priority since "typical" viewer count is 1 (the user's own browser), but it's a one-line cleanup.

### 3e. Web render

`MeterTable` does `[...players].sort(...)` and `reduce(...)` every render. Wrap them in `useMemo` keyed on `snapshot.generatedAt` + `mode`. Also `RowTooltip` mounts and unmounts on every hover — fine, but the `createPortal` to `document.body` re-attaches a fresh node each tick the snapshot updates; consider memoizing the tooltip body separately from the player it renders for so re-renders don't reflow.

These are all small wins individually. Together on a busy fight they should drop the agent's CPU footprint noticeably and cut JSON throughput when nothing's happening.

---

## Part 4 — Feature backlog drawn from competing trackers

Survey of community tools and what they do well that this meter doesn't yet do. After parts 1–3 land, **propose** which of these to build next — don't auto-build them. Each entry below is sized as "small" (≤ 1 day), "medium" (~ 1 week), or "large" (cross-cutting).

**Statistics Analysis Tool (SAT) — the C# reference fork**

- *Loot logger* (medium): record items picked up during a session, tagged with zone + fight number. SAT's table view is the pattern.
- *Trade monitoring* (medium): mail + market sales/purchases tracked into a session-level P&L. Useful for "did this dungeon pay?" framing.
- *Map history* (small): a running list of zones entered with timestamps. The agent already captures `JoinFinished`, so this is a UI add.
- *Dungeon tracker* (medium): solo/group/avalonian/mists, with timers + fame/silver/respec scoped per run. The session economy already tracks deltas; needs a per-run scope.
- *Party builder w/ IP* (small once Part 2 lands): a roster panel showing each party member's IP and class chip side-by-side, so you can sanity-check a ZvZ comp before committing.

**Albibong**

- *Auto-detection of dungeon name + duration*: already partially in the agent (zone label). The piece that's missing is the run's start/end edges; `JoinFinished` into a dungeon map followed by a `JoinFinished` out is the trigger.
- *Cross-platform support*: not relevant — the agent is Windows-only by Photon capture necessity. Skip.

**Murder Ledger / AlbionOnline2D**

- *Killboard integration* (medium): when a player dies (`EventDied`), look up the killboard entry by name on the public killboard API and link the fight archive to it. Pulls in build, IP, gear graveyard, vs. opponent gear. https://murderledger.albiononline2d.com/
- *Twitch VOD lookup by timestamp* (large): nice-to-have for streamers. Out of scope for solo play; deprioritize.

**Albion Online Grind**

- *Item Power calculator embedded in tooltips* (small once Part 2 lands): hovering a player's IP chip shows the per-slot breakdown (MainHand 1320, Cape 1100, etc.) with the same math the agent used. Mirrors albiononlinegrind.com/item-enchanting.

**Albion Online Data Project (AODP) — albion-online-data.com**

- *Live market price overlay* (medium): for the user's own activity, when they pick up a Tier 6+ item or a noteworthy artifact, hit the AODP API and surface "this is worth ~X silver in Caerleon right now." No need to upload prices — only consume them.
- *Session value bar* (small): on top of the existing session strip, total estimated silver value of loot picked up this session. Depends on the loot logger.

**Crossroads to consider**

- *Cloud history per-token*: today the worker keeps the latest snapshot only. Persisting recent fights to a KV store, keyed on token, would let the user open the URL on their phone and see what just happened. Light feature, real privacy footprint (token = identity).
- *Multi-agent same-room*: if two party members both run the agent against the same token, the worker's `MeterRoom` would receive snapshots from both. Today it just clobbers `latest`. A "merge by name/guid" view would be a real differentiator vs. SAT.

After parts 1–3 are in, write the part-4 proposal as a short markdown file at `proposals/feature_backlog.md` with one bullet per feature: name, sketch, file paths, estimated size. Don't implement; wait for review.

---

## Sources

- [Albion Online Wiki — Item Power](https://wiki.albiononline.com/wiki/Item_Power)
- [Albion Online Wiki — Quality](https://wiki.albiononline.com/wiki/Quality)
- [Albion Online Wiki — Enchanting](https://wiki.albiononline.com/wiki/Enchanting)
- [Statistics Analysis Tool (Triky313)](https://github.com/Triky313/AlbionOnline-StatisticsAnalysis)
- [Albibong](https://github.com/imjangkar/albibong)
- [Albion Online Stats (mazurwiktor)](https://github.com/mazurwiktor/albion-online-stats)
- [Murder Ledger](https://murderledger.albiononline2d.com/)
- [Albion Online Grind](https://albiononlinegrind.com/)
- [Albion Online Data Project](https://www.albion-online-data.com/)
- [AlbionKit](https://albionkit.com/)
- [Albion Free Market](https://albionfreemarket.com/)
