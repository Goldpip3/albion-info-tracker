# Per-Player Loot Tracker — make "who farmed what this session" scannable

Read `CLAUDE.md`, `ARCHITECTURE.md`, and the existing loot code first:

- `agent/internal/domain/loot.go` — already implements `LootEntry`, `LooterTotals`, and `LooterTotalsList()` (per-looter rollup with item count + silver + AODP value).
- `agent/internal/aodp/client.go` — already polls the West Albion Online Data Project API and back-fills `SilverValue` on entries within ~1 s of looting.
- `web/src/LootPanel.tsx` — existing modal with `Totals` and `Items` tabs.

**Most of the data already exists.** This work is mostly UX: making the rollup glanceable enough that a guild leader can pull it up the moment a run ends and instantly tell who farmed what, ordered by value, with the silver and item counts visible at a glance.

The user described the goal as: *"a screen that I can easily tell who looted, how much silver, and how much stuff they have gotten per session."* Read that as the acceptance criteria — if a viewer has to read more than 2 seconds to answer "who's the top farmer right now?" the screen has failed.

Default scope is the **active session** (resets when the user clicks New Session). Don't add a time-range picker yet — keep it simple.

---

## Part 1 — Beef up the rollup math

The agent's `LooterTotalsList` returns `{Name, IsLocal, ItemCount, SilverPicked, SilverValueLoot}` today. Add three fields the UI needs to render the at-a-glance view:

- `TopItemName string` — the highest-value single item this looter picked up this session (display name from items.bin localization, e.g. `"Adept's Boltcaster"`). Empty string when no priced items exist.
- `TopItemValue int64` — silver value of that top item.
- `LastPickupAt time.Time` — most-recent loot entry timestamp for this looter (drives a "active 3 sec ago" indicator).

Compute these inside the existing rollup loop in `loot.go::LooterTotalsList` — one pass, no extra allocations beyond the rollup map you already maintain. Surface them in `web/src/types.ts::LooterTotals` (parallel changes).

While you're in `loot.go`, double-check the rollup buckets:

- `ItemCount` should count individual pickups (each `OtherGrabbedLoot` event is one row), not stack quantities. Confirm this matches what the UI already shows — if it doesn't, fix the field name to make the meaning unambiguous (e.g. `Pickups` vs `Units`). Add a separate `UnitsTotal` field if both numbers are useful.
- `SilverPicked` is the direct silver picked up via `TakeSilver` events (not item-value). `SilverValueLoot` is the AODP-derived value of items. Keep them separate — the UI shows both, and you don't want to double-count.

---

## Part 2 — Promote the Totals view to a glanceable "Per-Player" screen

The current `LootPanel.tsx::TotalsTab` is a flat table. Rebuild it into the layout below. **This is the screen the user opens at the end of a run to see who looted what.** Skim-readable in 2 seconds.

### Layout (top to bottom)

**Header strip** — `SESSION LOOT · <X> looters · <Y> pickups · <Z>m elapsed`. Pull session elapsed from the existing `snapshot.session.elapsedMs`. Render in monospace at the top of the panel, muted.

**Top farmer card** — one large highlighted row at the top showing the #1 looter by `SilverValueLoot + SilverPicked`. Big number (their total value), their name, their top item name and value as a one-liner. This is the "TL;DR" — who carried this run.

**Player rows** — one row per looter, sorted by total value descending. Each row shows:

- IP chip (left) — call the existing `<IPChip>` if we have the looter's IP from the entity store; fall back to a muted dot if not.
- Player name (mid-left). If `isLocal`, tint with the local-row accent (you already use `--sk-local`).
- Horizontal value bar — width proportional to total silver value relative to the top farmer. Same visual language as the damage meter rows in `MeterTable.tsx::PlayerRow`. Color: a single neutral gold accent (use `var(--sk-silver, #d4af37)` if defined, else add it).
- Pickup count — small mono number, secondary color.
- Total silver value — mono, primary color, right-aligned.
- "Top item" sub-line under the name — the player's most valuable single drop, item name + its silver value. Small font, secondary color.
- Activity dot — a small green/grey dot that's green when `LastPickupAt` is within the last 30 seconds, fading to grey beyond that. Helps the viewer see who's still actively farming vs done.

Sort is stable by total-value descending, tie-break by name. Don't shuffle on every snapshot — memoize on `snapshot.generatedAt`.

### Click a row → drill-in

Clicking a player row swaps the panel body into a per-player drill view (no new modal, just swap content inside the same LootPanel modal — keep the header):

- Player name + total value at the top
- A chronological list of every loot entry by that player this session, descending by time
- Each entry: timestamp (relative — "2m ago"), item icon stub (we don't have icons yet; leave a 14px square for future), item name, qty, source ("from <mob/chest>" — pull from the existing `Source` field on `LootEntry`), silver value
- A "Back" button in the panel header that returns to the player list

Limit the drill list to the last 200 entries per player; older entries truncate with a "<N> more entries…" footer. Keeps DOM size sane on long sessions.

### Empty state

When `looterTotals.length === 0`, render: "No loot yet this session. Kills and chest opens will populate this once you start farming."

When the user is solo (only the local looter has entries), the layout still works — top-farmer card shows the local player, the row list is one row, that's fine. Don't add special-case "you're alone" copy.

---

## Part 3 — Add a fullscreen `/loot` route for post-run review

The modal is great for mid-run checks. After a run, the user wants to spend more than 2 seconds reviewing — they're going to paste numbers into Discord, screenshot it, sometimes scroll. Add a dedicated route `/loot` that renders the same layout fullscreen.

- New file `web/src/LootPage.tsx` (or factor `LootPanel.tsx` so the body component is shared and there are two wrappers — `LootPanel` modal vs `LootPage` route).
- Wire the route in `web/src/main.tsx` or `App.tsx` — simplest pattern: check `window.location.pathname` and conditionally render `<LootPage>` instead of `<App>`'s main view. Don't pull in a router library for one route.
- Add a small "Open as page" button in the top-right of the modal header that does `window.open("/loot?token=" + token, "_blank")` so the user can pop it onto a second monitor while still using the main meter.
- The page version drops the modal chrome (no backdrop, no Close button) and respects the existing background tokens so it looks at-home.

---

## Part 4 — Copy-to-clipboard / Discord share

Add a small `Copy Summary` button in the Per-Player view's header that copies a plain-text summary in this shape:

```
Session loot · 18m
1. Goldpipe       2.4M silver  ·  47 pickups  ·  top: Adept's Boltcaster (1.1M)
2. Lirien         1.8M silver  ·  31 pickups  ·  top: Journeyman's Cape (640K)
3. Mervyn         900K silver  ·  19 pickups  ·  top: Fiber Bundle (180K)
```

Use `navigator.clipboard.writeText`. Show a "Copied" toast for 1.5 s. This is the killer feature for guild use — anyone can paste the rollup into Discord and the team sees who did what.

Optional second button: `Copy as table` for Discord codeblock format (same data, monospace-aligned, three backticks wrapping).

---

## Part 5 — Verification

- [ ] Solo farming: pick up 5 different items in a low-tier zone, open `/loot` — exactly one row, top-farmer card shows you, pickup count matches.
- [ ] With one party member nearby (or with `ALBION_AGENT_SHOW_ALL=1`): both players show with separate rollups, sorted by silver value descending.
- [ ] AODP pricing populates within ~1 s for known items. Items with no AODP price (e.g. brand-new patch items) render value as `—`, not `0`. The rollup excludes them from sort weighting until priced.
- [ ] Click a row → drill view shows chronological entries. Click Back → returns to list with sort preserved.
- [ ] `Copy Summary` produces text that pastes cleanly into Discord (no smart quotes, no emoji garbage).
- [ ] New Session resets the rollup. (Hit New Session in the header, the panel empties.)
- [ ] Switching from solo to group via re-zone doesn't double-count anyone's earlier solo entries.

---

## What's deliberately out of scope (don't build now)

- Multi-agent shared room aggregation. Today each agent has its own room. A shared "guild room" where multiple agents push to the same DO and the rollup merges across agents is a real feature but it's its own design (auth, dedupe by looter guid not name, conflict resolution when two agents both saw the same pickup). File it as a separate proposal in `proposals/feature_backlog.md` if not already there.
- Per-zone loot scoping. Today the rollup is session-wide. Zone-scoped buckets are useful but require zone metadata on each `LootEntry`, which means a `loot.go` schema bump. Defer.
- Per-fight scoping. Same reasoning. Useful but a real refactor.
- Loot graveyard from PvP kills. `EventCorpseLootChanged` exists but isn't handled. Out of scope for this sprint.
- Mob/chest source name resolution. Sources today render as raw mob ObjectIds or chest indices. The "mob name resolution" task is already its own follow-up (see `PROMPT_followup_fixes.md` Part B). Once that lands, the source field renders mob names natively — no work here.
- Item icons. We don't have icon assets locally. Leave the 14px placeholder square so the layout is icon-ready when icons land.

---

## File-by-file summary

- `agent/internal/domain/loot.go` — extend `LooterTotals` with `TopItemName`, `TopItemValue`, `LastPickupAt`; populate in the existing rollup loop.
- `web/src/types.ts` — mirror the new fields on `LooterTotals` interface.
- `web/src/LootPanel.tsx` — replace `TotalsTab` body with the new layout (header strip, top-farmer card, player rows with bars, activity dot). Add drill-in state. Add `Copy Summary` button.
- `web/src/LootPage.tsx` — new file, fullscreen variant. Share the body component with the modal so they don't drift.
- `web/src/App.tsx` or `web/src/main.tsx` — route on `/loot`.

Total expected change size: ~400-600 lines added, mostly UI. The agent-side work is < 50 lines. If you find yourself rewriting the loot capture code, you've gone off the rails — that part already works.

When all five Verification boxes check, ship.
