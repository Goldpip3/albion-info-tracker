# CLAUDE.md

Personal fork of [Triky313/AlbionOnline-StatisticsAnalysis](https://github.com/Triky313/AlbionOnline-StatisticsAnalysis) (SAT) — a third-party Albion Online stats / damage-meter app. The repo now contains **two parallel implementations on different branches**:

- **`sat-fork`** — the original C# / WPF / .NET 10 SAT fork. Production-quality, used for actual gameplay. Two WIP commits on top of upstream (current+overall meter + mount-event ObjectId binding); both compile clean but were **never verified in-game** before the user pivoted to Go.
- **`go-port`** — clean-room Go rewrite that pushes JSON snapshots to a Cloudflare Worker and renders the meter in a React web UI hosted on Cloudflare Pages. **Currently the active development branch and confirmed working end-to-end** (local player's damage attributes correctly in the browser).

Nothing here is intended to go upstream.

## Launch Claude Code from this folder

CLAUDE.md auto-loads when Claude Code is run from the project root or a subdir. Run `claude` from `C:\Users\colom\Claude Projects\Albion DPS\`. If launched from `OneDrive\Documents\Albion Analysis\` (the production install), this file won't be loaded.

## Layout

Project root: `C:\Users\colom\Claude Projects\Albion DPS\`

```
src/        C# SAT solution (sat-fork stack; preserved on go-port too)
agent/      Go agent — capture + parse + state + WS push (go-port)
worker/     Cloudflare Worker + Durable Object backend (go-port)
web/        React + Vite + Tailwind v4 frontend (go-port)
SAT (Dev Build).lnk   double-click to launch the C# dev build
```

## Git

- Remote: `https://github.com/Goldpip3/albion-info-tracker` (carries the previous project's name).
- Active branch: **`go-port`** (started from `sat-fork`, no merges back).
- `origin/main` and `origin/webify` preserve the much-older `albion-info-tracker` project (a headless C# WebSocket DPS meter forked from `akashi-sym/Minimal-Albion-Online-DPS-Meter`). Replaced with SAT source in commit `0e540a6`.
- Neither branch is pushed to a remote yet. **Before pushing `go-port`, rotate the `pushToken` in `agent/agent.json`** — an early commit accidentally tracked it and the file was untracked in a follow-up, but the secret is still in history.

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

1. **Current+overall damage meter columns** (WoW Skada style). Each row shows current-fight | session total side by side. Touches:
   - `DamageMeter\CombatEvent.cs` — `GetEffectiveDuration()` (floor 1s for DPS sanity)
   - `DamageMeter\CombatEventTracker.cs` — `GetActiveOrLastCompletedEventSnapshot()`
   - `DamageMeter\DamageMeterFragment.cs` — `CurrentXxx` props + `XxxDualString` computed
   - `DamageMeter\DamageMeterSnapshotFragment.cs` — `XxxDualString` returns overall-only
   - `Network\Manager\CombatController.cs` — `ApplyCurrentFightStats()` per refresh tick
   - `Styles\DamageMeterStyles.xaml`, `UserControls\DamageMeterControl.xaml`, `Views\DamageMeterWindow.xaml`

2. **Mount-event ObjectId↔UserGuid binding** to mitigate the mid-zone-start bug. New `MountStartEvent`/`NewMountObjectEvent` plus handlers, plus `EntityController.StashMountStartObjectId` + `BindGuidToRecentMountStart` with a 2-second transient window. Also a name-preservation fix in `EntityController.AddEntity` (was overwriting names with empty values during entity merges).

### Settings storage

`%LocalAppData%\StatisticsAnalysisTool\Instances\<id>\Settings.json`. Debug builds use the literal `Debug` instance folder; Release/path-based builds use an 8-char hash of the exe path (e.g. `208AC6C2`). Settings persist across rebuilds. The user's Albion install is at `C:\Program Files (x86)\AlbionOnline` (one word — not the default `Albion Online`).

### Code conventions

- The damage meter row template in `Styles\DamageMeterStyles.xaml` has **two near-duplicate DataTemplates**: `DamageMeterFragmentTemplate` (live, `DamageMeterFragment`) and `DamageMeterSnapshotFragmentTemplate` (historical, `DamageMeterSnapshotFragment`). When changing bindings, both need parallel updates or only one template should be touched.
- Indentation differs between the live and snapshot templates inside the TakenDamage block — `Edit replace_all=true` can be safe for the Damage/Dps and Heal/Hps blocks (identical text in both), but the TakenDamage block requires a precise single-target edit.
- WPF settings: extend `SettingsObject.cs`, access via `SettingsController.CurrentSettings.<field>`.

---

## Go agent (`go-port` branch, `agent/`)

Single static Windows binary. Captures Albion UDP traffic via raw sockets (`SIO_RCVALL`, requires admin), parses Photon protocol, maintains entity/combat state, pushes JSON snapshots to a Cloudflare Worker every 500ms.

### Layout

```
agent/
├── cmd/agent/main.go        entry — capture + parse + render heartbeat + WS push
├── cmd/extract/main.go      one-shot CLI to decrypt items/spells/mobs.bin
├── internal/capture/        raw-socket Photon UDP capture (Windows)
├── internal/photon/         Protocol18 deserializer + outer parser + fragment reassembly
├── internal/gamecodes/      generated EventCodes/OperationCodes enums (683 + 540 entries)
├── internal/gamedata/       DES-CBC + gzip + XML decode of game data .bin files
├── internal/domain/         entity store + combat tracker + Photon event handlers
├── internal/push/           WebSocket push client (coder/websocket, reconnect+backoff)
└── internal/config/         agent.json + env var loader
```

### Build & run

```powershell
cd agent
go build -o agent.exe ./cmd/agent
# Admin PowerShell — SIO_RCVALL requires elevation.
.\agent.exe
```

Diagnostic env vars: `ALBION_AGENT_VERBOSE=1` (per-event logging), `ALBION_AGENT_SHOW_ALL=1` (drop the party-membership gate in the snapshot).

### agent.json — push config (gitignored)

```json
{
  "pushUrl":   "wss://albion-meter.goldpipe.workers.dev/ingest",
  "pushToken": "<32 hex chars>"
}
```

Env overrides: `ALBION_AGENT_URL`, `ALBION_AGENT_TOKEN`, `ALBION_INSTALL`. Missing `agent.json` keeps the agent in stdout-only mode.

### What's handled

| Event / op | Code | What we extract |
|---|---|---|
| `HealthUpdate` | 6 | affectedId, healthChange, causerId — damage/heal/taken attribution by ObjectId |
| `NewCharacter` | 29 | objectId + name + guid + guild — primary identity source for non-local players |
| `Join` response | op 2 | objectId + guid + name + guild — **only** authoritative source for the local player |
| `JoinFinished` | 2 | zone-change marker (params are empty in practice) |
| `MountStart` | 212 | rider objectId — stashed for next NewMountObject |
| `NewMountObject` | 310 | rider guid — paired with stashed objectId within a 2 s window |
| `PartyJoined` | 231 | guids byte-array + names array — full member list |
| `PartyPlayerJoined` | 233 | guid + name (no ObjectId — comes later from NewCharacter/Mount) |
| `PartyPlayerLeft` | 235 | guid leaving |
| `PartyDisbanded` | 237 | resets party flags |

Not handled: dungeon tracker, loot, trade, market, guild, mail, harvest, chat. Damage-meter MVP only.

### Photon protocol quirks (rediscovered during port)

- Outer envelope is big-endian; Protocol18 inner is little-endian.
- A "signature" byte (typically `0xF3`) sits between the command header and the message-type byte in `SendReliable` payloads — easy to miss when porting.
- Application event/operation codes >255 live in parameter `252` (events) or `253` (operations). The EventData.Code byte is just a transport-level marker, often `1`. Always prefer the parameter value when present — SAT's `DebugConsole.GetIdFromParams` documents this.
- Custom types (Protocol18 type 19) carry a 1-byte typeCode + length-prefixed bytes. GUIDs are 16-byte custom types; `.NET`'s `Guid(byte[])` order is what's on the wire (first 4 bytes byte-swapped vs textual representation).
- Albion **never broadcasts the local player via NewCharacter** — local identity comes only from the `Join` operation response. Easy gotcha when porting.

---

## Cloudflare Worker (`go-port` branch, `worker/`)

TypeScript Worker + Durable Object backend. Deployed at `https://albion-meter.goldpipe.workers.dev`.

### Routes

| Path | Protocol | Who |
|---|---|---|
| `/ingest?token=…` | WebSocket | the Go agent — pushes snapshots |
| `/view?token=…` | WebSocket | browser viewers — receives every snapshot the agent sends, plus an immediate replay on connect |
| `/healthz` | HTTP | liveness — returns `ok` |
| `/` | HTTP | tiny landing page |

Auth: SHA-256 hash of the bearer token names a Durable Object room. No DB, no signup. Agent and browser sharing a token meet in the same room.

`MeterRoom` uses the WebSocket Hibernation API (`ctx.acceptWebSocket`, `ctx.getWebSockets` by tag) so the DO holds idle browser connections without burning CPU. Latest snapshot is kept in memory only; agents republish every 500 ms so hibernation losing state is harmless.

### Build & deploy

```powershell
cd worker
npx wrangler login        # one-time
npx wrangler deploy
```

Wrangler config: `wrangler.toml` with `nodejs_compat` and a SQLite-backed Durable Object class. Free tier (~1 M DO requests/month) easily covers a solo user.

---

## React web UI (`go-port` branch, `web/`)

Vite + React 19 + Tailwind v4. Deployed at `https://albion-meter-web.pages.dev` (Cloudflare Pages project `albion-meter-web`).

Setup screen prompts for Worker URL + token, persisted in `localStorage`. Three view modes (Damage / Heal / Taken). Each row shows CURRENT | OVERALL | per-second with a bar-fill background proportional to the top performer. ★ marks the local player.

Build & deploy:

```powershell
cd web
npm run build
npx wrangler pages deploy ./dist --project-name albion-meter-web
```

Tailwind v4: no `postcss.config` or `tailwind.config` needed — the `@tailwindcss/vite` plugin handles everything. `index.css` just `@import "tailwindcss";`.

---

## Working / not working

✅ **Confirmed working end-to-end**: agent captures Albion packets, parses them, identifies the local player from the Join response, pushes snapshots to the Worker, browser renders the live meter with the user's row populating as damage is dealt.

⚠️ **Open issues**:

- **Mid-zone party detection** — same fundamental gap as SAT. When the agent starts mid-zone with a party already formed, neither `PartyJoined` nor `PartyPlayerJoined` fires retroactively. Mount-event binding helps if anyone fresh-mounts during the session; otherwise the user must re-zone, or rely on `ALBION_AGENT_SHOW_ALL=1` to see all entities with combat activity.
- **No spell-level breakdown yet** — every `HealthUpdate` carries `CausingSpellIndex` and a full `SpellCatalog` is loaded from `spells.bin` (8856 spells), but the engine sums all damage per player rather than per spell. Wiring per-spell sub-rows in the React UI is the natural next feature.
- **C# `sat-fork` work never verified in-game** — both the current+overall meter and the mount-event fix compile clean but were superseded by the Go pivot before testing.
- **`origin` has no `go-port`** — branch is local-only. If pushed, rotate the agent.json token first.

---

## User preferences observed

- Iterates fast, screenshots-driven. Prefers concise responses.
- Keeps production SAT running in the background during dev; the dev build is separate and only for testing changes.
- Wants WoW-style damage-meter UX (Skada / Recount / Details! conventions).
- Pivoted from C# to Go mid-session, motivated by wanting a "single binary that pushes to a website" architecture. Original C# SAT preserved on `sat-fork` for actual gameplay.
- Comfortable in admin PowerShell; uses `winget` for tool installs (Go SDK, .NET SDK).
- For Windows-specific ops (file moves, shortcuts, process management) prefer the PowerShell tool. Bash is available for POSIX-style scripts and works via Git Bash.
