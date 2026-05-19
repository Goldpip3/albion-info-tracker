# Post-Overhaul Review — what shipped, what's still off, and is this safe to keep using

Date: 2026-05-19. Reviewer: Cowork (without a live agent connected). Site: https://albion-meter-web.pages.dev/.

I clicked through every reachable surface on the live site, diffed the four commits Claude Code shipped on top of `aba96fb`, and ran a fresh research pass on Sandbox Interactive's stance on third-party tools. Three deliverables follow: a what-works / what-doesn't summary, a Terms-of-Service / ban-risk review, and a separate companion file (`PROMPT_followup_fixes.md`) with the concrete bug list to feed back to Claude Code.

---

## 1. What I could verify works

The codebase moved a long way. Confirmed by inspection of the merged code:

- **Auto-attack normalization** — `engine.go::localizedSpellName` catches `AUTO_ATTACK` / `AUTOATTACK` substrings and returns the string `"Auto Attack"`. Universal across every weapon, no per-weapon allowlist. ✓
- **Spell-name fallback chain + override table** — `gamedata/localization.go` and `gamedata/spells_override.go` are in place. Caltrops and Flickershot are seeded in the override map. ✓
- **Role-label typo + ordering** — `role.go` no longer prints `"ARLIGHT BLASTER"`. Both `DUALCROSSBOW_AVALON` and `CROSSBOW_AVALON` resolve to `"ARCLIGHT BLASTERS"` (lines 50 and 57). ✓
- **IPChip replaces the 3-letter weapon chip** — both `MeterTable.tsx` (the live row) and `DrillIn.tsx` (the per-player drilldown) now render `<IPChip>` instead of `<ClassChip>`. Hover reveals a per-slot breakdown. ✓
- **Equipment + quality decoding** — `parseEquipmentParams` reads param 40/2 for equipment, param 41/3 for quality, param 7 for active spells. Quality is being captured, in theory. ✓ (needs live verification — see §3.)
- **Resource trims** — `Snapshot()` builds, `topSpells` / `topTargets` paths still run per push. The push interval is 200 ms (the footer shows `push 0.2s`). I did not see an explicit dirty-generation cache landing, but the snapshot is leaner than before because the equipment payload is now pre-computed in `applyEquipmentArray` and the IPChip's slot list is built once per equipment change rather than per push. Acceptable.

**Net-new features beyond what was asked for in the prompt**, all visible on the live site:

- A **Party · Live Loadout** modal (bottom-right `PARTY` button) — shows everyone tracked with their IP + class.
- A **Loot · Friends' Farming** modal (`LOOT` button) — has `Totals` and `Items` tabs, shows entries from the loot logger. AODP integration is in `agent/internal/aodp/client.go` so live silver values should populate.
- A **Sessions on Disk** modal (`SESSIONS` button) — archives a session each time you hit `New Session`.
- A **Tank** pane sits alongside Damage / Healing on the main meter (`Tank` mode in the header).
- A **Combat Fame** stat card on the top strip (Respec was relabeled to Combat Fame in commit `95c76ac`).
- An **Activity log** drawer (top-right header icon) with ALL / DEATHS / BIG HITS / HEALS filters.
- A **dungeon scope** (`agent/internal/domain/dungeon.go`) that scopes session stats per dungeon run.

That's six features past the original Part 1–3 scope — Claude Code burned through Part 4 too. Most of it looks well structured but I can't fully validate without live data.

## 2. What still needs work

The full bug list with reproduction details and file paths lives in `PROMPT_followup_fixes.md`. The headline items:

**IP shows Base IP, not Average IP.** Your screenshot shows Albion's character panel reporting `Average Item Power: 1153` and `Base Average Item Power: 1004`. The 149-IP gap is the quality bonus across slots (and possibly a small mastery contribution). The agent's `AverageItemPower` function in `gamedata/itempower.go` already takes a quality array and includes the quality bonus in the math — but in practice the meter is likely showing ~1004, not 1153. Two probable causes: (a) the quality parameter index Albion uses on the current patch is not 41 / 3 as the code assumes, so `qualities[]` stays all-zero and the function falls back to `quality=0` everywhere; (b) the quality array is being shipped under a different parameter or a different shape (uint8 vs int16) and `intsOfArray` is silently returning empty. Either way: needs to be verified with `ALBION_AGENT_VERBOSE=1` and a quick log of the parsed quality array next time you connect.

**Targets show as `#8087` instead of `Fox`.** The Go agent recognizes the `NewMob` Photon event code (`gamecodes/events.go:131`) but has no handler for it. The C# SAT codebase has the full pattern — `NewMobEventHandler` → `CombatEventTracker.TrackNewMob` → `MobDataResolver` reads `mobs.bin`. We need a `MobCatalog` loader (parallel to `ItemCatalog`), a `NewMob` handler that registers the mob's ObjectId in the Store with its localized name, and `topTargets` to resolve via that store before falling back to `#<id>`. `agent/internal/gamedata/paths.go:27` already lists `mobs.bin` in the expected files — it's just not loaded.

**Quality is plausibly correct, just untested.** Per above — without a live capture I can't tell whether the chip shows 1004 or 1153. The follow-up prompt has Claude Code instrument the decoder and verify on next run.

**IP exposed as an optional column in Settings.** The Settings panel I opened shows `Item Power` as one of several toggleable columns under `DISPLAY → Columns`. The intent of Part 2 was that IP **replaces** the 3-letter chip in the leftmost position — which it does — but Item Power also exists as a secondary column toggle. Either remove the redundant column option, or repurpose it as "show IP after the player name" if that's wanted as a second IP surface (not necessary IMO — the chip is enough).

**The footer reads `v0.6.0 · stale`.** "Stale" here means the agent hasn't pushed recently. Make sure when you fire it up next, the indicator flips to `live` and stays there.

**Crit % column.** The Settings panel exposes a `Crit %` toggle. The original CLAUDE.md notes Albion doesn't expose a crit flag in HealthUpdate. This is going to render as `—` / 0 forever unless someone reverse-engineers a crit detector. Either pull it from the UI or label it `coming soon` like Mechanics.

## 3. ToS / ban-risk review

Bottom line: **what you're doing today is in the same risk category as Statistics Analysis Tool, Albibong, and mazurwiktor/albion-online-stats — all of which have run publicly for years without coordinated bans of their users.** That's "gray, but historically safe." Some specifics:

The authoritative statement is Korn's [Regarding 3rd Party Software](https://forum.albiononline.com/index.php/Thread/124819-Regarding-3rd-Party-Software-and-Network-Traffic-aka-do-not-cheat-Update-16-45-U/) post from Feb 2020, still the operative policy as of 2026. The relevant carve-outs:

- *In-game overlays are not allowed, including damage meters as overlays.*
- *Data made available on a website, with no in-game advantage and no overlay, is generally OK.*
- *Anything providing a direct PvE / PvP / gathering advantage is not OK.*

The architecture as it stands today maps cleanly to the "website" pattern: passive UDP capture, JSON snapshot pushed to a Cloudflare Worker, rendered in a browser window that's not attached to the Albion process. There's no DLL injection, no memory reads, no client modification, no automation, no in-game overlay. That's the exact pattern Sandbox said is acceptable.

**Lower risk territory:**

- Damage / healing / DPS / IP attribution from your own party — post-event analytics, no scouting value, indistinguishable from SAT.
- Loot logging from your own pickups + market-value lookup via AODP — AODP is itself an established community service that Sandbox has tolerated; this is consumer-side use of it.
- Session archive on disk, fame/silver/might rollup — all derived from in-game UI events already visible to the player.

**Higher risk territory (handle carefully):**

- **Multi-agent guild aggregation.** If members of a guild push to a shared room and the worker fans out a *real-time* "who is where, who's tracked nearby" view during an active fight, that crosses into the scouting-advantage zone. As a *post-fight* tool for reviewing the engagement, it's fine. The distinction matters: render the merged view *after* a fight ends, not during.
- **Hostile-presence intel.** A "we've seen GuildX 17 times this week in Black Zone 03" rollup is fine when stale. A live alert that fires the moment a hostile spawns on your zone could be argued as a real-time advantage. Cap the freshness at +N minutes if you build it.
- **Anything that reads enemy position data.** The Photon stream carries some position info; the agent should keep treating that as out of scope (it does today). If a future feature needs position data, that's the line not to cross.

**What I'd do to lock this in:**

1. Add a one-line statement to `README.md` saying: passive packet capture only, browser-rendered, no in-game overlay, no automation, no client modification.
2. If you want to be fully above-board, email `support@albiononline.com` describing what the tool does (link to the public site, the Worker URL, the GitHub repo) and ask for confirmation. Sandbox's policy specifically invites this. The worst case is "no, please don't" and you stop. The likely case is "this matches the website pattern, you're fine."
3. Do not run the agent as an Administrator service that auto-starts — keep it user-launched. That's a small thing but separates this from automation tooling that runs in the background.
4. Don't add overlay rendering. Stay in the browser window. Even if the user asks for it later — Korn's post explicitly forbids it.

**Risk verdict:** low for personal / party use, medium-low for guild use as long as the guild view is post-fight rather than live scouting. Lower than installing OBS during PvP. Significantly lower than running any cheat tool that touches the client.

---

## Sources

- [Albion Online — Regarding 3rd Party Software (Feb 2020)](https://forum.albiononline.com/index.php/Thread/124819-Regarding-3rd-Party-Software-and-Network-Traffic-aka-do-not-cheat-Update-16-45-U/)
- [Albion Online Terms and Conditions](https://albiononline.com/terms_and_conditions)
- [Albion Online — DPS Meter forum thread](https://forum.albiononline.com/index.php/Thread/127270-Dps-Meter/)
- [Statistics Analysis Tool (Triky313)](https://github.com/Triky313/AlbionOnline-StatisticsAnalysis) — prior art
- [Albibong](https://github.com/imjangkar/albibong) — prior art
- [Albion Online Stats (mazurwiktor)](https://github.com/mazurwiktor/albion-online-stats) — prior art
- [Albion Online Data Project](https://www.albion-online-data.com/) — used by this project
- [Albion Online Wiki — Item Power](https://wiki.albiononline.com/wiki/Item_Power)
