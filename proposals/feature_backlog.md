# Feature Backlog Proposal

After parts 1–3 of the meter overhaul landed (naming fixes, IP chip,
resource trims), here's the backlog of "what to build next" drawn from
the competing trackers we surveyed. **None of these are implemented yet.**
Pick the ones you want and we'll build in priority order.

Each entry: name · sketch · file paths · size (S ≤ 1 day, M ~ 1 week,
L cross-cutting).

---

## Recommended first wave (highest value : effort ratio)

### 1. Session persistence with manual delete  ·  **S**

What you asked for at the end of the overhaul prompt. Today everything
lives in agent RAM and dies when the agent closes. Add lightweight
sessions:

- On `ResetSession`, archive the prior session to a JSON file under
  `%LocalAppData%\GDA\sessions\<startedAt>.json`. Files are
  human-readable so you can grep them for "what was I doing on Tuesday."
- New "Sessions" view in the web UI (or under Settings) lists archived
  sessions with name, duration, fight count, total fame/silver. A trash
  icon per row deletes that file.
- Optional: tag a session manually ("Morgana boss farm 11/19") via the
  UI; tag is saved alongside the file.

Files: `agent/internal/domain/sessions.go` (new), `agent/cmd/agent/main.go`
(load existing sessions on boot), `worker/src/meterRoom.ts` (a
`sessionList` / `sessionLoad` / `sessionDelete` command), `web/src/SessionsPanel.tsx` (new).

### 2. Map history  ·  **S**

The agent already captures zone names via `JoinFinished`. Surface them
as a running list — "where have I been this session, and for how long."
Useful for "did that mistwalker actually pay?"

Files: `agent/internal/domain/zonehistory.go` (new, append-only list),
hook in `engine.go::handleJoinResponse`, `web/src/ZoneLog.tsx` (new
panel under the activity log toggle).

### 3. Party-builder / roster panel with IP  ·  **S**  *(unlocks now that Part 2 is in)*

A separate compact panel listing every party member with their IP chip,
class chip, and weapon name. Pre-fight sanity check. Doesn't need its
own data — reads the existing snapshot.

Files: `web/src/PartyPanel.tsx` (new). Settings toggle to show/hide.

### 4. IP per-slot tooltip  ·  **S**

Hovering a player's IP chip shows the slot-by-slot breakdown (MainHand
1340, Cape 1280, …) with the same math the agent used. Frontend-only —
the agent already ships `Equipment [10]int` to the snapshot once we
expose the field over the wire.

Files: extend `PlayerSnapshot.Equipment` over the wire in
`agent/internal/domain/snapshot.go`, `web/src/IPChip.tsx` adds a hover
tooltip.

### 5. Dungeon tracker (run-scoped session)  ·  **M**

Detect dungeon entry / exit via `JoinFinished` map index pattern
(`@HIDEOUT@`, `@RANDOMDUNGEON@`, mists IDs). Open a "dungeon run" scope
when you enter, close it when you leave. Track fame/silver/respec
deltas and player damage during that scope only.

Files: `agent/internal/domain/dungeon.go` (new), wire from
`handleJoinResponse`, expose in snapshot as `Dungeon` block, render
as a strip above the meter while inside one.

---

## Second wave (real value, larger lift)

### 6. Loot logger  ·  **M**

Subscribe to the loot event family (SAT's `NewLootEvent`,
`TakeSilverFromCorpse`, etc.) and accumulate a per-session loot list:
item + quantity + zone + fight number. Persisted under the same session
file from feature #1.

Files: `agent/internal/domain/loot.go` (new), new event codes wired in
`engine.go`, `web/src/LootPanel.tsx`.

### 7. AODP price overlay  ·  **M**

When a high-value loot drop hits (T6+, artifacts), fetch the latest
market price from `https://west.albion-online-data.com/api/v2/stats/prices/<id>`
and surface "this is ~120K silver in Caerleon" on the loot row.

- Agent caches prices for ~10 min to stay polite.
- Aggregate "session estimated silver" alongside the existing fame
  total.

Files: `agent/internal/aodp/client.go` (new), `web/src/LootPanel.tsx`
extended. Depends on #6.

### 8. Killboard integration  ·  **M**

On `EventDied`, hit the public killboard
(`https://gameinfo.albiononline.com/api/gameinfo/players?search=<name>`
→ event list → match by timestamp) to enrich the fight archive with
build / gear graveyard / IP at death.

Files: `agent/internal/killboard/client.go` (new), wire from the death
handler. Each fight in the history dropdown gets a "killboard link"
when archived.

---

## Third wave (interesting but lower priority)

### 9. Cloud history per-token  ·  **M**, real privacy footprint

Persist the last N completed fights to Cloudflare KV keyed on the
SHA-256 token hash. Open the URL on a phone, see the most recent
session. Token == identity, so document the implication clearly.

Files: `worker/src/meterRoom.ts` adds a `history` API, KV namespace
in `wrangler.toml`. Web fetches on connect if local snapshot is empty.

### 10. Multi-agent same-room merge  ·  **L**

Two party members both running the agent, both pointed at the same
worker room. Today the second clobbers the first's snapshot in DO
memory. Build a merge view: union of player rows by Guid, sum stats,
prefer the higher value where the streams disagree (one agent might be
missing a player the other sees).

Files: `worker/src/meterRoom.ts` (state machine for multiple ingest
sources), `web/src/MeterTable.tsx` (no change — input is still a single
merged Snapshot). Bigger because every assumption about a single
`latest` snapshot has to be revisited.

### 11. Trade monitoring  ·  **M**

Mail + market sales/purchases parsed into session P&L. SAT's
`TradeMonitoring` is the reference. Useful for "did this dungeon
actually pay" framing — combine with #7's silver value.

Files: lots of new Photon event handlers, mostly out of damage-meter
scope.

### 12. Mechanics pane content  ·  **M**

The "Mechanics" tab is a placeholder. Make it surface deaths-by-cause
(boss mechanic vs PvP kill vs gank), interrupts landed, dodges. Needs
correlating events that don't all flow through `HealthUpdate`.

---

## Not recommended (deliberately out of scope)

- **Cross-platform support** — agent depends on Windows raw sockets for
  Photon capture. Out.
- **Twitch VOD lookup** — only useful for streamers, not the user.
- **Hard "Level 3" debuff attribution math** — would need a multiplier
  table per debuff (Frazzle 30%, Sunder 23%, …) maintained per patch.
  We have "Level 2" (damage during window) which is honest and useful.
- **Crit % per ability** — Albion doesn't expose a crit flag; SAT
  doesn't either. No reliable source.

---

## Suggested execution order

If you want a "ship something this week" plan:

1. **Session persistence (#1)** — direct user ask, isolated change.
2. **Map history (#2)** + **Party panel (#3)** — small, both unblock
   nicer UX layers.
3. **IP per-slot tooltip (#4)** — finishes Part 2's IP work.
4. **Dungeon tracker (#5)** — gets you the most "did this run pay"
   value for a 1-week build.

Hold #6–8 (loot / AODP / killboard) until #1 ships, since they all
depend on persistence to be useful.

Hold #9–12 unless / until they become a felt need.
