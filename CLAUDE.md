# CLAUDE.md

Personal fork of [Triky313/AlbionOnline-StatisticsAnalysis](https://github.com/Triky313/AlbionOnline-StatisticsAnalysis) (SAT) — a third-party Albion Online stats / damage-meter WPF app. The user is iterating on improvements for their own use; nothing here is intended to go upstream.

## Launch Claude Code from this folder

CLAUDE.md only auto-loads when Claude Code is run from the project root or a subdir of it. Run `claude` from `C:\Users\colom\Claude Projects\Albion DPS\`. If launched from `OneDrive\Documents\Albion Analysis\` (the production install), this file won't be loaded.

## Layout

- **Project root** (git repo): `C:\Users\colom\Claude Projects\Albion DPS\`
- **Source**: `src\StatisticsAnalysisTool\`
- **Solution**: `src\StatisticsAnalysisTool.sln`
- **Built exe** (Debug): `src\StatisticsAnalysisTool\bin\Debug\net10.0-windows\StatisticsAnalysisTool.exe`
- **Launch shortcut** (double-click to run dev build): `SAT (Dev Build).lnk` at project root
- **Reference production install** (Triky313 v9.2.1, used by the user during actual gameplay): `C:\Users\colom\OneDrive\Documents\Albion Analysis\` — do not modify

## Build & run

- Requires **.NET 10 SDK** (10.0.300 installed via winget; `global.json` pins to 10.0.100 with `latestFeature` rollforward).
- `dotnet build src\StatisticsAnalysisTool.sln` from project root.
- The dev build always boots in Debug config — the "DEBUG" badge in the title bar is hardcoded via `#if DEBUG` in `ViewModels\MainWindowViewModel.cs:365-367`. Cannot be toggled at runtime; would require a Release build to suppress.
- Iteration loop: edit → `dotnet build` → close any running dev SAT (it holds the exe locked) → double-click `SAT (Dev Build).lnk`.
- Cannot verify WPF rendering from the agent side. Visual confirmation requires the user to relaunch and screenshot.

## Git

- Remote: `https://github.com/Goldpip3/albion-info-tracker` (still carries the previous project's name).
- Active branch: **`sat-fork`**.
- `origin/main` and `origin/webify` preserve the prior `albion-info-tracker` project (a headless WebSocket DPS meter forked from `akashi-sym/Minimal-Albion-Online-DPS-Meter`). The working tree was wiped and replaced with SAT source in commit `0e540a6` — old files recoverable via `git checkout <prior-commit>` on `main`/`webify`.

## Current work in progress

- **Damage meter "current + overall" feature** (WoW Skada-style): each player row shows current-fight value `|` session total, side by side, for damage/DPS/heal/HPS/taken-damage. Column header `CURRENT | OVERALL` sits above the data rows. Implemented across:
  - `DamageMeter\CombatEvent.cs` — `GetEffectiveDuration()` (floor 1s for DPS sanity)
  - `DamageMeter\CombatEventTracker.cs` — `GetActiveOrLastCompletedEventSnapshot()`
  - `DamageMeter\DamageMeterFragment.cs` — `CurrentDamage`/`CurrentDps`/etc. + `XxxDualString` computed props
  - `DamageMeter\DamageMeterSnapshotFragment.cs` — `XxxDualString` returns overall-only (snapshots have no live fight)
  - `Network\Manager\CombatController.cs` — `ApplyCurrentFightStats()` helper called per refresh tick
  - `Styles\DamageMeterStyles.xaml` — swap to DualString bindings, center-aligned, widened columns
  - `UserControls\DamageMeterControl.xaml`, `Views\DamageMeterWindow.xaml` — `CURRENT | OVERALL` header row
  - Status: builds clean, currently uncommitted on `sat-fork`.

- **Open bug — party members don't appear unless SAT was running at zone-in.** SAT only registers players via `NewCharacterEvent`, which Albion broadcasts once per zone entry. If SAT starts mid-zone, those packets are missed and party members never appear in the damage meter. The workaround is re-zoning. Root cause and fix paths are planned (`Path A`: find an existing packet that carries `(UserGuid, Name, ObjectId)` together; `Path B`: three-way stitch from `PartyJoinedEvent` + combat events). Next concrete step is a discovery-phase debug packet logger before any code changes.

## Settings storage

SAT writes to `%LocalAppData%\StatisticsAnalysisTool\Instances\<id>\Settings.json`. Debug builds use the literal `Debug` instance folder; Release/path-based builds use an 8-char hash of the exe path (e.g. `208AC6C2`). Settings persist across rebuilds. On-disk fields confirmed correct after the SAT import: `MainGameFolderPath`, `HasCompletedFirstStartGuide=True`, `IsNpcapInfoDialogShownOnStart=False`.

## Code conventions

- The damage meter row template in `Styles\DamageMeterStyles.xaml` has **two near-duplicate DataTemplates**: `DamageMeterFragmentTemplate` (live, uses `DamageMeterFragment`) and `DamageMeterSnapshotFragmentTemplate` (historical, uses `DamageMeterSnapshotFragment`). When changing bindings, the two classes need parallel property additions or only one template should be touched.
- Indentation differs between the live and snapshot templates inside the TakenDamage block — `Edit replace_all=true` can be safe for the Damage/Dps and Heal/Hps blocks (identical text in both), but the TakenDamage block requires a precise single-target edit.
- WPF user settings on Windows: when adding settings, prefer extending `SettingsObject.cs` and using `SettingsController.CurrentSettings.<field>` — that's the persistence path SAT uses.
- For Windows-specific ops (file moves, shortcuts, process management) prefer the PowerShell tool. Bash is available for POSIX-style scripts.

## User preferences observed

- Iterates fast, screenshots-driven. Prefers concise responses.
- Keeps production SAT running in the background during dev; the dev build is separate and used only for testing changes.
- Wants WoW-style UX patterns for the damage meter (Skada/Recount/Details! conventions).
