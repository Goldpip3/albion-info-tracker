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
src/                   C# SAT solution (sat-fork stack; preserved on go-port too)
agent/                 Go agent — capture + parse + state + WS push
  cmd/agent/           entrypoint
  cmd/probe/           one-shot diagnostic — loads items/spells/localization without UAC
  internal/capture/    Windows raw-socket Photon UDP capture
  internal/photon/     Protocol18 deserializer + outer parser + fragment reassembly
  internal/gamecodes/  generated EventCodes/OperationCodes enums (mirrored from SAT)
  internal/gamedata/   DES-CBC + gzip + XML decoders for *.bin game files
  internal/domain/     entity store, combat tracker, Photon event handlers, session economy
  internal/push/       WebSocket push client (coder/websocket, reconnect+backoff)
  internal/config/     agent.json + env-var loader
worker/                Cloudflare Worker + Durable Object backend (TS)
web/                   React + Vite + Tailwind v4 frontend
  src/                 components, hooks, types, demo data
  public/assets/       GDA brand mark (16/32/48/64/128/256/512 PNG + SVG)
GDA Launcher.lnk       double-click to launch the agent with --open-browser
ARCHITECTURE.md        deep-dive into protocol, indexing, race fixes, data flow
```

## Git

- Remote: `https://github.com/Goldpip3/albion-info-tracker` (carries the previous project's name).
- Active branch: **`go-port`** (started from `sat-fork`, no merges back).
- `origin/main` and `origin/webify` preserve the much-older `albion-info-tracker` project. Replaced with SAT source in commit `0e540a6`.
- **Before pushing `go-port` to a public mirror, rotate the `pushToken`** — an early commit briefly tracked `agent.json`. Token is currently `cd80e1d86f30432a35bd17840cf763fd`.

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

Single static Windows binary. Captures Albion UDP traffic via raw sockets (`SIO_RCVALL`, requires admin), parses Photon protocol, maintains entity/combat state, pushes JSON snapshots to a Cloudflare Worker every **250 ms**.

### Build & run

```powershell
cd agent
go build -o agent.exe ./cmd/agent
# Admin PowerShell — SIO_RCVALL requires elevation.
.\agent.exe
```

Diagnostic env vars:
- `ALBION_AGENT_VERBOSE=1` — per-event logging (useful for diagnosing classification + party detection)
- `ALBION_AGENT_SHOW_ALL=1` — drop the party-membership gate in the snapshot (show every entity with combat activity)
- `ALBION_AGENT_URL=wss://...` — override push URL
- `ALBION_AGENT_TOKEN=...` — override push token
- `ALBION_INSTALL=C:\Program Files (x86)\AlbionOnline` — override Albion install root
- `ALBION_AGENT_OPEN_BROWSER=1` — open the meter URL in the default browser on launch (also via `--open-browser` flag)

### agent.json — push config (gitignored)

```json
{
  "pushUrl":   "wss://albion-meter.goldpipe.workers.dev/ingest",
  "pushToken": "<32 hex chars>",
  "albionInstallRoot": "C:\\Program Files (x86)\\AlbionOnline"
}
```

Missing `agent.json` triggers the first-run wizard (auto-generates a token, opens the browser to the magic-pair URL, writes the file).

### What the agent handles

| Event / op | Actual code | What we extract |
|---|---|---|
| `HealthUpdate` | 6 | affectedId, healthChange, causerId — damage/heal/taken attribution by ObjectId |
| `NewCharacter` | 29 | objectId + name + guid + guild + equipment array (param 40, slot 0 = MainHand) — primary identity source for non-local players |
| `Join` response | op 2 | objectId + guid + name + guild + zone (param 8) — **only** authoritative source for the local player |
| `JoinFinished` | 2 | zone-change marker |
| `CharacterEquipmentChanged` | 90 | objectId + equipment array (param 2) — **arrives BEFORE Join response for the local player**, so cached in `pendingEquip` and replayed |
| `ChangeEquipment` | 5 | older variant of the above; same handler |
| `MountStart` | 212 | rider objectId — stashed for next NewMountObject |
| `NewMountObject` | 310 | rider guid — paired with stashed objectId within a 2 s window |
| `PartyJoined` | 231 | guids byte-array + names array — full member list |
| `PartyPlayerJoined` | 233 | guid + name |
| `PartyPlayerLeft` | 235 | guid leaving |
| `PartyDisbanded` | 237 | resets party flags |
| `Died` | (see events.go) | name + killer for activity log + per-entity death count |
| `CastFinished` | (see events.go) | per-spell cast count (BySpell + BySpellSession), recent-cast ring buffer for debuff-window attribution |
| `ActiveSpellEffectsUpdate` | (see events.go) | active buffs/debuffs on target — diffed against prior set to open/close debuff windows |
| `UpdateFame` | 91 | TotalPlayerFame (FixPoint) — delta-tracked into session.fameTotal |
| `UpdateMoney` | 89 | CurrentPlayerSilver (FixPoint) — delta-tracked into session.silverTotal |
| `UpdateReSpecPoints` | 92 | per-event gained credits OR lifetime total (array form, param 0 element 1) — chosen via baseline-delta logic |
| `MightAndFavorReceivedEvent` | 470 | Might gained (FixPoint) — session.mightTotal |

**Not handled** (deliberately out of scope): dungeon tracker, loot, trade, market, guild, mail, harvest, chat.

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

`MeterRoom` uses the WebSocket Hibernation API. Latest snapshot is kept in memory; agents republish every 250 ms so hibernation losing state is harmless.

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
- Settings are stored in localStorage under key `gda:settings:v2` (bump the suffix to force-reset for returning users when design tokens change).
- Per-deployment hash URLs (`<hash>.albion-meter-web.pages.dev`) are separate browser origins and don't share the user's pairing token — always link to the production alias.

---

## Current state (working / known gaps)

✅ **Confirmed working end-to-end**:
- Agent captures + parses Photon UDP
- Local player class detection via `CharacterEquipmentChanged` + pre-Join equipment cache
- Items.bin loader produces 12,060 entries with enchantment expansion
- Localization translates spell/item uniquenames to tooltip names ("Adept's Arclight Blasters", "Chain Slash")
- Per-class accent colours: Daggers red, Fire amber, Frost cyan, Hammer steel, Arcane violet, Holy gold, Nature green
- Multi-pane mode (Damage / Healing / Tank side-by-side) with compact 4-column layout
- Stat cards (Fame / Silver / Respec / Might) with sparkline pulse, accent rails
- Drill-in with four tabs: Fight / Session / Targets / Assists
- Debuff-window attribution ("Level 2" assist tracking)
- Session-level cast counts
- Per-target damage breakdown
- Fight history archive (last 20 fights, kept in agent RAM)
- 250 ms snapshot push rate

⚠️ **Known gaps**:
- **Mid-zone party detection** — same fundamental limitation as SAT. `PartyJoined` doesn't fire retroactively. Mitigations: mount-event binding (any party member fresh-mounting binds them), re-zone, `ALBION_AGENT_SHOW_ALL=1`.
- **Some spells return blank tooltip name** — passive sub-effects like `SKILLSHOT_TELEPORT_BUFF` aren't in localization.bin; fall through to the prettified uniquename. Acceptable; the relevant abilities ("Flickershot", "Chain Slash") do resolve correctly.
- **Crit % detection** — Albion doesn't expose a crit flag in HealthUpdate; SAT doesn't track this either. Would require server-side damage formula reproduction.
- **Mechanics pane content** — placeholder only.

---

## User preferences observed

- Iterates fast, screenshots-driven. Prefers concise responses.
- Keeps production SAT running in the background during dev; the dev build is separate and only for testing changes.
- Wants WoW-style damage-meter UX (Skada / Recount / Details! conventions).
- Pivoted from C# to Go mid-session, motivated by wanting a "single binary that pushes to a website" architecture.
- Comfortable in admin PowerShell; uses `winget` for tool installs.
- For Windows-specific ops (file moves, shortcuts, process management) prefer the PowerShell tool. Bash is available via Git Bash for POSIX-style scripts.
