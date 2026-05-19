# Overnight status — 2026-05-19

Started: 2026-05-19T22:00:00-08:00
Last update: 2026-05-19T22:05:00-08:00

## Reconnaissance findings

Walked git log + working tree before doing any new code. Substantial work
has already shipped through this conversation but **was uncommitted** at
the time of reconnaissance — flushing it now in two logical batches
before moving on so the morning user can read the git history as the
breadcrumb trail.

### Shipped since `dffd7ab` (already committed)

- `aba96fb` Overhaul Part 1-3: naming fixes, IP chip, resource trims
- `ebddb46` Refresh docs to match the F1-F7 + silver/fame fixes
- `0161d31` Parts A1 + B from follow-up: equipment verbose log + mobs.bin → "Fox" names
- `bd51825` Parts C + D from follow-up: dead toggles, footer wording, SAFETY.md, auto-attack name
- `7d5a001` Per-player loot tracker: glanceable rollup + /loot page + clipboard

### Uncommitted at reconnaissance (this conversation's output)

About 20 files modified / created across `agent/` and `web/`. Two
commits will land them tonight:

- **UI shell + loot polish**: tab-bar navigation, in-place body swap, loot
  column headers, silver QTY fix, source prettifier, recent/only-silver
  fallback row, PARTY/GUILD/FRIEND badges, party-page stable sort, empty
  IPChip placeholder, compound-word splitter, passive filter on bound
  spell slots.
- **Combat correctness pass**: `LastFight` carry-over so rows don't snap
  to zero between fights, `touchCombat` scoped to local-player damage so
  the fight counter stops ticking on distant mob-on-mob noise, fight
  picker `mousedown → click` fix, party allowlist via
  `agent.json::alwaysIncludeNames`, loot filter mirrors the meter filter
  (party + same-guild + allowlist), session silver also credited from
  OtherGrabbedLoot for the local player.

### What's left across the four source prompts

After the catch-up commits land, the open items are:

- **`PROMPT_ui_reorg_and_loot_fixes.md`**
  - Part 2f *toggle* — Settings → "Include guildies" toggle that sends a
    `setLootFilter` command to the agent. Badges already render; the
    toggle itself was deferred. (~50 LOC across web + agent + worker.)

- **`PROMPT_followup_fixes.md`**
  - Part A — IP base / averaged label clarity. User said "worry about
    this later" during the conversation; deferred until they bring it
    up again.
  - Part B — mob name resolution via `mobs.bin`. The catalog loader is
    in (`mobs.go`, wired through `engine.SetMobCatalog`); needs verification
    that drill-in actually renders "Fox" not "#8087", and that the loot
    source prettifier hands off to the catalog when available.
  - Part C — Settings cleanup (Item Power column toggle, Crit % toggle).
    Done (`bd51825`).
  - Part D — `SAFETY.md`. Done (`bd51825`).

- **`PROMPT_loot_tracker_per_player.md`** — all main parts shipped in
  `7d5a001` plus tonight's loot-polish batch. Verification checklist
  needs a live walk.

- **`PROMPT_meter_overhaul.md`** — Part 3 (resource utilization): byObject
  index in place, dirty-gen cache in place, MeterTable sort memoised on
  `(generatedAt, activeSub, players.length)`. Spot-check passed during
  recon.

### New observations during recon

- The fight picker dropdown's "close on click outside" still ignores the
  portal'd menu contents; the recent `mousedown → click` flip should
  have fixed it but the user reported it's still broken on the last
  walkthrough. Will re-verify after the commits land and add a more
  defensive guard (ref the portal, exclude its hit-area).
- Browser tab order on a fresh load shows tabs evenly-spaced now after
  the `flex: 1` change. Good.

## Shipped tonight

(populated as commits land)

## In progress

- Catch-up commits, then Part 2f toggle.

## Blocked / needs user

- (none yet)

## Plan for next loop

1. Commit the UI shell + loot polish batch.
2. Commit the combat correctness batch.
3. Ship the Include-guildies toggle (Part 2f) + verify with `tsc` + deploy.
4. Verify mob-name resolution end-to-end (read code, confirm wiring).
5. Pick 1–2 polish items from §6 by impact-per-context.

## Notes for the morning

- I'm operating under sharply limited context, so I'm being honest about
  scope: the polish backlog (§6 A–O) is mostly out of reach in this
  session. If you wake up to one item beyond the four-prompt queue, it
  was the highest-impact one I could afford.
- Anything I skip with a clear reason goes here so you can decide
  whether to pick it up.
