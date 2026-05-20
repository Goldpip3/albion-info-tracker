# Architecture & Inner Workings — Goldpipe's Data Analytics

This document is the deep-dive reference for the `go-port` stack. The
quickstart and current-state view lives in `CLAUDE.md`.

Three boxes, two WebSockets, one tab.

```
   ┌─────────────┐    UDP        ┌─────────────────┐  WSS ingest   ┌─────────────────┐  WSS view  ┌──────────────┐
   │ Albion      │ ─────────────▶│ Go agent        │ ─────────────▶│ Cloudflare      │ ─────────▶ │ React web    │
   │ client      │   (Photon)    │ (Windows .exe,  │   snapshots   │ Worker + DO     │  snapshots │ UI (Pages)   │
   │ (any zone)  │               │  raw sockets)   │   every 250ms │ (per-token room)│            │              │
   └─────────────┘               └─────────────────┘               └─────────────────┘            └──────────────┘
```

The agent is the only piece that touches Albion. It speaks **Photon
Protocol18** over UDP, parses events, maintains the entity / fight /
session state machine, and pushes one JSON `Snapshot` per tick to a
Cloudflare Worker. The Worker is a thin fan-out that holds the latest
snapshot in a Durable Object and replays it to any viewer who connects
with the matching token. The frontend is purely a renderer — no
computation lives there beyond sparkline buffers and sort.

---

## 1. Capture (`internal/capture/`)

Windows-only. Uses Winsock raw sockets with `SIO_RCVALL` to read every
UDP packet on the local machine. Requires admin (the agent ships a UAC
manifest in `agent.manifest`).

- Bound to a real network adapter; receives **all** UDP traffic, then
  filters by Albion's known ports (5055 / 5056) in `internal/photon/`.
- Returns `Packet{Payload []byte, Source net.Addr}` via a `SinkFunc`
  callback. The agent doesn't reassemble at the IP layer; Windows does
  that before delivery.

---

## 2. Photon Protocol18 (`internal/photon/`)

Albion uses the Photon Server engine. Wire format:

- **Outer envelope** is **big-endian**. Contains transport header (peer
  ID, command count, etc.) and a list of commands.
- **Inner Protocol18 message** is **little-endian**. Carries the
  application-level payload: events, operation requests, operation
  responses.
- A **signature byte** (typically `0xF3`) sits between the command
  header and the message-type byte in `SendReliable` payloads. Easy to
  miss when porting; we skip over it.

### Event / operation codes >255

The transport-level `EventData.Code` is a single byte (often just `1`).
The **actual** application code lives in parameter `252` (events) or
`253` (operations) when the value exceeds 255. Our `realCode()` helper
in `engine.go` checks both 252 and 253 first and falls back to the
transport byte.

This is why `CharacterEquipmentChanged` (Photon code **90**) shows up
in verbose logs as `... 252:90` — that's the param we trust.

### Custom types

Protocol18 type 19 carries a 1-byte typeCode + length-prefixed bytes:

- **GUID** = 16 bytes. Byte order matches .NET's `Guid(byte[])` — the
  first 4 bytes are byte-swapped relative to the textual form.
- Used by `paramGuid()` and `Guid` type in `internal/domain/`.

---

## 3. Domain layer (`internal/domain/`)

The entire game state lives here. No I/O.

### Entity store (`entity.go`)

`Store.byGuid` is the canonical map keyed by `UserGuid` (the stable
identity). `ObjectId` is the **transient session-level handle** Albion
uses for network references — it changes every zone load. We track
both because:

- `HealthUpdate` events reference players by `ObjectId`.
- `Party*` events reference players by `Guid`.
- We need to join the two streams.

Mount events (`MountStart` + `NewMountObject`) are the secondary join
path — they give us `ObjectId ↔ Guid` mapping for party members the
agent saw mid-zone after `PartyJoined` had already fired.

### Fight lifecycle (`engine.go`)

- Combat-enter on first `HealthUpdate` damage tick.
- `combatEnterIdle = 4s` of no damage → "out of combat" flag flips
  (lazy, on next `FightStatus` call).
- `fightAutoEnd = 6s` of no damage → fight ends, **archive** the
  current state into `fightHistory` (cap 20), zero every entity's
  `Current` bucket, bump `fightNumber`.

`Current` resets between fights. `Overall` persists for the session.
`BySpell` (per-fight) resets; `BySpellSession` does not.

### Session economy (`session.go`)

Four FixPoint-internal counters (10,000 = 1 unit):

- **Fame** — delta-tracked from `UpdateFame` param 1 (TotalPlayerFame).
- **Silver** — delta-tracked from `UpdateMoney` param 1 (CurrentPlayerSilver).
- **Respec** — per-event gained (param 2) OR lifetime baseline-delta
  (param 0 array element 1) — whichever is non-zero. See
  `chooseRespecGain()` for the reconciliation logic.
- **Might** — per-event gained from `MightAndFavorReceivedEvent` param 1.

Frontend divides all four by 10,000 before display.

### Equipment race + the `pendingEquip` cache

The single most important bug we fixed during the port:

```
Albion server               Agent receives
─────────────               ──────────────
CharacterEquipmentChanged   ObjectId 11069 has no entity yet → was DROPPED
CharacterEquipmentChanged   (fires twice for redundancy)     → was DROPPED
Join response               ObjectId 11069 ↔ Goldpipe registered
JoinFinished
NewMountObject
```

Equipment events for the **local player** arrive BEFORE the Join
response that registers them. The naive handler — "look up entity by
ObjectId, set equipment" — silently drops them, and the local player's
class chip stays blank forever.

**Fix**: `handleEquipmentChanged` now writes the MainHand item index
into `e.pendingEquip[objectId]` regardless of whether the entity
exists yet. `handleJoinResponse` (and `handleNewCharacter`, for
symmetry) calls `applyCachedEquipment(ent)` after upserting, which
takes any pending value and classifies the entity.

### Debuff-window attribution ("Level 2 assists")

Two state objects feed the system:

1. **`recentCasts`** — ring buffer (cap 64) of `(spellIdx, casterId, time)`
   from every `CastFinished` event. Used to attribute a debuff to its
   caster when the buff appears on a target.
2. **`activeWindows[targetId][spellIdx]`** — open intervals.
   `ActiveSpellEffectsUpdate` ships the **full** active set on a
   target. We diff against the prior set:
   - new effect → open window with caster from `recentCasts` (2 s lookback)
   - removed effect → close window, accumulate into the caster's
     `AssistsBySpell[spellIdx].UptimeMs` and `.DamageDuring`.
3. Every damage hit on a target charges any open windows on that
   target via `chargeWindowsLocked()`.

What it means: "while Player A's Frazzle was up on Boss X for 18 s,
the party dealt 1.8 M damage to Boss X." Doesn't claim Player A
**caused** 1.8 M — that's the Level 3 problem (multiplier table) and
not yet built.

### Snapshot wire format (`snapshot.go`)

`Engine.Snapshot()` reads under `store.mu.RLock()` and produces:

```go
type Snapshot struct {
    GeneratedAt time.Time
    Players     []PlayerSnapshot   // sorted/limited by frontend
    Composition Composition         // T/H/R/M/S/C/Unknown counts
    Fight       Fight               // number, elapsedMs, inCombat, zone
    Session     Session              // Fame/Silver/Respec/Might/Deaths
    Recent      []FightArchive      // last 20 completed fights
    Events      []ActivityEvent     // last 48 hit/heal/death log entries
}
```

`PlayerSnapshot` includes everything the frontend needs to render a
row: identity, class chip, current+overall damage/heal/taken, top
spells (current fight + session), top targets, active buff/debuff
indices, and the Assists table.

JSON is pushed verbatim by the agent every 250 ms; the Worker stores
the latest snapshot in DO memory and replays to viewers.

---

## 4. Game data (`internal/gamedata/`)

Albion ships its game definitions as encrypted+compressed XML in `.bin`
files under `Albion-Online_Data/StreamingAssets/GameData/`.

### Decryption (`decrypt.go`)

DES-CBC with a well-known key + IV (`decrypt.go` has both), followed
by gzip decompression. The bytes are then plain UTF-8 XML.

### items.bin loader (`items.go`)

**This is the trickiest game-data file.** Albion's protocol assigns
items numeric IDs by document-order index in this XML, but with two
critical wrinkles:

1. **Enchantment variants count.** For every base item with an
   `<enchantments>` child, each enchantment level (`@1`, `@2`, `@3`,
   `@4`) consumes a separate index. So `T6_2H_CROSSBOW` at index N
   becomes `T6_2H_CROSSBOW@1` at N+1, etc.
2. **Journal items get `_EMPTY` + `_FULL` post-pass entries.** Every
   `<journalitem>` allocates two additional indices after the main
   walk completes.

If you skip these (as a naive `for range XML` would), you stop at ~5800
entries and never index Tier 8 enchanted weapons whose IDs live in the
6000-12000 range. Goldpipe's `T4_2H_DUALCROSSBOW_CRYSTAL@2` lands at
index **6623** — out of range for a 5800-entry loader.

Algorithm (mirrors SAT's `StatisticAnalysisTool.Extractor.ItemData.cs`):

```
idx := 1
for each top-level XML element:
    if no `uniquename` attribute, skip subtree
    record entry at idx with uniquename (+ "@<level>" if non-zero enchantmentlevel)
    idx++
    if element is journalitem, remember uniquename
    walk subtree for <enchantments>:
        for each <enchantment> child:
            record entry at idx as "<parent uniquename>@<level>"
            idx++
for each journal uniquename:
    record entry at idx as "<name>_EMPTY"; idx++
    record entry at idx as "<name>_FULL"; idx++
```

Result: 12,060 entries for current Albion patch.

### spells.bin loader (`spells.go`)

Same DES+gzip+XML format. Each `<spell>` element has a `uniquename`
attribute and an implicit document-order index. We index every element
1:1. Result: 8,856 entries.

### localization.bin loader (`localization.go`)

TMX (Translation Memory eXchange) format:

```xml
<tu tuid="@SPELLS_CROSSBOW_FLICKERSHOT_E">
  <tuv xml:lang="EN-US"><seg>Flickershot</seg></tuv>
  <tuv xml:lang="DE-DE"><seg>...</seg></tuv>
</tu>
```

We keep only `EN-US`. Lookup helpers:

- `loc.ItemName(uniquename)` looks up `@ITEMS_<uniquename>` (strips any
  `@<level>` suffix first — localization is per base item).
- `loc.SpellName(uniquename)` looks up `@SPELLS_<uniquename>`, then
  falls back to `@MOB_ABILITIES_<uniquename>`.

Result: 38,174 EN-US strings.

### Weapon classifier (`role.go`)

`ClassifyWeapon(uniquename)` returns `{Code, Role, Label}`. The Code
(3-letter chip text) and Role (T/H/R/M/S/C/?) come from substring
matching the uniquename: "DUALCROSSBOW" → `XBW / RangedDPS`,
"HAMMER" → `HAM / Tank`, etc.

Order matters: artifact names match first (Hellish/Morgana/Avalon
variants), then the generic family. Crystal League weapons fall
through to generic.

The `Label` is the generic family name ("CROSSBOW"). The engine layer
prefers the localized item name when available:

```go
label := c.Label                       // "CROSSBOW"
if loc := e.loc.ItemName(uniquename); loc != "" {
    label = strings.ToUpper(loc)       // "ADEPT'S ARCLIGHT BLASTERS"
}
```

`InferClassFromSpell(spellName)` is a fallback for the rare case where
equipment never arrives but the player casts a spell — most spell
uniquenames contain the weapon family ("CROSSBOW_FLICKERSHOT_E").

---

## 5. Cloudflare Worker (`worker/`)

Single TypeScript Worker. Durable Object class `MeterRoom` (SQLite-backed
namespace, but used purely for in-memory state).

### Token → room hashing

```ts
const roomId = sha256(token).slice(0, 32);
env.METER_ROOM.idFromName(roomId);
```

A token is just an opaque string. Anyone who knows it joins the same
room. No DB, no signup. Rotate the token to rotate the room.

### MeterRoom internals

- Uses **WebSocket Hibernation API**: `ctx.acceptWebSocket(socket, [tag])`
  and `ctx.getWebSockets(tag)`. The DO can sleep with sockets attached
  and Cloudflare re-wakes it on incoming data.
- Tags: `"agent"` for ingest sockets, `"view"` for viewer sockets.
- On viewer connect: replay the latest cached snapshot immediately
  (no waiting for the next push).
- On agent → snapshot: cache + fan out to all "view" sockets.
- On viewer → command (e.g. `{"type":"command","action":"resetSession"}`):
  forward to the "agent" socket.

State is in-memory only. Hibernation losing it is fine — agent
republishes within 250 ms.

---

## 6. React UI (`web/`)

Vite + React 19 + Tailwind v4. ~12 components.

### Top-level structure (`App.tsx`)

```
App
├── LiveApp (real WebSocket, default)
└── DemoApp (?demo=1 — hand-crafted snapshot from demo.ts)
   ├── Header
   │  ├── TitleBar (GDA icon + breadcrumb + AgentPill + buttons)
   │  └── FightHeader (combat status + fight time + zone + tabs + summary stats)
   ├── SessionStrip (FarmStrip — 4 cards with sparklines)
   ├── <main>
   │  ├── MeterTable × N panes (Damage / Healing / Tank)
   │  └── ActivityLog (toggleable)
   ├── Footer (tick rate + last update + version)
   ├── SettingsPanel (modal)
   └── DrillIn (per-player modal, 4 tabs)
```

### Class accent palette (`format.ts`)

`classAccent(classCode, roleKey)` is the single source of truth for
per-row colour:

```
DGR / GRT / SWD / AXE / SPR  → #ff6464 (red — melee DPS)
LBW / WBW / BOW / XBW / FIR / CRS → #ffb43a (amber — ranged DPS)
FRO                          → #6fd8ff (cyan — control)
HAM / MAC / QRT / KNK        → #8aa3ff (steel — tank)
ARC                          → #c08cff (violet — support)
HLY / DVN / RED              → #ffd770 (gold — radiant healer)
NTR / WLD / FAL              → #5cf0a4 (green — life healer)
unknown                      → #6a6a76 (dim grey)
```

Drives the chip border + fill, bar outline + tint, DPS column colour,
and role-label colour on every row.

### Sparkline rate-pulse (`SessionStrip.tsx`)

Each card's sparkline plots **per-second gain**, not cumulative total.
A 1 Hz interval samples the live value, diffs against the previous
sample, pushes the delta into a 22-element ring buffer. Out of combat
samples sit near zero; spikes line up with kills / loot drops.

Reset when `session.startedAt` changes (the agent fired ResetSession).

### Demo mode (`demo.ts`)

Activated via `?demo=1`. Returns a hand-crafted `Snapshot` with 10
fake players covering every class. Used to verify visual changes
without standing up a real combat session.

```powershell
cd web; npm run dev
# http://localhost:5173/?demo=1
```

### LocalStorage keys

- `gda:settings:v2` — UI prefs (accent, density, bar style, pinned
  panes). Bump the `v2` suffix on token-breaking design changes to
  force returning users back to defaults.
- `skirmish:url`, `skirmish:token` — pairing config from the first-run
  wizard. Legacy key names (kept for backwards compatibility).

---

## 7. Magic-link pairing flow

1. User double-clicks `GDA Launcher.lnk`.
2. Shortcut runs `agent.exe --open-browser`.
3. First-run path (`config.PushToken == ""`):
   - Generates a fresh 32-hex token via `crypto/rand`.
   - Writes `agent.json` with `pushUrl + pushToken + albionInstallRoot`.
   - Opens the browser to `https://albion-meter-web.pages.dev/?pair=<token>`.
4. Frontend reads `?pair=<token>` from `window.location.search`, saves
   it to `localStorage["skirmish:token"]`, strips the param via
   `history.replaceState`, and dials the WebSocket.
5. Same agent on subsequent launches: token already saved, browser
   opens to the bare URL, frontend uses the saved token.

A friend can share the launcher (+ agent.exe + agent.json) and the
auto-pair flow puts them in their own meter room.

---

## 8. Key bugs we fixed during the port (chronological)

- **Token-in-history (commit 6ae176a)** — agent.json was committed
  briefly; token rotated to `cd80e1d8...`, file added to .gitignore.
- **Mount-event race** — ObjectId↔Guid binding within a 2 s transient
  window. Mirrors the same fix on `sat-fork`.
- **Local player class never resolves** — `CharacterEquipmentChanged`
  arrives before Join response. Cached `pendingEquip` replays on
  upsert.
- **Items.bin index range** — naive loader missed enchantment variants
  + journal pairs, capped at 5808. Now produces 12,060 to match SAT.
- **Spells.bin index range** — same shape of bug: naive loader walked
  ALL nested elements via recursive token stream. SAT counts only
  top-level passivespell / activespell / togglespell; activespell with
  a `<channelingspell>` child claims TWO consecutive indices. Result:
  every CausingSpellIndex on the wire mapped to the wrong uniquename
  for high-tier abilities. "Explosive Bolt" (BOLTSHOT) is at 3021 now,
  not 2953.
- **Tooltip name mismatch** — uniquenames are internal; in-game names
  come from localization.bin. Loaded and wired through.
- **Silver tracked via wrong event** — we listened to `UpdateMoney`
  (wallet sync, fires on deposits/purchases) instead of `TakeSilver`
  (loot pickup). Loot silver was missing from the dashboard. Switched
  to `TakeSilver` with `YieldAfterTax = YieldPreTax - GuildTax`,
  matching SAT.
- **Fame under-counted** — delta-tracking `TotalPlayerFame` lagged the
  in-game popup by ~1 tick and dropped premium / satchel / bonus
  multipliers. Switched to SAT's per-event
  `(FameWithZoneMultiplier + Premium + Satchel) × Bonus`.
- **Frost Staff misclassified as RangedDPS** — reclassified to
  RoleControl to match Albion's vocabulary.
- **Healer accent split** — Holy Staff (gold) vs Nature Staff (green)
  now distinguish via per-weapon accent table instead of shared
  RoleHealer colour.
- **Row shuffling** — `BySpell` map iteration is randomized in Go;
  sort tie-breaks on `(damage, casts, index)` for determinism. Same
  fix applied to the player-row sort.
- **Tooltip clip** — drill-in row hover tooltip portaled to
  `document.body` so it doesn't clip inside narrow pane containers.
- **Fame divisor missing** — `UpdateFame` param 1 is FixPoint
  (10,000 = 1 fame); frontend was treating it as raw. Divided.
- **Respec inflation** — `UpdateReSpec` param 2 may carry cumulative
  on some patches. `chooseRespecGain()` reconciles between per-event
  gained (param 2) and lifetime baseline-delta (param 0 array[1]),
  preferring the smaller positive value.
- **Settings persist across token bumps** — bumped the localStorage
  key from `skirmish:settings` to `gda:settings:v2` after defaults
  changed (density 28 → 40).

---

## 9. Diagnostic playbook

### Agent says "items.bin OK — 5808 entries" instead of 12,060

You're running an old binary. The loader was rewritten in commit
`517eb06` to expand enchantment variants. Rebuild:

```powershell
cd agent
go build -o agent.exe ./cmd/agent
```

### Local player class chip stays `—`

1. `cd agent; go run ./cmd/probe` — confirm items.bin loads and the
   item index for the user's MainHand is in range.
2. Run agent with `ALBION_AGENT_VERBOSE=1` (admin PowerShell):
   ```
   Get-Content C:\tmp\gda.log | Select-String "CharacterEquipment|Join response"
   ```
3. If `CharacterEquipmentChanged` fires for the user's ObjectId but
   class doesn't resolve → `applyCachedEquipment` regression. Check
   `pendingEquip` state.
4. If `CharacterEquipmentChanged` fires for OTHER players but not the
   local user → Albion patch broke local broadcast. Fall back to
   `InferClassFromSpell` (already wired).

### "Adept's Arclight Blasters" vs "Crystal Dual Crossbow"

The label comes from `loc.ItemName(uniquename)`. If localization.bin
doesn't load, we fall back to `ClassifyWeapon`'s generic label. Check
the boot output:

```
localization.bin OK — 38174 EN-US strings
```

### Sparklines flat or jittery

Sparklines plot per-second gain. If the agent isn't pushing
`session.fameTotal` deltas (e.g. you're not killing anything),
they'll stay flat. That's correct.

### Worker says "stale" but agent is running

Check the latest deployment URL. The production alias auto-updates
but per-deployment preview subdomains are static. Open
`https://albion-meter-web.pages.dev` (no hash).

---

## 10. Persistence and run-scoping (F1–F5)

After the initial damage-meter MVP, four feature families layered onto
the engine to give the agent a longer memory.

### F1 — Session persistence (`internal/domain/sessions.go`)

When the user clicks **"New session"**, before zeroing counters the
engine writes the prior session's metadata to
`%LocalAppData%\GDA\sessions\<unix-ms>.json` (or
`$XDG_CACHE_HOME/gda/sessions` on POSIX). The file carries final
fame/silver/combatFame/might/deaths, fight count, zone, local player
name. Boot rescans the directory; the snapshot exposes the cached list
in `snap.sessions`. The web `SessionsPanel` renders that list with
per-row delete; deletion goes through the existing command channel as
`{action: "deleteSession", arg: <id>}`.

No cloud, no token-tied identity — everything stays on the user's
machine.

### F2 — Zone history (`internal/domain/zones.go`)

`zoneLog` is an in-memory ring buffer (cap 25) of `ZoneVisit{Name,
EnteredAt, LeftAt, DurationMs}`. Updated from `handleJoinResponse`
after the zone name is prettified. De-dupes consecutive same-zone
re-entries (Albion fires `JoinResponse` on respawn into the same map).
Snapshot exposes the list in `snap.zones`.

### F3 — Party panel + live equipment/spell tracking

`CharacterEquipmentChanged` already updates `Entity.Equipment[10]`. We
extended `parseEquipmentParams` to also read the **active spells array**
(param 7, `short[14]`) into `Entity.ActiveSpells`. Snapshot exposes a
resolved per-slot view via `equipmentSlots[]` (slot label + name + IP)
and `activeSpellSlots[]` (slot key + localized spell name). The web
`PartyPanel` modal renders one row per player with chips for every
bound ability and equipped piece — automatically refreshing on every
push (every 200 ms when state changes).

Slot key mapping for spells (mirrors SAT):
- 0/1/2 → MainHand Q / W / E
- 3 → Armor (chest active)
- 4 → Head
- 5 → Shoes
- 12 → Potion
- 13 → Food

### F4 — IP per-slot tooltip

The `IPChip` (which replaced the legacy `ClassChip` text in player
rows) now portals a tooltip on hover showing MainHand 1340 / OffHand —
/ Head 1280 / Chest 1300 / Shoes 1310 / Cape 1290 with the localized
item names. Bag/Mount/Potion/Food are hidden — they don't contribute
to the averaged IP.

### F5 — Dungeon tracker (`internal/domain/dungeon.go`)

`classifyDungeon(rawMapIndex)` recognises Albion's instance patterns:
- `@HELLGATE`, `AVALON_ROAD`, `@AVALONIAN`
- `@RANDOMDUNGEON_SOLO` / `SOLODUNGEON`
- `@RANDOMDUNGEON_GROUP` / `GROUPDUNGEON`
- `MISTS_` / `@MISTS`
- `@CORRUPTED`

On a JoinResponse into one of these zones, `openDungeon` stamps a
`DungeonRun` with baselines = current session counters. On exit (any
non-dungeon `JoinResponse`), `closeDungeon` freezes the end time.
`CurrentDungeon()` returns a snapshot-friendly copy with live deltas
(current session counters minus baselines), so the
`DungeonStrip` above the meter shows "what this run paid" in real
time.

## 11. Loot logger + AODP prices (F6 + F7)

### Loot capture (`internal/domain/loot.go`)

`OtherGrabbedLoot` (event 285) fires when ANY visible player loots
something — including the user. Params:
- 1: lootedFromName (corpse owner / mob name)
- 2: looterByName (the friend or you)
- 3: isSilver (bool)
- 4: itemIndex (items.bin index, 0 when isSilver)
- 5: quantity

`handleOtherGrabbedLoot` builds a `LootEntry` with the resolved
in-game name (via items.bin + localization), tags it with the current
zone + dungeon id, marks `LooterIsLocal` when the name matches the
local entity. Stored in a ring buffer (cap 500). Snapshot exposes
`snap.loot` (chronological) + `snap.looterTotals` (per-looter rollup,
local first then by value desc).

### Price client (`internal/aodp/client.go`)

`aodp.Client` polls the Albion Online Data Project HTTP API for market
prices. Default endpoint: `https://west.albion-online-data.com/api/v2
/stats/prices/<uniquename>.json`. 5-minute per-item TTL, throttled to
1 request/second (their published limit). All cached in memory only,
no disk persistence.

When a non-silver loot event lands, `noteLoot` enqueues a price fetch
for the item. The next time `LootLog()` is called for the snapshot,
cached prices are applied to compute `silverValue = price × qty`. If
the agent's offline or the item isn't priced yet, value stays 0 — the
loot still logs.

`LooterTotalsList` rolls up by looter name. Each `LooterTotals` row
carries `ItemCount`, `SilverTotal` (direct silver pickups), and
`ValueTotal` (silver + estimated item value). The web `LootPanel`
renders "Totals" and "Items" tabs with a party-total footer.

## 12. What's deliberately not built

- **Trade / market / mail / guild events / harvesting** — out of scope.
  SAT does these; we are damage-meter-plus-loot.
- **HTTP gameinfo API calls** for player profile data — SAT hits
  `https://gameinfo.albiononline.com/api/gameinfo/players/<name>`. We
  have everything we need from the Photon stream + game-data files.
  AODP for market prices is the only HTTP dependency we accept.
- **Crit % per ability** — Albion doesn't expose a crit flag in
  `HealthUpdate`; SAT doesn't track this either.
- **Level-3 attribution math** — multiplying assist damage by the
  exact debuff modifier (e.g. Frazzle = +30%). Would require a
  hardcoded multiplier table per debuff; current "damage during
  window" is the honest version.
- **Multi-language UI** — localization loader keeps only EN-US.
- **Cloud history per-token** — drafted in `proposals/feature_backlog.md`
  but not built. Real privacy footprint (token = identity).

---

## 13. Refinements after the F-series

Changes layered on after sections 1–12. Code pointers, not full rewrites.

### Batched HealthUpdates (the DPS fix)
Albion sends single hits (mostly auto-attacks) as `HealthUpdate` (EventCode 6)
but **batches** most ability / multi-source damage into `HealthUpdates`
(EventCode 7) as parallel arrays. We had no case for 7, so the bulk of damage
was dropped → DPS read near auto-attack-only. `handleHealthUpdates` decodes the
arrays (`paramDoubleArray`/`paramLongArray` in `params.go`, tolerant of every
numeric slice type incl. `[]any`) and folds each entry through the shared
`applyHealthChange` — the same path the singular handler uses, so the two can't
drift. Regression-tested in `health_test.go`.

### Silver attribution
Silver is credited **only** from `TakeSilver` (`engine.go::handleTakeSilver`),
gated to the local player. The local ObjectId comes from the bound entity if a
Join set it, else from `localObjIdHint` — learned from `UpdateMoney` param 0 (a
self-only wallet event), so silver counts even when the agent starts mid-zone
with no Join. `OtherGrabbedLoot` does **not** credit silver (it fires for the
same pickup → would double-count). `NewSilverObject` ids populate a `silverDrops`
set; a TakeSilver whose source object is in it is tagged into
`session.MobSilverTotal` (the loot-panel `mob:` breakout). `UpdateMoney` itself
is never credited (whole wallet incl. deposits/sales).

### Fight scope + last-fight carryover (`engine.go`, `combat.go`)
`touchCombat` only fires when `localInvolved(causer, affected)` — fights start /
the Current bucket resets only on the local player's combat, so distant
mob-on-mob fights don't tick the counter while idle. On a fight boundary
`resetAllCurrent` snapshots `Current → LastFight`; the snapshot reads
`Current.Or(LastFight)` so rows show the previous fight's numbers in the gap
before new damage lands instead of snapping to zero.

### Membership scope (`snapshot.go::scopedMembers`)
One shared filter for both the live snapshot **and** `archiveCurrentFight`:
`party` / `partyGuild` / `everyone` (set by the `setLootFilter` command →
`LootFilterMode()`; `ALBION_AGENT_SHOW_ALL` forces `everyone`). Because both the
live view and archives use it, a past fight shows exactly who the live meter
showed. `allowedLooters`/`looterSources` (`loot.go`) apply the same scope to the
loot panel and tag each looter `local/party/guild/friend`.

### Party persistence (`party_store.go`, `guid.go`)
`PartyJoined` is non-retroactive, so confirmed members are written to
`%LocalAppData%\GDA\party.json` (30-min freshness) on every party event + manual
edit; `RestoreParty()` rehydrates on boot. `ParseGuid` inverts `Guid.String()`
(round-trip tested) so persisted guids reparse. Manual `addPartyMember` /
`removePartyMember` / `clearParty` commands + `VisiblePlayers` in the snapshot
drive the Party tab's add/remove/clear UI.

### Capture self-heal (`main.go::superviseCapture`, `sockets_windows.go`)
`Run` now closes its sockets on ctx cancel (was deadlocking `wg.Wait` on a silent
socket). `superviseCapture` watchdogs `packetsSeen`: after traffic has flowed,
~45 s of silence cancels + reopens (re-enumerating interfaces, fixing
adapter-change / sleep stalls). First-run errors before any packet stay fatal.

### Web: navigation + panes (`App.tsx`, `TabBar.tsx`, `MeterTable.tsx`)
Top-tab routing (`/`, `/loot`, `/party`, `/sessions`) via `history.pushState` +
`popstate` in a single mounted shell. `PaneSet` became `Record<SubMetric,
boolean>` — each pane is one metric+scope, so the same metric can render twice
(Current beside Session); `MeterPanes` renders the grid, `MeterTable` is locked
to a `sub` and shows one number per row. `rankBy(selector)` (`format.ts`) is the
shared ranker (metric desc, `userGuid` tiebreak) — plus the agent sorts
`out.Players` by guid — killing the Top/Carried-by flicker. Sparklines plot an
EMA of per-second gain (`SessionStrip.tsx`), not raw deltas, for a calm pace
line. The `SessionStrip` is now the C3 editorial Fame hero; currencies are
floored to match the loot panel (the 651/652 fix).

### Agent ergonomics
`--verbose` flag (`SetVerbose` + `setupVerboseLog` teeing to
`agent/agent-verbose.log`) for diagnostic captures that can be read from a file.
`Restart GDA Agent.cmd` / `(Verbose).cmd` self-elevate, stop, rebuild, relaunch.
