# Overnight Autonomy v2 — finish the polish pass while the user sleeps

You are working unattended for the next 6–10 hours. The user is asleep. Work has been in flight today on the UI reorganization — top-tab navigation has already shipped (METER · LOOT · PARTY · SESSIONS routes are live as of the last screenshot). You're picking up mid-stream. Your job is to (1) **reconnoiter first** to see exactly where the work stands now, (2) finish the open chunks across the four follow-up prompts, and (3) apply a polish pass inspired by the best DPS / stats tools in other MMOs. Wake the user up to something that feels like a real product.

This is a meta-prompt about *how* to work overnight. The *what* lives across five other prompt files in the repo root plus the polish backlog in §6 below.

---

## §1. Reconnaissance — do this first, before reading any source prompt

The codebase is moving. The state described in `REVIEW_post_overhaul.md` is hours old; the four `PROMPT_*.md` files were written against an earlier state. Don't blindly execute their checklists — re-verify the current state first.

In your first 15 minutes:

1. `git log --oneline -30` and `git status` from the repo root. Capture which commits landed since the last entry in `REVIEW_post_overhaul.md` (which referenced commit `dffd7ab`).
2. Open https://albion-meter-web.pages.dev/ in the browser tools you have access to. Walk every tab (METER, LOOT, PARTY, SESSIONS), screenshot each. Compare against the screenshots referenced in `PROMPT_ui_reorg_and_loot_fixes.md` and `REVIEW_post_overhaul.md`.
3. For each of the four prompts in the work queue (§5), open the file and tick boxes for items that are already done in the current code. Don't redo finished work — that's the most common autonomous-agent failure mode.
4. Write a `STATUS_overnight.md` file in the repo root with a "Reconnaissance findings" section as the first entry. List: what's shipped since `dffd7ab`, what's left in each of the four prompts, what new bugs you spotted during the walk. **This is your map for the rest of the night.**

Only after this reconnaissance do you begin executing.

---

## §2. Methodology — every chunk of work follows this loop

For every task you pick up, run the same five-step loop. No exceptions.

1. **Research.** Re-read every file the task touches. Re-read the *callers* of files you'll edit so you don't break a contract. For Photon protocol, equipment params, or game-data parsing, cross-reference the C# SAT source under `src/StatisticsAnalysisTool/` — that's ground truth for protocol indices.
2. **Plan.** Write a short plan (3–10 bullets, plain prose) into the task's section of `STATUS_overnight.md`. State the files you'll change, the contracts you'll preserve, the risks. **Re-read the plan before executing.** Anything that smells like a 200+ line rewrite gets broken down further. Bias toward small, reversible changes.
3. **Execute.** Make the change. Keep the diff scoped to the plan. If you discover during execution that the plan was wrong, stop, update `STATUS_overnight.md`, and continue with the new approach. Don't quietly drift.
4. **Verify.** Run the local checks in §3. If any fail, fix before moving on — don't pile broken work.
5. **Commit.** One commit per logical chunk, with a clear message that references the source prompt and the task name. Push to `go-port`. Never push to `main`. Never force-push.

Update `STATUS_overnight.md` after every loop with: what shipped, what's left, anything the user needs to see in the morning.

---

## §3. Verification — what "works" means

Run these in order. Stop on the first failure and fix it before continuing.

- `cd agent && go build ./...` — must succeed.
- `cd web && npm run build` — must succeed. TypeScript errors fail this.
- For web changes, deploy a preview: `cd web && npx wrangler pages deploy ./dist --project-name albion-meter-web --commit-dirty=true`. Capture the preview URL.
- Open https://albion-meter-web.pages.dev/ in Chrome. Take a screenshot. Compare against your reconnaissance screenshots from §1 for unintended regressions. The agent may not be live — use `?demo=1` to verify against demo data.
- For agent changes, build the binary: `go build -o agent.exe ./cmd/agent`. You can't run it (admin + live game), but a clean build is the floor.
- For game-data parsing changes, run the probe: `go run ./cmd/probe`. Confirm output plausible: items.bin > 10000 entries, spells.bin > 8000, mobs.bin > 0 (once you load it).

If a verification step fails for a build-environment reason (not the code), document it in the status doc and continue. If the code is the cause, fix it.

---

## §4. Quality bars — when something is "done"

Done means:

- All verification checks pass
- Committed and pushed to `go-port`
- The relevant checklist box in the source prompt has its ✓ ticked (edit the prompt file, that's the breadcrumb the user sees)
- `STATUS_overnight.md` has a one-line entry

NOT done:

- "It compiles" — floor, not bar
- "I rewrote it but didn't verify" — verify, or roll back
- "TypeScript got complex so I added `any`" — solve the type, or write down why you couldn't and revert
- "80% there, finish in the morning" — finish or roll back. Half-merged is worse than not starting.

---

## §5. Work queue — priority order

Read these five files end-to-end during reconnaissance. Tick what's done. Then execute the rest in this order:

1. **`PROMPT_ui_reorg_and_loot_fixes.md`** — top-tab navigation is shipped (confirmed via screenshots), so audit the rest: Loot view column headers, silver-pile QTY bug, loot-source prettifier, party panel reshuffle, spell-slot mislabeling, compound-word splitter, looter-source badges, "Include guildies" toggle. Most of these are still open.
2. **`PROMPT_followup_fixes.md`** — IP average vs base (Part A), mob name resolution `#8087 → Fox` (Part B), Settings cleanup of redundant column toggles (Part C), `SAFETY.md` (Part D). Part B intersects with the loot-source prettifier from #1 — once `mobs.bin` is loaded, the prettifier becomes a real lookup.
3. **`PROMPT_loot_tracker_per_player.md`** — most already shipped (`dffd7ab`). Audit for items that were marked pending in `REVIEW_post_overhaul.md`. Don't redo finished work.
4. **`PROMPT_meter_overhaul.md`** — the original. Spot-check Part 3 (resource utilization): byObjectId index, dirty-generation cache, MeterTable sort memoization. If any regressed or were skipped, fix.
5. **§6 of this prompt — the polish backlog.** Only after #1–#4 are clean. Pick items from §6 based on time remaining and impact.

Finish #1 cleanly (every verification box checked, committed, pushed) before starting #2. The user judges the morning's progress on whether the existing experience reads as polished, not on how many new features got bolted on.

---

## §6. Polish backlog — features stolen from the best DPS tools in MMO history

These are *optional*. Reach for them only after the four-prompt queue in §5 is clean. Each one is sized small (≤2 hours) / medium (~half a day) / large (full day+). Pick by impact-per-hour given the time remaining.

I researched the leading DPS and stats tools across other MMOs to inform this list. Each item below cites the source.

### A. Color-coded percentile chip on each fight — small

**From [FFLogs / Warcraft Logs](https://www.fflogs.com/help/ranks/)**, where every parse gets a colored percentile badge (100 tan, 99 pink, 95 orange, 75 purple, 50 blue, 25 green, white below). Visceral signal of how the run went.

You don't need a global corpus to start. Compute the percentile against the user's own session archive: "this fight's DPS was 78th percentile of your last 30 fights on this weapon." Render a small color-coded pill next to the fight number in `FightHeader`. Color scale: copy FFLogs's palette directly so it reads correctly to anyone who's used logs before.

File touches: `agent/internal/domain/history.go` to expose per-weapon percentile cache; `web/src/MeterTable.tsx` for the chip; new `web/src/PercentileChip.tsx`.

### B. Death recap drawer — small

**From [Details! Damage Meter](https://www.curseforge.com/wow/addons/details-damage-meter) and [Cactbot OopsyRaidsy](https://overlayplugin.github.io/cactbot/)**, both of which surface "what killed you" as a first-class view. Cactbot's framing: "mistake tracking and death reporting, to reduce the time wasted understanding what went wrong."

When the local player dies, freeze a snapshot of the last 5–10 hits that landed on them plus an HP-over-time sparkline. Show as a slide-up drawer for ~10 seconds with a `Pin` option. The data is already in the agent (HealthUpdate events with the affected = local objectId).

File touches: `agent/internal/domain/events.go` to track recent-damage-taken ring buffer per entity; `web/src/DeathRecap.tsx` new component; integrate into the Activity Log component.

### C. Boon / debuff uptime panel — medium

**From [ArcDPS](https://arcdps.com/)** in GW2, where "the Buffs > Boons tab shows your boon uptime, important for support players, but DPS players should also look at the table to see if their personal uptimes were unusually low compared to their subgroup."

Albion has buffs and debuffs that matter (Cripple, Frazzle, Bloodlust, healing-debuff effects). The `Assists` tab in the drill-in already tracks debuff windows. Extend it to also cover beneficial buffs — uptime % per player per buff, color-coded against expected baselines. New "Buffs" tab in DrillIn next to "Assists".

File touches: `agent/internal/domain/engine.go::handleActiveSpellEffects` (already runs); add a `BuffsBySpell` map on Entity parallel to `AssistsBySpell`; surface in snapshot. UI: extend `web/src/DrillIn.tsx`.

### D. Skada-vs-Recount style DPS calc toggle — small

**From [Skada vs Recount comparison](https://www.curseforge.com/wow/addons/skada)**: the meaningful difference between meters is *how* they compute DPS — combat-time (Skada) vs active-time (Recount). Skada divides total damage by total combat duration; Recount divides by time spent actively dealing damage.

This is a settings toggle: `DPS calculation → Combat time (Skada-style) | Active time (Recount-style)`. The agent computes both — UI picks which to display. Useful for healer / hybrid roles where "DPS while doing damage" is more flattering than "DPS across the whole fight."

File touches: `agent/internal/domain/combat.go::DPS()` add a second method `ActiveDPS()`; snapshot exposes both; `web/src/SettingsPanel.tsx` adds the toggle; `web/src/MeterTable.tsx` reads the setting.

### E. Encounter timeline scrubber — large

**From [Warcraft Logs replay system](https://www.warcraftlogs.com/help/ranks/)**, where every fight is a scrubbable timeline showing damage / heals / deaths / cooldowns on a horizontal time axis. The killer feature of WCL.

For archived fights only (the existing `FightArchive`), render a 0..durationMs scrubber. Above it, three lanes: damage events (colored by player), heal events, deaths (red drop markers). Hovering a point shows the event detail. Scrubbing time updates a side panel with "DPS at this moment". This is the feature that turns the tool from a damage meter into a fight reviewer.

File touches: extend `FightArchive` with per-second damage buckets; new `web/src/FightTimeline.tsx` SVG-based renderer; route mount `/fight/:n`.

### F. Hotkeys — small

**From [Skada](https://www.curseforge.com/wow/addons/skada), [Details!](https://www.curseforge.com/wow/addons/details-damage-meter), every serious WoW addon**. Keyboard-driven users live or die on hotkeys.

- `M` / `L` / `P` / `S` cycle the top tabs (Meter / Loot / Party / Sessions)
- `D` / `H` / `T` cycle Damage / Healing / Tank panes on the Meter view
- `R` reset session (with confirm)
- `Esc` close any open drill-in or modal
- `?` open a hotkey reference overlay

File touches: `web/src/App.tsx` global keydown listener; new `web/src/HotkeyHelp.tsx`.

### G. Active-window flash on big events — small

**From [Cactbot's alert tiers](https://overlayplugin.github.io/cactbot/)** (info / alert / alarm escalating colors and sizes). On a passive analytics page this becomes: a brief border flash on the page when a noteworthy event happens (you crit > 2× your average, a party member died, a session milestone hit). Toggleable; off by default to not annoy.

File touches: `web/src/Footer.tsx` or `web/src/App.tsx` event flash subscription; setting in `SettingsPanel.tsx`.

### H. Discord-ready battle report image export — medium

**From [Charlemagne (Destiny 2)](https://www.warmind.io/) and [zKillboard battle reports](https://zkillboard.com/)**, both of which excel at producing a single image you paste into chat and the whole team can read. This is the killer share format for guilds.

Generate a PNG of the current session summary: top farmer card + top 5 looters + fight stats. Render via SVG → PNG in the browser (no server work). Button next to the existing "Copy Summary" on the Loot page.

File touches: new `web/src/exportImage.ts` using SVG-foreignObject + canvas; button wired into `LootBody.tsx`.

### I. Mini-bars per spell within each row — small

**From [Details! Damage Meter](https://www.curseforge.com/wow/addons/details-damage-meter)**, where each player's row has 2–3 tiny mini-bars showing the relative contribution of their top spells without needing to drill in. Lets the viewer see the whole party's loadout at a glance.

For each player row in `MeterTable.tsx`, add a thin row of 3 horizontal mini-bars below the main bar, one per their top-3 spells, each tinted to the spell's class accent. Skip rows under a height threshold. Falls back gracefully when there are no spells.

File touches: `web/src/MeterTable.tsx::PlayerRow`.

### J. Session pace ETA — small

**From [Wise Old Man (OSRS)](https://wiseoldman.net/)**, the runescape stats tracker, which projects when you'll hit your next goal at the current XP/hour. Same math applies to fame.

In the FAME / SILVER / COMBAT FAME / MIGHT cards, when not in combat for > 1 minute, show a small ETA underneath: "at this pace, 1M fame by 6:42 PM". Useful for grind sessions.

File touches: `web/src/SessionStrip.tsx`.

### K. Composition visualization — small

The composition counter already exists (`COMPOSITION · 5/20`) but it's text-only. Make it visual: five small role chips (T·H·R·M·S) with counts, colored to the existing role tokens. Hover to see member names per role. Mirrors the way every MMO raid frame shows role split.

File touches: `web/src/App.tsx` header strip.

### L. Better empty states — small

Empty states across PARTY, LOOT, SESSIONS currently read as bare one-liner sentences. Add a faint icon (the GDA mark at low opacity) plus a "What you'll see here" sub-line: PARTY → "Other players in your party will appear with their IP, weapon, and bound abilities." Same for LOOT and SESSIONS. This is the polish that signals "real product" vs "prototype."

File touches: `web/src/PartyPanel.tsx` (or page version), `web/src/LootBody.tsx`, `web/src/SessionsPanel.tsx`.

### M. Theme picker — small

The settings panel already has a per-row "Theme accent" color picker (5 swatches). Extend with full theme presets: Dark Default · Dark High-Contrast · Dark Warm. Each preset sets the full `--sk-*` token palette in `web/src/main.tsx` or `index.css`. Persist in `useSettings.ts`.

File touches: `web/src/SettingsPanel.tsx`, `web/src/useSettings.ts`, new `web/src/themes.ts`.

### N. In-combat band — small

A 2–3px band at the bottom of the meter view that turns orange when the agent reports `fight.inCombat === true`. Easy at-a-glance "are we fighting" without needing to look at the timer. Mirrors the in-combat indicators in essentially every MMO UI.

File touches: `web/src/App.tsx` or `web/src/Footer.tsx`.

### O. Audio cues — small, careful

**From Cactbot's audio alerts.** Three optional sounds: ding on session milestone (every 100K fame), soft chime on dungeon entry, quiet beep on personal best DPS for the current weapon. Off by default. Toggleable per-sound in settings.

File touches: `web/src/sounds.ts` new; `web/src/SettingsPanel.tsx`; small `.mp3` assets in `public/audio/`.

---

## §7. Status doc — the one thing the user reads in the morning

`STATUS_overnight.md` at the repo root. Update after every loop. Format:

```
# Overnight status — <date>

Started: <ISO timestamp>
Last update: <ISO timestamp>

## Reconnaissance findings
- <what's shipped since dffd7ab>
- <what's still pending in each of the four prompts>
- <new bugs spotted>

## Shipped tonight
- <task name> — <commit hash> — <one-line summary>

## In progress
- <task name> — <what step you're on>

## Blocked / needs user
- <task name> — <why you couldn't continue>

## Plan for next loop
- <next task>

## Notes for the morning
- <anything the user should know — surprises, design calls you made>
```

Keep it under 250 lines. Prune oldest Shipped entries (they're in git). "Blocked" and "Notes" sections are sacred — never lose those.

---

## §8. When to stop and wait

You MUST escalate (write to "Blocked / needs user", move on) when:

- Choosing between two reasonable designs where reverting costs > 30 minutes. Make the call, document it, and if you're wrong the user reverts in the morning.
- Anything needing a credential you don't have.
- Anything touching money — domains, plans, etc.
- Pushing to `main` or any branch other than `go-port`. Forbidden.
- Deleting > 200 lines in a single commit. Split first.
- Changing the agent↔web wire protocol in a way that breaks older agent binaries. The user has a cached build. If you need a protocol bump, bump the `v` field AND keep backward compat for one version.

Two-attempts-max on any bug. On the third failure, document the symptom + what you tried in "Blocked / needs user" and move on. The morning user will help unstick.

---

## §9. Boundaries — out of scope even if tempting

- Don't redesign the meter view. The damage/healing/tank panes work, leave them.
- Don't add features past §5 + §6. If you see a quick-win not on either list, write it to "Notes for the morning."
- Don't refactor for its own sake. Renames, restructures, folder moves — no.
- Don't touch the C# SAT code under `src/`. Read-only reference. Looking is encouraged; editing forbidden.
- Don't touch `.git/`, `node_modules/`, or `agent.json`. `agent.json` contains the live push token.

---

## §10. Practical guidance — working alone

- **Read more than you write.** Every committed line is backed by a file you re-read this loop.
- **Smaller commits, faster cycles.** 50–150 meaningful lines per commit. If a commit message has "and also" in it, split.
- **Lean on the probe and `?demo=1`.** Closest thing to a live agent.
- **Be honest in the status doc.** Skipped verification? Say so. Uncertain about a call? Say so. The user trusts a doc that admits uncertainty more than one that claims perfection.
- **Don't loop on a bug.** Two attempts, then document and move on.
- **The 5-minute rule.** Before a non-trivial change, ask: "If I came back tomorrow with no context, would this diff make sense?" If no, leave a brief code comment.

---

## §11. Start now

1. Create `STATUS_overnight.md`. First entry: `Started reconnaissance. Reading git log + walking the live site.`
2. Reconnaissance per §1.
3. Read the five prompt files. Tick what's done. Update the status doc with findings.
4. Begin with the next open item in `PROMPT_ui_reorg_and_loot_fixes.md`.
5. Loop per §2 / §3.
6. When §5 is clean, dip into §6 by impact.

When you hit a real stopping point — queue empty or enough blockers documented that further progress requires the user — make a final commit with a clean status doc and stop. Don't fill time with cosmetic churn.

Good luck. Make it look like a real product by morning.

---

## Sources

- [Skada Damage Meter](https://www.curseforge.com/wow/addons/skada) — combat-time DPS calc, modular window system
- [Details! Damage Meter](https://www.curseforge.com/wow/addons/details-damage-meter) — mini-bars, death recap, plugin pattern
- [Warcraft Logs](https://www.warcraftlogs.com/help/ranks/) — percentile rankings, timeline replay
- [FF Logs](https://www.fflogs.com/help/ranks/) — color-coded parses, historical vs current rankings
- [ArcDPS](https://arcdps.com/) — boon uptime tracking, plugin ecosystem
- [Cactbot](https://overlayplugin.github.io/cactbot/) — alert tiers, death reporting / OopsyRaidsy
- [Advanced Combat Tracker](https://advancedcombattracker.com/) — log parsing baseline for FFXIV
- [Wise Old Man](https://wiseoldman.net/) — XP-pace ETA projections, group milestones
- [zKillboard](https://zkillboard.com/) — battle reports cross-referencing multiple players
- [Charlemagne](https://www.warmind.io/) — Discord-bot image card render pattern
