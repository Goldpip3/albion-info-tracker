# CLAUDE.md

Personal fork of [Triky313/AlbionOnline-StatisticsAnalysis](https://github.com/Triky313/AlbionOnline-StatisticsAnalysis) (SAT) — a third-party Albion Online stats / damage-meter app. The repo now contains **two parallel implementations on different branches**:

- **`sat-fork`** — the original C# / WPF / .NET 10 SAT fork. Production-quality, used for actual gameplay. Two WIP commits on top of upstream (current+overall meter + mount-event ObjectId binding); both compile clean but were **never verified in-game** before the user pivoted to Go.
- **`go-port`** — clean-room Go rewrite that pushes JSON snapshots to a Cloudflare Worker and renders the meter in a React web UI hosted on Cloudflare Pages. **Currently the active development branch and confirmed working end-to-end.**

Nothing here is intended to go upstream.

**For deep architecture / inner workings, read `ARCHITECTURE.md`.** This file is the quickstart + current-state view.

## Launch Claude Code from this folder

CLAUDE.md auto-loads when Claude Code is run from the project root or a subdir. Run `claude` from `C:\Users\colom\Claude Projects\Albion DPS\`. If launched from `OneDrive\Documents\Albion Analysis\` (the production install), this file won't be loaded.

## Layout

Project root: `C:\Users\colom\Claude Projects\Albion DPS\`

```
src/                       C# SAT solution (sat-fork stack; preserved on go-port too)
agent/                     Go agent — capture + parse + state + WS push
  cmd/agent/               entrypoint
  cmd/probe/               one-shot diagnostic — loads items/spells/localization without UAC
  internal/capture/        Windows raw-socket Photon UDP capture
  internal/photon/         Protocol18 deserializer + outer parser + fragment reassembly
  internal/gamecodes/      generated EventCodes/OperationCodes enums (mirrored from SAT)
  internal/gamedata/       DES-CBC + gzip + XML decoders for *.bin game files
                           includes role.go (weapon classifier), itempower.go (IP math),
                           localization.go (TMX), spells_override.go (hand-curated names)
  internal/domain/         entity store, combat tracker, Photon event handlers,
                           session economy, sessions.go (disk archive), zones.go (map log),
                           dungeon.go (run-scoped scope), loot.go (per-looter rollup),
                           party_store.go (party.json persistence), loot_source.go
                           (@MOB_… → "T6 Harvester" prettifier), guid.go (ParseGuid)
  internal/aodp/           Albion Online Data Project price client (loot value)
  internal/push/           WebSocket push client (coder/websocket, reconnect+backoff)
  internal/config/         agent.json + env-var loader (incl. alwaysIncludeNames)
worker/                    Cloudflare Worker + Durable Object backend (TS)
web/                       React + Vite + Tailwind v4 frontend
  src/                     App.tsx (top-tab routing + MeterPanes), TabBar/tabs (nav),
                           MeterTable, SessionStrip (C3 editorial Fame hero),
                           LootBody/LootPage, PartyBody/PartyPage, SessionsBody/SessionsPage,
                           EmptyState, DungeonStrip, IPChip, DrillIn, Header, Footer,
                           PlayerPicker ("Track Players" menu under the Meter)
                           (legacy *Panel modals kept but unmounted)
  public/assets/           GDA brand mark (16/32/48/64/128/256/512 PNG + SVG)
GDA App (Local).cmd        LOCAL mode: agent serves the meter at http://localhost:8787,
                           no Cloudflare push (no request budget). restart-agent.ps1 -Local.
GDA Website.cmd            WEBSITE mode: pushes to Cloudflare, opens the hosted site.
Restart GDA Agent.cmd      one-click stop → rebuild → relaunch elevated (self-elevates).
                           Equivalent to GDA Website.cmd (cloud mode).
Restart GDA Agent (Verbose).cmd  same + --verbose → writes agent/agent-verbose.log
restart-agent.ps1          the script all .cmd wrappers call (-Trace verbose, -Local local mode)
GDA Launcher.lnk           double-click to launch the agent with --open-browser
agent/internal/localserver/  embedded web UI (webdist/) + HTTP/WS server for local mode
SAFETY.md                  passive-capture / no-overlay / no-automation disclosure
ARCHITECTURE.md            deep-dive into protocol, indexing, race fixes, data flow
proposals/                 feature backlog drafts (not yet implemented)
```

## Git

- Remote: `https://github.com/Goldpip3/albion-info-tracker` (carries the previous project's name).
- Active branch: **`go-port`** (started from `sat-fork`, no merges back).
- `origin/main` and `origin/webify` preserve the much-older `albion-info-tracker` project. Replaced with SAT source in commit `0e540a6`.
- **The `pushToken` is a per-machine secret — it lives only in `%LocalAppData%\GDA\agent.json` (gitignored), never in the repo.** An early commit briefly exposed a token in this file + `agent.json`; that token has been **rotated**, so the value still visible in git history is dead (it only ever named a now-abandoned Cloudflare room). The desktop installer ships in local mode and uses no token. To rotate again: clear `pushToken` in `agent.json` — the agent generates a fresh one and re-pairs the website on the next Website-mode launch.

## Deploy URLs

- **Web UI (production alias)**: <https://albion-meter-web.pages.dev> — always points to the latest deployment. **Use this URL, not the per-deployment `<hash>.albion-meter-web.pages.dev` previews** — those are separate browser origins and don't share the user's saved pairing token.
- **Worker**: <https://albion-meter.goldpipe.workers.dev>
- **Ingest endpoint**: `wss://albion-meter.goldpipe.workers.dev/ingest?token=<token>`
- **Viewer endpoint**: `wss://albion-meter.goldpipe.workers.dev/view?token=<token>` (consumed by the web UI)

---

## C# SAT (`sat-fork` branch, `src/`)

### Build & run

- Requires .NET 10 SDK (10.0.300 installed via winget; `global.json` pins to 10.0.100 with `latestFeature` rollforward).
- `dotnet build src\StatisticsAnalysisTool.sln` from project root.
- "DEBUG" badge in the title bar is hardcoded via `#if DEBUG` in `ViewModels\MainWindowViewModel.cs:365-367`. Release build suppresses it.
- Iteration loop: edit → `dotnet build` → close any running dev SAT (it holds the exe locked) → double-click `SAT (Dev Build).lnk`.
- Cannot verify WPF rendering from the agent side. Visual confirmation requires the user to relaunch + screenshot.
- Built exe: `src\StatisticsAnalysisTool\bin\Debug\net10.0-windows\StatisticsAnalysisTool.exe`.
- Reference production install (Triky313 v9.2.1): `C:\Users\colom\OneDrive\Documents\Albion Analysis\` — do not modify.

### What's on sat-fork beyond upstream

Two commits, both compile clean, **neither tested in-game**:

1. **Current+overall damage meter columns** (WoW Skada style).
2. **Mount-event ObjectId↔UserGuid binding** + name-preservation fix in `EntityController.AddEntity`.

Both were superseded by the Go pivot. The Go agent ports both ideas + more.

### Settings storage

`%LocalAppData%\StatisticsAnalysisTool\Instances\<id>\Settings.json`. Debug builds use the literal `Debug` instance folder; Release/path-based builds use an 8-char hash of the exe path. The user's Albion install is at `C:\Program Files (x86)\AlbionOnline` (one word — not the default `Albion Online`).

---

## Go agent (`go-port` branch, `agent/`)

Single static Windows binary. Captures Albion UDP traffic via raw sockets (`SIO_RCVALL`, requires admin), parses Photon protocol, maintains entity/combat state, pushes JSON snapshots to a Cloudflare Worker every **1 s** during activity (1 Hz), throttled to a **60 s** idle heartbeat when nothing changed (dirty-gen short-circuit). Cadence is the main lever on Cloudflare's free-tier 100k-requests/day budget — at 1/s idle the heartbeat alone burned ~86k/day just from leaving the agent running (was 400ms/15s; slowed to 1s/60s after the user hit the cap). Each push ≈ 1 request, so this is the budget knob. The web UI's "stale" grace is 70s to tolerate the 60s heartbeat.

### Local mode vs Website mode (two launchers)

The agent **always** runs an embedded web server (`internal/localserver`, bound to `127.0.0.1:8787`) that serves the same React bundle (`webdist/`, committed so `go build` works) plus a `/view` WebSocket streaming snapshots + accepting commands — the exact `push.Envelope` protocol the Worker uses. The bundle auto-detects a `localhost` origin (`isLocalMode()` in App.tsx) and connects to that local socket with no token, **bypassing Cloudflare entirely (zero request budget)**.

- **`GDA App (Local).cmd`** → `restart-agent.ps1 -Local` → agent runs with `--local`: serves localhost, **Cloudflare push OFF**, opens `http://localhost:8787`. Everyday use.
- **`GDA Website.cmd`** (≡ `Restart GDA Agent.cmd`) → no `--local`: pushes to Cloudflare AND serves localhost; opens the hosted site. For phone/remote access; uses the daily budget.

`--local` also suppresses the first-run Cloudflare pairing pop-up (local needs no token). Both modes serve localhost; the flag only gates the cloud push + which URL `--open-browser` opens.

### Build & run

**Preferred iteration loop after an agent change: double-click `Restart GDA Agent.cmd`.**
It self-elevates (one UAC prompt), stops the running agent, rebuilds, and relaunches it
elevated with `--open-browser`. The agent runs as admin (raw-socket capture) and Windows
locks `agent.exe` while running, so a manual `go build` fails until the old process exits —
the script handles all of that. **Web-only changes need no restart — just hard-refresh.**

For a diagnostic capture, double-click `Restart GDA Agent (Verbose).cmd` — it adds
`--verbose`, which tees every event to `agent/agent-verbose.log` (truncated each launch,
gitignored) so the log can be read back directly instead of scraping the elevated console.

Manual build (e.g. CI / just compiling):

```powershell
cd agent
go build -o agent.exe ./cmd/agent
go test ./internal/domain/   # ParseGuid round-trip + batched HealthUpdates regression
# Admin PowerShell — SIO_RCVALL requires elevation.
.\agent.exe
```

Flags: `--open-browser` / `-b` (open the meter on launch), `--verbose` / `-v` (debug log + file).

Diagnostic env vars:
- `ALBION_AGENT_VERBOSE=1` — per-event logging (same as `--verbose`)
- `ALBION_AGENT_SHOW_ALL=1` — power-user escape hatch: show every entity with combat activity, bypassing the scope filter (see Meter scope below)
- `ALBION_AGENT_URL=wss://...` — override push URL
- `ALBION_AGENT_TOKEN=...` — override push token
- `ALBION_INSTALL=C:\Program Files (x86)\AlbionOnline` — override Albion install root
- `ALBION_AGENT_OPEN_BROWSER=1` — open the meter URL in the default browser on launch (also via `--open-browser` flag)

### agent.json — push config (gitignored)

```json
{
  "pushUrl":   "wss://albion-meter.goldpipe.workers.dev/ingest",
  "pushToken": "<32 hex chars>",
  "albionInstallRoot": "C:\\Program Files (x86)\\AlbionOnline",
  "alwaysIncludeNames": ["FriendName1", "FriendName2"]
}
```

Missing `agent.json` triggers the first-run wizard (auto-generates a token, opens the browser to the magic-pair URL, writes the file).

`alwaysIncludeNames` (optional, case-sensitive): force these player names into the meter + loot views even when they're not in your guild and `PartyJoined` never fired — for non-guild friends you party with regularly. See the party-detection notes below.

### What the agent handles

| Event / op | Actual code | What we extract |
|---|---|---|
| `HealthUpdate` | 6 | **singular** hit (mostly auto-attacks): affectedId, healthChange, causerId — damage/heal/taken by ObjectId. Shares `applyHealthChange` with the batched variant. |
| `HealthUpdates` | 7 | **batched** hits (most ability / multi-source damage) as parallel arrays — was unhandled, which under-counted DPS to near-auto-attack-only. Decoded per-entry through the same path. |
| `NewCharacter` | 29 | objectId + name + guid + guild + equipment array (param 40), qualities (41), spells (43) |
| `Join` response | op 2 | objectId + guid + name + guild + zone (param 8) — authoritative source for the local player (only fires on zone change) |
| `JoinFinished` | 2 | zone-change marker |
| `CharacterEquipmentChanged` | 90 | objectId + equipment array (param 2), qualities (3), **active spells (param 7)** — arrives BEFORE Join response for the local player; cached in `pendingEquip` and replayed on Upsert |
| `ChangeEquipment` | 5 | older variant of the above; same handler |
| `MountStart` | 212 | rider objectId — stashed for next NewMountObject |
| `NewMountObject` | 310 | rider guid — paired with stashed objectId within a 2 s window |
| `PartyJoined` | 231 | guids byte-array + names array — full member list; persisted to `party.json` |
| `PartyPlayerJoined` | 233 | guid + name; persisted |
| `PartyPlayerLeft` | 235 | guid leaving; persisted |
| `PartyDisbanded` | 237 | resets party flags + clears `party.json` |
| `Died` | (see events.go) | name + killer for activity log + per-entity death count |
| `CastFinished` | (see events.go) | per-spell cast count + recent-cast ring buffer for debuff-window attribution |
| `ActiveSpellEffectsUpdate` | (see events.go) | active buffs/debuffs — diffed to open/close debuff windows |
| `OtherGrabbedLoot` | 285 | looter name + corpse source + item index + qty + isSilver → loot logger. Source key prettified (`@MOB_T6_HARVESTER_PLAYERSPAWN` → "T6 Harvester"). Does NOT credit silver (TakeSilver does). |
| `UpdateFame` | 91 | FameWithZoneMultiplier (param 2) + PremiumFame + SatchelFame + BonusFactor → per-event TotalGainedFame (SAT formula) accumulated into `session.fameTotal` |
| `TakeSilver` | 70 | YieldPreTax (FixPoint) - GuildTax → credited to `session.silverTotal` when looter (param 0) is the local player. Drop-source (param 2) seen via NewSilverObject → tagged into `session.mobSilverTotal`. |
| `UpdateMoney` | 89 | NOT a silver source (whole wallet, incl. deposits). Param 0 = local player's ObjectId → `localObjIdHint`, lets TakeSilver attribute loot when no Join fired this session. |
| `NewSilverObject` | 52 | silver-object id → `silverDrops` set; a TakeSilver from one of these is a ground/mob drop (mob-silver breakout). |
| `UpdateReSpecPoints` | 92 | "Combat Fame Credits" — per-event gained OR lifetime baseline-delta; UI labels this as "Combat Fame" |
| `MightAndFavorReceivedEvent` | 470 | Might gained (FixPoint) — `session.mightTotal` |

**Critical fixes worth highlighting**:
- **Batched HealthUpdates (code 7)** carries the bulk of ability damage; missing its handler under-counted DPS to near-auto-attack-only totals. Both the singular and batched handlers fold through one `applyHealthChange`.
- **Silver** is credited from `TakeSilver` only (never `UpdateMoney`, which is the whole wallet incl. deposits/sales; and never OtherGrabbedLoot, which fires for the same pickup → would double-count). The local player is identified by `UpdateMoney`'s param 0 hint, so silver counts even when the agent starts mid-zone with no Join. `session.mobSilverTotal` breaks out ground-drop silver.
- **Fame** uses SAT's `TotalGainedFame = (FameWithZoneMultiplier + Premium + Satchel) × Bonus`.
- **Fight scope**: `touchCombat` (fight counter + Current-bucket reset) only fires when the **local player** is the causer/affected (`localInvolved`), so distant mob-on-mob fights don't tick the counter while idle.
- **LastFight carryover**: `Current.Or(LastFight)` in the snapshot shows the previous fight's numbers in the gap after a new fight starts but before damage lands, so rows don't snap to zero between fights.

**Not handled** (deliberately out of scope): trade, market, guild events, mail, harvest, chat. Loot tracking IS in (F6 + F7).

### Party detection, meter scope & commands

`PartyJoined` only fires the instant a member joins — never retroactively, and (verified against SAT + albion-online-stats) **no Photon packet re-broadcasts a static roster** mid-session; a passive sniffer can only ever hear the join-moment event. So the clean workflow is **launch the agent first, then group up** — it auto-pulls the whole party. If the agent starts (or is rebuilt + relaunched) while already grouped, the roster is otherwise lost. Mitigations, in order:

1. **Guild scope (best for guild groups)** — see below; shows every same-guild combatant with no roster at all. For an all-guild party this is the answer, not party tracking.
2. **Persistence** — confirmed party members are written to `%LocalAppData%\GDA\party.json` on every party event + manual edit; `RestoreParty()` rehydrates them on boot (30-min freshness gate). A background ticker (`RepersistPartyIfActive`, main.go) **re-saves the roster every 1 min while grouped** so a stable party — which emits no join/leave events to refresh the file — doesn't age out and survives a restart.
3. **Manual add/remove + the player picker** — the snapshot ships `VisiblePlayers` (tracked players not in the party) **and** `Roster` (every named player seen, with `isInParty` + `sameGuild`, guildmates sorted to top). The **Track Players** picker under the Meter (and the Party tab's "Add players in range") render these: click to toggle tracking, an **"Add all guild"** one-tap bulk-add, search, and a re-zone hint when the local guild isn't known yet. Each party row has Remove; **Clear party** wipes the roster + `party.json`.
4. **`alwaysIncludeNames`** allowlist in agent.json — non-guild friends always show (additive; surfaced via `AlwaysIncludes` in `scopedMembers`).

**Meter scope** (Settings → Visibility, and the title-bar `Party / Guild / All` switch) controls who appears in the meter + loot + archived fights, via the `setLootFilter` command → `Engine.LootFilterMode()`:
- `party` — confirmed party + local + allowlist only (tight dungeons)
- `partyGuild` (default) — the **UNION** of party + local + allowlist + every same-guild player with combat activity. Not "trust the party roster exclusively once it has >1 member" (that old behaviour let a partial/stale roster *hide* fighting guildmates) — the union means a 20-person guild group appears the instant members deal/take damage, no roster needed. Requires the local guild to be known (set on zone change), so re-zone once after a restart.
- `everyone` — every entity with combat activity (ZvZ)

`scopedMembers()` is the single filter shared by the live snapshot **and** `archiveCurrentFight`, so a past fight shows exactly who the live meter showed at the time.

**Web → agent commands** (over the same WebSocket, forwarded by the Worker; `Engine.HandleCommand`): `resetSession`, `deleteSession <id>`, `setLootFilter <party|partyGuild|everyone>`, `addPartyMember <guid>`, `removePartyMember <guid>`, `clearParty`.

### Capture self-heal

The Windows raw socket binds to the interfaces present at startup and blocks in `Recvfrom`. If the active adapter changes (VPN/Wi-Fi) or the PC sleeps, packets silently stop and never resume. `superviseCapture` (main.go) watchdogs `packetsSeen`: after traffic has flowed, ~45 s of silence cancels + reopens the socket (re-enumerating interfaces). A first run that errors before any packet (no admin / no interface) is still fatal, not retried. The console heartbeat (`packets=N` every 5 s) is the at-a-glance liveness signal.

### Game-data loaders

- **items.bin** — 12,060 entries. 1-based index, includes enchantment variants (`@1`/`@2`/`@3`) and `_EMPTY`/`_FULL` journal pairs. See `gamedata/items.go` for the algorithm (mirrors SAT's `ItemData.cs`).
- **spells.bin** — 8,856 entries. Index → uniquename, used by `BySpell` rendering.
- **localization.bin** — 38,174 EN-US strings. TMX format. Used to translate spell/item uniquenames to in-game display names ("Adept's Arclight Blasters" instead of `T4_2H_DUALCROSSBOW_CRYSTAL`).

All three are DES-CBC + gzip + XML. Decryption code: `gamedata/decrypt.go`.

### Diagnostic probe

`agent/cmd/probe/main.go` is a tiny CLI that loads the three game-data files and prints counts + sample lookups. No admin required, no Photon capture. Useful for verifying the path / catalog correctness without launching the full agent.

```powershell
$env:ALBION_INSTALL = "C:\Program Files (x86)\AlbionOnline"
go run ./cmd/probe
```

---

## Cloudflare Worker (`go-port` branch, `worker/`)

TypeScript Worker + Durable Object backend. Deployed at `https://albion-meter.goldpipe.workers.dev`.

### Routes

| Path | Protocol | Who |
|---|---|---|
| `/ingest?token=…` | WebSocket | the Go agent — pushes snapshots |
| `/view?token=…` | WebSocket | browser viewers — receives every snapshot the agent sends, plus an immediate replay on connect |
| `/healthz` | HTTP | liveness |
| `/` | HTTP | tiny landing page |

Auth: SHA-256 hash of the bearer token names a Durable Object room. No DB, no signup. Agent and browser sharing a token meet in the same room.

`MeterRoom` uses the WebSocket Hibernation API. Latest snapshot is kept in memory; agents republish on activity (≤400 ms) so hibernation losing state is harmless. The room also **forwards viewer→agent command messages** (New Session, delete session, set scope, party add/remove/clear) transparently to the ingest socket — see `Engine.HandleCommand`. WebSocket *messages* over an open socket don't count as Cloudflare requests; only the connection upgrade does — so the daily-request budget is driven by snapshot/DO message volume, hence the cadence throttle on the agent.

### Build & deploy

```powershell
cd worker
npx wrangler login        # one-time
npx wrangler deploy
```

---

## React web UI (`go-port` branch, `web/`)

Vite + React 19 + Tailwind v4. Deployed at <https://albion-meter-web.pages.dev>.

### Build & deploy

```powershell
cd web
npm run build
npx wrangler pages deploy ./dist --project-name albion-meter-web --commit-dirty=true
```

### Local dev / demo

```powershell
cd web
npm run dev
# Open http://localhost:5173/?demo=1 to see the meter rendered against
# hand-crafted demo data (no agent required). Drops the ?demo=1 to use
# the real WebSocket path.
```

### Important caveats

- Tailwind v4: no `postcss.config` or `tailwind.config` needed — the `@tailwindcss/vite` plugin handles everything.
- Settings are stored in localStorage under key `gda:settings:v2` (bump the suffix to force-reset for returning users when design tokens change). `panes` is now `Record<SubMetric, boolean>` (six keys: `damageCurrent/damageTotal/healCurrent/healTotal/takenCurrent/takenTotal`); the merge falls returning users back to `damageCurrent: true`. `meterScope` (`party|partyGuild|everyone`) also lives here and is mirrored to the agent via `setLootFilter` on connect.
- Per-deployment hash URLs (`<hash>.albion-meter-web.pages.dev`) are separate browser origins and don't share the user's pairing token — always link to the production alias.
- Use `Restart GDA Agent.cmd` after agent changes; web changes only need a hard refresh (Empty Cache and Hard Reload — plain Ctrl+Shift+R hasn't reliably busted the cache).

---

## Current state (working / known gaps)

### Navigation (web)
- **Top tab bar** — `Meter · Loot · Party · Sessions`, each a real route (`/`, `/loot`, `/party`, `/sessions`). Soft-navigated via `history.pushState` + `popstate` so a single `LiveApp`/`DemoApp` shell stays mounted (no reload, no scroll jump, one WebSocket across tabs). The old footer-corner buttons are gone; legacy `*Panel` modals remain in the tree but unmounted.
- **Hotkeys** — `M/L/P/S` cycle tabs, `D/H/T` toggle the Current pane of each metric, `Esc` closes drill-in/Settings.
- `web/public/_redirects` (`/* → /index.html 200`) makes Pages serve the SPA on every route.

### Meter (web)
- **C3 editorial Fame hero** (`SessionStrip`) — Fame as a giant `clamp(64–128px)` number with glyph + per-hour pace + a wide **smoothed pace sparkline** (EMA of per-second gain, not raw deltas), Silver / Combat Fame / Might stacked beside it, "Carried by `<top dealer>`" credit (Fraunces italic). Currencies floored (matches the loot panel exactly).
- **Per-scope panes** — `PaneSet` is `Record<SubMetric, boolean>`; each pane is one metric+scope, so the SAME metric can open twice side-by-side (`Damage · Current` next to `Damage · Session`). Rows show one number + share% + DPS/HPS (no dual `↳ session`). Header has `Cur/Ses` pill pairs per metric; shared `MeterPanes` renders the grid.
- **Stable ordering** — `rankBy(selector)` (one shared comparator) ranks by metric desc with a `userGuid` tiebreak; the agent also sorts `out.Players` by guid. Kills the "Top"/"Carried by" name flicker on ties.
- **IPChip** — averaged IP across core slots + per-slot hover tooltip. Note: **base IP only** — Albion no longer ships item quality on the wire, so the quality multiplier can't be applied.
- Per-class accent colours; live equipment/spell swaps reflect within ~400 ms.
- **Track Players picker** (`PlayerPicker`, button under the Meter) — every player in range, guildmates sorted to top, click to toggle tracking, search + "Add all guild" bulk-add. Reads the snapshot's `Roster` field.
- Drill-in tabs: Fight / Session / Targets / Assists (Level-2 debuff-window attribution); mob targets resolve to names via mobs.bin.
- Fight history archive (last 20 fights, agent RAM); scope-aware (a dungeon run archives only the party).

### Loot (web — `/loot`)
- Per-player rollup: top-farmer card, IPChip rows, gold value bars, activity dot, `PARTY/GUILD/FRIEND` source badges, sticky column headers. Subline falls back: priced top item → "silver only · X" → "recent: X (unpriced)".
- Silver QTY shows `—` for piles (not raw FixPoint); local row's silver is the authoritative `session.SilverTotal` with a `· mob:` ground-drop breakout.
- `Copy Summary` / `Copy as table` for Discord.

### Economy & data
- **Silver** counts mob/world drops (via TakeSilver + UpdateMoney local-id hint), not just chests; `mobSilverTotal` breakout. No double-count, deposits/sales ignored.
- **Combat Fame Credits** correctly labeled (was "Respec").
- Session persistence (`%LocalAppData%\GDA\sessions`), zone history, dungeon scope, AODP loot pricing — all as before.
- Game-data loaders: items.bin (12,060), spells.bin (~9,166), mobs.bin (4,595, `index-15` shift), localization.bin (38,174 EN-US).

### Robustness
- **Capture self-heal** — watchdog reopens the raw socket on a packet stall (adapter change / sleep).
- **Party persistence + 1-min re-persist while grouped + manual add/remove/clear + picker** — survives agent restart; the guild-union scope covers guild groups with no roster at all.
- **Snapshot deadlock fixed** — `Snapshot()` holds `store.mu.RLock` and used to call the locking `AllPlayers()` again (recursive RLock). Go's `RWMutex` forbids recursive read-locking, so in a busy zone (16-player dungeon) the second RLock blocked behind a pending writer and froze the whole agent — push stalled *and* the renderLoop heartbeat stopped (both call `Snapshot()`). Now uses `allPlayersLocked()` (no re-lock). Watch for this pattern: never call a `store.mu`-locking method while already holding the lock — use the `…Locked` variant.
- **Request-budget cadence** — 400 ms active / 15 s idle heartbeat keeps Cloudflare's free-tier 100k/day request limit comfortable.

⚠️ **Known gaps**:
- **Item quality not on the wire** — `parseEquipmentParams` qualities array is all-zero on the current patch, so IP is base only and there's no quality-tinted gear display.
- **Mid-session party capture is impossible passively** — Albion only sends the roster at the join moment (no re-broadcast packet exists; confirmed against SAT + albion-online-stats). Mitigated by: launch-before-grouping, guild scope (no roster needed), the 1-min persistence, and manual/"Add all guild" picking. There's no magic "scan my party" button — the tool can only listen, never request.
- **Some passive sub-effects** fall through to the prettified uniquename (compound-word + city-name splitter added; main abilities resolve correctly).
- **Crit %** — Albion doesn't expose a crit flag.
- **Mechanics pane** — removed from the tab cluster (was a placeholder).
- **Level-3 debuff attribution** — Assists show "damage during your debuff window," not a hard multiplier.

---

## User preferences observed

- Iterates fast, screenshots-driven. Prefers concise responses.
- **Not a coder** — give clear, numbered, copy-paste step-by-step guides; no jargon. Say which launcher to double-click, not which commands to type.
- After an **agent** change, tell them to double-click `Restart GDA Agent.cmd` (Verbose variant only when a capture is needed) — never hand over the manual `go build` + relaunch. After a **web-only** change, say "hard-refresh." Be explicit which kind a fix was.
- Often plans via `/ultraplan` (a remote planning session that returns a plan for approval); when one is pending, point them at the URL to review, then implement once approved.
- Keeps production SAT running in the background during dev; the dev build is separate and only for testing changes.
- Wants WoW-style damage-meter UX (Skada / Recount / Details! conventions).
- Pivoted from C# to Go mid-session, motivated by wanting a "single binary that pushes to a website" architecture.
- Comfortable in admin PowerShell; uses `winget` for tool installs.
- For Windows-specific ops (file moves, shortcuts, process management) prefer the PowerShell tool. Bash is available via Git Bash for POSIX-style scripts.
