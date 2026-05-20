package domain

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Goldpip3/albion-info-tracker/agent/internal/aodp"
	"github.com/Goldpip3/albion-info-tracker/agent/internal/gamecodes"
	"github.com/Goldpip3/albion-info-tracker/agent/internal/gamedata"
	"github.com/Goldpip3/albion-info-tracker/agent/internal/photon"
)

// verbose toggles per-event debug logging. Set ALBION_AGENT_VERBOSE=1 to
// enable; useful for figuring out why party detection isn't kicking in.
var verbose = os.Getenv("ALBION_AGENT_VERBOSE") != ""

// SetVerbose flips verbose logging at runtime. The package-level var is
// captured from the env at init, so a --verbose CLI flag (parsed in
// main, after init) needs this setter to take effect. dbg reads the var
// per-call, so flipping it before events flow is enough.
func SetVerbose(v bool) { verbose = v }

func dbg(format string, args ...any) {
	if verbose {
		log.Printf(format, args...)
	}
}

// Engine wires Photon events into the entity store and combat tracker.
// It is the public surface the rest of the agent talks to.
type Engine struct {
	store *Store

	// guard protects flushedFights and other engine-only fields. The Store
	// has its own lock for entity state.
	guard sync.Mutex
	now   func() time.Time

	// items, when set, lets handleNewCharacter classify a player's weapon
	// into a role + 3-letter class chip. Nil is fine — entities just get
	// "?" / "—" until the catalog loads or never loads.
	items *gamedata.ItemCatalog

	// spells, when set, resolves CausingSpellIndex from HealthUpdate to a
	// uniquename for the drill-in screen. Nil is fine — the drill-in just
	// shows the numeric index.
	spells *gamedata.SpellCatalog

	// mobs, when set, resolves the MobIndex param on NewMob events to a
	// uniquename so the drill-in's "Targets" tab can read "Fox" instead
	// of "#8087".
	mobs *gamedata.MobCatalog

	// mobNames caches resolved English mob names keyed by ObjectId.
	// Populated on NewMob, consumed by topTargets when the target isn't
	// a tracked player entity. Mobs aren't entered into the Store —
	// the store is reserved for player-shaped entities.
	mobNamesMu sync.RWMutex
	mobNames   map[int64]string

	// loc, when set, turns spells.bin / items.bin / mobs.bin uniquenames
	// into the user-facing English names shown in Albion's tooltips
	// ("Flickershot" instead of "CROSSBOW_FLICKERSHOT_E").
	loc *gamedata.Localization

	// events is the activity log ring buffer surfaced in the snapshot.
	events *eventBuffer

	// Fight lifecycle. CombatStart is when the current fight began;
	// LastDamageAt is the most-recent damage tick we saw. Out → in
	// transitions increment FightNumber and reset every entity's
	// Current bucket (Overall persists for the session total).
	fightMu       sync.Mutex
	fightNumber   int
	fightStart    time.Time
	lastDamageAt  time.Time
	inCombat      bool
	// fightHistory is a ring of the last maxFightHistory completed fights.
	// Protected by fightMu since archive happens on the fight-end transition.
	fightHistory []FightArchive

	// Session economy + lifecycle, protected by sessionMu.
	sessionMu sync.Mutex
	session   SessionStats

	// zoneName tracks the current Albion zone for the FightHeader subtitle.
	zoneMu   sync.Mutex
	zoneName string

	// sessions is the on-disk archive of completed sessions. Optional —
	// when nil the agent skips persistence and resets just clear RAM.
	sessions *SessionsStore

	// party persists the confirmed party roster across agent restarts.
	// Optional — when nil the roster lives only in RAM (the old
	// behaviour) and a restart loses it.
	party *PartyStore

	// zones is the append-only log of zone visits, capped at zoneLogCap.
	// Updated from handleJoinResponse via noteZoneEntry.
	zonesMu sync.Mutex
	zones   []ZoneVisit

	// dungeon is the in-flight DungeonRun (nil = not in a dungeon).
	// Opens / closes on JoinResponse based on classifyDungeon.
	dungeonMu sync.Mutex
	dungeon   *DungeonRun

	// lootLog is the bounded loot history. Updated on OtherGrabbedLoot.
	lootMu  sync.Mutex
	lootLog []LootEntry

	// prices is the AODP price client. Optional — nil = no value
	// estimation, items still record with Quantity but SilverValue=0.
	prices *aodp.Client

	// alwaysInclude is the case-sensitive name allowlist that forces
	// players into the meter even when they're outside the local
	// player's guild. Sourced from agent.json::alwaysIncludeNames so
	// users can keep non-guild friends visible without flipping the
	// global show-all escape hatch.
	alwaysIncludeMu sync.RWMutex
	alwaysInclude   map[string]struct{}

	// lootFilterMode controls the loot view's membership scope. Two
	// values are valid: "partyGuild" (default) includes confirmed
	// party members, same-guild players, and the alwaysInclude
	// allowlist; "party" strips the guild branch so only confirmed
	// party + allowlist + local are visible. Toggled by the
	// "setLootFilter" command from the web Settings panel.
	filterModeMu sync.RWMutex
	lootFilter   string

	// assistMu guards the debuff-window tracker + recent-casts buffer.
	assistMu       sync.Mutex
	activeWindows  map[int64]map[int]*debuffWindow // target → spell → window
	recentCasts    []recentCast                    // ring of recent CastFinished events
	recentCastsCap int

	// dirtyGen ticks every time a Photon handler mutates state the
	// snapshot consumer cares about. The push client compares against
	// its last-sent generation and skips the send when nothing has
	// changed — collapses 95% of out-of-combat traffic to a heartbeat
	// roughly every second (the heartbeat still goes out so viewers
	// know the agent is alive).
	dirtyGen atomic.Uint64

	// pendingEquip caches FULL equipment arrays + qualities from
	// CharacterEquipmentChanged events that arrived before we knew about
	// the target entity. Albion fires equipment events for the local player
	// BEFORE the Join response, so without this cache the local player's
	// class and IP never resolve.
	// Keyed by ObjectId. Cleared when consumed.
	pendingEquipMu sync.Mutex
	pendingEquip   map[int64]pendingEquipEntry
}

// pendingEquipEntry holds a 10-slot equipment + qualities + 14-slot
// spell array for an ObjectId whose entity hasn't been registered yet.
// Replayed by applyCachedEquipment as soon as the entity appears.
type pendingEquipEntry struct {
	Equipment    [10]int
	Qualities    [10]int
	ActiveSpells [14]int
}

// debuffWindow is one open "this player's debuff is on this target right
// now" interval. Damage that lands on the target while it's open accrues
// to DamageDuring; closing the window adds Uptime + DamageDuring to the
// caster's AssistsBySpell totals.
type debuffWindow struct {
	SpellIdx        int
	CasterObjectId  int64
	Started         time.Time
	DamageDuring    int64
}

// recentCast remembers a recent CastFinished event so we can infer the
// caster of a debuff that just appeared on a target. ActiveSpellEffectsUpdate
// only tells us "this effect is on this target now" — never who cast it.
type recentCast struct {
	SpellIdx int
	CasterId int64
	At       time.Time
}

// Zone returns the current zone label.
func (e *Engine) Zone() string {
	e.zoneMu.Lock()
	defer e.zoneMu.Unlock()
	return e.zoneName
}

const (
	// combatEnterIdle: how long after the prior damage tick we still
	// consider the player engaged. Albion's own in-combat flag stays on
	// for ~5s of inactivity; we want a slightly tighter window so distinct
	// pulls don't merge into one fight.
	combatEnterIdle = 4 * time.Second
	// fightAutoEnd: longer idle threshold that closes the current fight.
	fightAutoEnd = 6 * time.Second
)

// SetAlwaysIncludeNames replaces the meter's non-guild allowlist.
// Empty / nil clears it. Case-sensitive — names must match exactly the
// Name Albion ships in NewCharacter (no whitespace, no rendering quirks).
func (e *Engine) SetAlwaysIncludeNames(names []string) {
	e.alwaysIncludeMu.Lock()
	defer e.alwaysIncludeMu.Unlock()
	if len(names) == 0 {
		e.alwaysInclude = nil
		return
	}
	m := make(map[string]struct{}, len(names))
	for _, n := range names {
		if n != "" {
			m[n] = struct{}{}
		}
	}
	e.alwaysInclude = m
}

// AlwaysIncludes reports whether name is on the agent.json allowlist.
// Safe to call concurrently. Used by the snapshot's membership filter
// so non-guild friends survive the guild-only fallback.
func (e *Engine) AlwaysIncludes(name string) bool {
	if name == "" {
		return false
	}
	e.alwaysIncludeMu.RLock()
	defer e.alwaysIncludeMu.RUnlock()
	_, ok := e.alwaysInclude[name]
	return ok
}

// NewEngine constructs an Engine backed by a fresh Store.
func NewEngine() *Engine {
	now := time.Now()
	return &Engine{
		store:          NewStore(),
		now:            time.Now,
		events:         newEventBuffer(),
		session:        SessionStats{Start: now},
		activeWindows:  make(map[int64]map[int]*debuffWindow),
		recentCastsCap: 64,
		pendingEquip:   make(map[int64]pendingEquipEntry),
		mobNames:       make(map[int64]string),
	}
}

// ResetSession clears every running counter — combat stats, per-spell
// breakdowns, deaths, session economy, fight counter, fight history,
// activity log — and stamps a fresh session start time. Called when the
// user clicks "New session" in the UI.
//
// Before zeroing, archives the prior session to the local-disk sessions
// store (when configured). The archive captures metadata + final
// economy totals; the full fight detail isn't preserved (too big for
// long-running users), just enough to answer "what was that session."
func (e *Engine) ResetSession() {
	now := e.now()
	if e.sessions != nil {
		e.archiveCurrentSession(now)
	}
	e.store.mu.Lock()
	for _, ent := range e.store.byGuid {
		ent.Current.Reset()
		ent.Overall.Reset()
		ent.LastFight.Reset()
		ent.BySpell = nil
		ent.BySpellSession = nil
		ent.ByTarget = nil
		ent.ActiveEffects = nil
		ent.AssistsBySpell = nil
		ent.Deaths = 0
	}
	e.store.mu.Unlock()
	e.resetLootOnNewSession()

	e.assistMu.Lock()
	e.activeWindows = make(map[int64]map[int]*debuffWindow)
	e.recentCasts = nil
	e.assistMu.Unlock()

	e.fightMu.Lock()
	e.fightNumber = 0
	e.fightStart = time.Time{}
	e.lastDamageAt = time.Time{}
	e.inCombat = false
	e.fightHistory = nil
	e.fightMu.Unlock()

	e.sessionMu.Lock()
	e.session.Reset(now)
	e.sessionMu.Unlock()

	e.events = newEventBuffer()
}

// HandleCommand dispatches a command from the browser (forwarded by the
// Worker over the same WebSocket). The command may have an Arg payload
// for commands that need a target (e.g. deleteSession <id>).
func (e *Engine) HandleCommand(action, arg string) {
	switch action {
	case "resetSession":
		e.ResetSession()
	case "deleteSession":
		if e.sessions != nil && arg != "" {
			_ = e.sessions.Delete(arg)
			e.markDirty()
		}
	case "addPartyMember":
		// Manual roster add from the web — covers mixed-guild parties
		// the agent never saw form. arg is the player's guid string.
		if g, err := ParseGuid(arg); err == nil && !g.IsZero() {
			e.store.MarkInParty(g, true)
			e.persistParty()
			e.markDirty()
		}
	case "removePartyMember":
		if g, err := ParseGuid(arg); err == nil && !g.IsZero() {
			e.store.MarkInParty(g, false)
			e.persistParty()
			e.markDirty()
		}
	case "setLootFilter":
		// arg is the meter scope: "party" (confirmed party + allowlist),
		// "partyGuild" (default; + same-guild), or "everyone" (all
		// activity, for ZvZ). Anything else falls back to the default so
		// a buggy client can't strand the agent in a bad state.
		mode := "partyGuild"
		switch arg {
		case "party", "everyone":
			mode = arg
		}
		e.filterModeMu.Lock()
		e.lootFilter = mode
		e.filterModeMu.Unlock()
		e.markDirty()
	}
}

// LootFilterMode returns "partyGuild" (default) or "party" depending on
// whether the user has asked for guild members to be filtered out of
// the loot view. Safe for concurrent reads from snapshot / loot code.
func (e *Engine) LootFilterMode() string {
	e.filterModeMu.RLock()
	defer e.filterModeMu.RUnlock()
	if e.lootFilter == "" {
		return "partyGuild"
	}
	return e.lootFilter
}

// archiveCurrentSession captures the current session's metadata into
// the local-disk archive. Called immediately before ResetSession zeroes
// state. Quietly no-ops when nothing's worth archiving (empty session).
func (e *Engine) archiveCurrentSession(endedAt time.Time) {
	if e.sessions == nil {
		return
	}
	e.sessionMu.Lock()
	startedAt := e.session.Start
	sess := ArchivedSession{
		Id:          fmt.Sprintf("%d", startedAt.UnixMilli()),
		StartedAt:   startedAt,
		EndedAt:     endedAt,
		DurationMs:  endedAt.Sub(startedAt).Milliseconds(),
		FameTotal:   e.session.FameTotal,
		SilverTotal: e.session.SilverTotal,
		RespecTotal: e.session.RespecTotal,
		MightTotal:  e.session.MightTotal,
		DeathsTotal: e.session.DeathsTotal,
	}
	e.sessionMu.Unlock()
	e.fightMu.Lock()
	sess.FightCount = e.fightNumber
	e.fightMu.Unlock()
	e.zoneMu.Lock()
	sess.Zone = e.zoneName
	e.zoneMu.Unlock()
	// Best-effort local-name lookup.
	if local := e.store.localGuidEntity(); local != nil {
		sess.LocalName = local.Name
	}
	// Skip empties so we don't pollute the archive when the user clicks
	// "New session" twice in a row.
	if sess.DurationMs < 1000 && sess.FameTotal == 0 && sess.SilverTotal == 0 {
		return
	}
	if err := e.sessions.Save(sess); err != nil {
		log.Printf("session archive: %v", err)
	}
}

// spellName looks up a uniquename for a spell index, or returns "".
func (e *Engine) spellName(idx int) string {
	if e.spells == nil {
		return ""
	}
	return e.spells.Name(idx)
}

// SetItemCatalog wires an items.bin catalog into the engine for weapon-based
// role classification. Safe to call before or after capture starts.
func (e *Engine) SetItemCatalog(c *gamedata.ItemCatalog) {
	e.items = c
}

// SetSpellCatalog wires a spells.bin catalog into the engine so the drill-in
// screen can resolve a CausingSpellIndex to a uniquename.
func (e *Engine) SetSpellCatalog(c *gamedata.SpellCatalog) {
	e.spells = c
}

// SetMobCatalog wires a mobs.bin catalog into the engine. Used by the
// NewMob handler to resolve MobIndex into a friendly name shown in the
// drill-in's Targets tab.
func (e *Engine) SetMobCatalog(c *gamedata.MobCatalog) {
	e.mobs = c
}

// MobName returns the cached English name for a mob ObjectId, or "" if
// the agent never saw a NewMob event for that ObjectId.
func (e *Engine) MobName(objectId int64) string {
	e.mobNamesMu.RLock()
	defer e.mobNamesMu.RUnlock()
	return e.mobNames[objectId]
}

// SetSessionsStore wires a disk-backed sessions archive. When set,
// ResetSession will write a snapshot of the current session to the
// store before zeroing counters. Optional — the agent runs fine without.
func (e *Engine) SetSessionsStore(s *SessionsStore) {
	e.sessions = s
}

// SetPartyStore wires the disk-backed party roster. When set, confirmed
// party membership is persisted on every change and restored on boot.
func (e *Engine) SetPartyStore(s *PartyStore) {
	e.party = s
}

// persistParty writes the current roster to disk. No-op when no store
// is wired. Cheap — a single small JSON file.
func (e *Engine) persistParty() {
	if e.party == nil {
		return
	}
	if err := e.party.Save(e.store.PartyRefs()); err != nil {
		log.Printf("  party persist: %v", err)
	}
}

// RestoreParty rehydrates the roster from disk on startup. Each ref
// becomes a named, IsInParty entity with ObjectId=0; it rebinds to
// live combat on the next NewCharacter exactly like a PartyJoined
// member. Stale rosters (older than partyFreshness) are ignored by the
// store's Load.
func (e *Engine) RestoreParty() {
	if e.party == nil {
		return
	}
	refs, err := e.party.Load()
	if err != nil {
		log.Printf("  party restore: %v", err)
		return
	}
	n := 0
	for _, ref := range refs {
		g, err := ParseGuid(ref.Guid)
		if err != nil || g.IsZero() {
			continue
		}
		e.store.UpsertByGuid(g, 0, ref.Name, "")
		e.store.MarkInParty(g, true)
		n++
	}
	if n > 0 {
		e.markDirty()
		log.Printf("  party restore: %d member(s) from disk", n)
	}
}

// SetPriceClient wires the AODP price client for loot value estimation.
// Optional — without it loot entries still log but SilverValue stays 0.
func (e *Engine) SetPriceClient(c *aodp.Client) {
	e.prices = c
}

// Sessions returns the configured store (or nil). Used by the snapshot
// builder to surface the archived-session list to the web.
func (e *Engine) Sessions() *SessionsStore {
	return e.sessions
}

// SetLocalization wires a localization.bin lookup so spells/items resolve
// to their in-game tooltip names instead of raw uniquenames.
func (e *Engine) SetLocalization(l *gamedata.Localization) {
	e.loc = l
}

// localizedSpellName resolves a spell index to its in-game display name.
// Fallback chain (in order):
//   1. Auto-attack uniquenames ("CROSSBOW_AUTO_ATTACK_JUMP" etc.) collapse
//      to "Auto Attack" — Albion's own localization for these is "1" or
//      "Light Attack" which isn't useful in the drill-in.
//   2. localization.bin lookup via @SPELLS_/ @SPELL_ / @MOB_ABILITIES_ /
//      @SPELLDESC_ / @ITEMS_<u>_SPELL prefix family (with trailing
//      _<digit> stripping).
//   3. Hardcoded overrides for stubborn abilities (Caltrops, Flickershot,
//      etc.) that don't show up anywhere in localization.bin.
//   4. Final prettifier: strip known weapon-family prefixes + slot
//      suffixes, Title-Case what's left. "BOLTCASTER_CALTROPS_E" → "Caltrops",
//      "CROSSBOW_FLICKERSHOT_E" → "Flickershot".
//
// Returns "" only when there's no spell at that index. Callers can fall
// back to a numeric label ("#1234") in that case.
func (e *Engine) localizedSpellName(idx int) string {
	// Auto-attacks arrive in HealthUpdate with CausingSpellIndex = -1
	// (no CastFinished, no uniquename in spells.bin). Treat any non-positive
	// index as the player's basic attack so the drill-in shows "Auto Attack"
	// instead of "#-1".
	if idx <= 0 {
		return "Auto Attack"
	}
	if e.spells == nil {
		return ""
	}
	uniqueName := e.spells.Name(idx)
	if uniqueName == "" {
		return ""
	}
	upperName := strings.ToUpper(uniqueName)
	if strings.Contains(upperName, "AUTO_ATTACK") || strings.Contains(upperName, "AUTOATTACK") {
		return "Auto Attack"
	}
	if e.loc != nil {
		if loc := e.loc.SpellName(uniqueName); loc != "" {
			return loc
		}
	}
	if name := gamedata.SpellOverride(uniqueName); name != "" {
		return name
	}
	return gamedata.PrettifySpell(uniqueName)
}

// SpellCatalog returns the configured spell catalog, or nil.
func (e *Engine) SpellCatalog() *gamedata.SpellCatalog {
	return e.spells
}

// Store exposes the underlying store for read-only consumers like the
// snapshot renderer.
func (e *Engine) Store() *Store { return e.store }

// Handlers returns a photon.Handlers value bound to this engine.
func (e *Engine) Handlers() photon.Handlers {
	return photon.Handlers{
		OnEvent:    e.onEvent,
		OnRequest:  e.onRequest,
		OnResponse: e.onResponse,
	}
}

// markDirty bumps the snapshot generation counter. Called from every
// Photon handler that touches state the snapshot consumer reads. Atomic
// because handlers may run on different goroutines than the push client's
// reader. Cheap — a single Add(1).
func (e *Engine) markDirty() {
	e.dirtyGen.Add(1)
}

// DirtyGen returns the current snapshot generation. The push client
// compares against its last-sent value to decide whether to skip a tick.
func (e *Engine) DirtyGen() uint64 {
	return e.dirtyGen.Load()
}

func (e *Engine) onEvent(ev photon.EventData) {
	code := realCode(ev.Parameters, ev.Code)
	e.markDirty()
	switch gamecodes.Event(code) {
	case gamecodes.EventHealthUpdate:
		e.handleHealthUpdate(ev.Parameters)
	case gamecodes.EventHealthUpdates:
		e.handleHealthUpdates(ev.Parameters)
	case gamecodes.EventNewCharacter:
		dbg("NewCharacter %v", ev.Parameters)
		e.handleNewCharacter(ev.Parameters)
	case gamecodes.EventCharacterEquipmentChanged:
		dbg("CharacterEquipmentChanged %v", ev.Parameters)
		e.handleEquipmentChanged(ev.Parameters)
	case gamecodes.EventChangeEquipment:
		dbg("ChangeEquipment %v", ev.Parameters)
		e.handleEquipmentChanged(ev.Parameters)
	case gamecodes.EventCastFinished:
		e.handleCastFinished(ev.Parameters)
	case gamecodes.EventActiveSpellEffectsUpdate:
		e.handleActiveSpellEffects(ev.Parameters)
	case gamecodes.EventOtherGrabbedLoot:
		dbg("OtherGrabbedLoot %v %s", ev.Parameters, e.localIdStr())
		e.handleOtherGrabbedLoot(ev.Parameters)
	case gamecodes.EventNewMob:
		e.handleNewMob(ev.Parameters)
	case gamecodes.EventPartyJoined:
		dbg("PartyJoined %v", ev.Parameters)
		e.handlePartyJoined(ev.Parameters)
	case gamecodes.EventPartyPlayerJoined:
		dbg("PartyPlayerJoined %v", ev.Parameters)
		e.handlePartyPlayerJoined(ev.Parameters)
	case gamecodes.EventPartyPlayerLeft:
		dbg("PartyPlayerLeft %v", ev.Parameters)
		e.handlePartyPlayerLeft(ev.Parameters)
	case gamecodes.EventPartyDisbanded:
		dbg("PartyDisbanded")
		e.store.ResetParty()
		if e.party != nil {
			_ = e.party.Clear()
		}
	case gamecodes.EventMountStart:
		dbg("MountStart %v", ev.Parameters)
		e.handleMountStart(ev.Parameters)
	case gamecodes.EventNewMountObject:
		dbg("NewMountObject %v", ev.Parameters)
		e.handleNewMountObject(ev.Parameters)
	case gamecodes.EventJoinFinished:
		dbg("JoinFinished %v", ev.Parameters)
		e.handleJoinFinished(ev.Parameters)
	case gamecodes.EventDied:
		dbg("Died %v", ev.Parameters)
		e.handleDied(ev.Parameters)
	case gamecodes.EventUpdateFame:
		dbg("UpdateFame %v", ev.Parameters)
		e.handleUpdateFame(ev.Parameters)
	case gamecodes.EventUpdateReSpecPoints:
		dbg("UpdateReSpec %v", ev.Parameters)
		e.handleUpdateReSpec(ev.Parameters)
	case gamecodes.EventMightAndFavorReceivedEvent:
		dbg("MightAndFavor %v", ev.Parameters)
		e.handleMightAndFavor(ev.Parameters)
	case gamecodes.EventTakeSilver:
		dbg("TakeSilver %v %s", ev.Parameters, e.localIdStr())
		e.handleTakeSilver(ev.Parameters)
		// UpdateMoney is intentionally NOT credited (it fires on every
		// wallet sync — deposits, purchases — and would double-count).
		// The cases below are DIAGNOSTIC ONLY (verbose mode): they log
		// the silver-adjacent events we currently ignore so a single
		// kill→loot capture reveals which one actually carries
		// open-world mob silver. None of them credit yet — wire the
		// confirmed carrier into creditSilver once identified.
	case gamecodes.EventUpdateMoney:
		dbg("UpdateMoney %v %s", ev.Parameters, e.localIdStr())
	case gamecodes.EventRemoveSilver:
		dbg("RemoveSilver %v %s", ev.Parameters, e.localIdStr())
	case gamecodes.EventNewSilverObject:
		dbg("NewSilverObject %v %s", ev.Parameters, e.localIdStr())
	case gamecodes.EventPartySilverGained:
		dbg("PartySilverGained %v %s", ev.Parameters, e.localIdStr())
	}
}

// localIdStr renders the local player's ObjectId + name for verbose
// silver diagnostics, so a param-0 / param-2 mismatch against the
// event's ids is visible at a glance.
func (e *Engine) localIdStr() string {
	local := e.store.localGuidEntity()
	if local == nil {
		return "local=unknown"
	}
	return fmt.Sprintf("local ObjId=%d Name=%q", local.ObjectId, local.Name)
}

func (e *Engine) onRequest(photon.OperationRequest) {}

func (e *Engine) onResponse(resp photon.OperationResponse) {
	code := realCode(resp.Parameters, resp.OperationCode)
	switch gamecodes.Op(code) {
	case gamecodes.OpJoin:
		dbg("Join response %v", resp.Parameters)
		e.handleJoinResponse(resp.Parameters)
	}
}

// handleJoinResponse processes the operation response sent when the local
// user enters a zone. It carries the only authoritative source of the
// local player's identity — the server never broadcasts the local user
// via NewCharacter, so this is how we learn our own ObjectId + Guid.
//
// Mirrors SAT's JoinResponseHandler. Params used:
//   0 → UserObjectId, 1 → UserGuid, 2 → Username,
//   8 → MapIndex (zone token), 58 → GuildName.
func (e *Engine) handleJoinResponse(p map[byte]any) {
	objectId, _ := paramLong(p, 0)
	guid, _ := paramGuid(p, 1)
	name, _ := paramString(p, 2)
	guild, _ := paramString(p, 58)
	if guid.IsZero() {
		return
	}
	ent := e.store.UpsertByGuid(guid, objectId, name, guild)
	e.store.SetLocalGuid(guid)
	// Local player is always in their own party for damage-meter purposes,
	// even when actually solo — that matches SAT's behaviour.
	e.store.MarkInParty(guid, true)
	// Zone label — JoinResponse param 8 is the MapIndex (e.g.
	// "KEEPERS_HIDE_FARM_2" or "BLACK_03"). Surface as the FightHeader
	// subtitle so the user knows where the data is coming from. Also
	// append to the zone history.
	if mi, ok := paramString(p, 8); ok && mi != "" {
		pretty := prettyZone(mi)
		e.setZone(mi)
		now := e.now()
		e.noteZoneEntry(pretty, now)
		// Dungeon edges — opening / closing a run-scoped scope based
		// on whether the zone we just joined looks like a dungeon
		// instance. classifyDungeon returns ("", false) for cities,
		// open world, hideouts.
		if kind, isDungeon := classifyDungeon(mi); isDungeon {
			e.openDungeon(pretty, kind, now)
		} else {
			e.closeDungeon(now)
		}
	}
	// Apply any equipment event that arrived BEFORE this Join response.
	// Albion broadcasts the local user's CharacterEquipmentChanged a tick
	// or two before the Join response itself, so the equipment landed in
	// pendingEquip without an entity to attach to. Replay now.
	e.applyCachedEquipment(ent)
}

// setZone caches the current zone label for inclusion in snapshots.
func (e *Engine) setZone(name string) {
	e.zoneMu.Lock()
	e.zoneName = prettyZone(name)
	e.zoneMu.Unlock()
}

// prettyZone turns Albion's internal map tokens into something readable.
// "KEEPERS_HIDE_FARM_2" → "Keepers Hide Farm 2". "MISTS_06" → "Mists 06".
func prettyZone(raw string) string {
	if raw == "" {
		return ""
	}
	parts := splitToken(raw)
	for i, p := range parts {
		if p == "" {
			continue
		}
		// Keep all-numeric tokens as-is.
		allDigits := true
		for _, r := range p {
			if r < '0' || r > '9' {
				allDigits = false
				break
			}
		}
		if allDigits {
			continue
		}
		parts[i] = upperFirst(p)
	}
	return joinSpaces(parts)
}

func splitToken(s string) []string {
	out := make([]string, 0, 4)
	cur := make([]byte, 0, 16)
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '_' || c == '-' {
			if len(cur) > 0 {
				out = append(out, string(cur))
				cur = cur[:0]
			}
			continue
		}
		cur = append(cur, c)
	}
	if len(cur) > 0 {
		out = append(out, string(cur))
	}
	return out
}

func upperFirst(s string) string {
	if s == "" {
		return s
	}
	b := []byte(s)
	if b[0] >= 'a' && b[0] <= 'z' {
		b[0] -= 32
	}
	for i := 1; i < len(b); i++ {
		if b[i] >= 'A' && b[i] <= 'Z' {
			b[i] += 32
		}
	}
	return string(b)
}

func joinSpaces(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += " "
		}
		out += p
	}
	return out
}

// handleHealthUpdate processes the SINGULAR variant (EventCode 6): one
// scalar hit. Auto-attacks mostly arrive here. Most ability / multi-
// source damage arrives BATCHED as EventCode 7 (handleHealthUpdates).
func (e *Engine) handleHealthUpdate(p map[byte]any) {
	affected, _ := paramLong(p, 0)
	change, _ := paramDouble(p, 2)
	newHP, _ := paramDouble(p, 3)
	causer, _ := paramLong(p, 6)
	spellIdx, _ := paramLong(p, 7)
	e.applyHealthChange(affected, causer, change, newHP, spellIdx, e.now())
}

// handleHealthUpdates processes the BATCHED variant (EventCode 7): a
// single affected target plus parallel per-hit arrays of health change,
// new HP, causer, and spell index. Albion batches most ability and
// multi-source damage here; the singular handler mostly sees auto-
// attacks. The agent previously had no case for code 7, so all batched
// hits were dropped — the meter under-counted to near auto-attack-only
// totals. Mirrors SAT's HealthUpdatesEvent array layout.
func (e *Engine) handleHealthUpdates(p map[byte]any) {
	affected, _ := paramLong(p, 0)
	changes := paramDoubleArray(p, 2)
	newHPs := paramDoubleArray(p, 3)
	causers := paramLongArray(p, 6)
	spells := paramLongArray(p, 7)
	n := len(changes)
	for _, l := range []int{len(newHPs), len(causers), len(spells)} {
		if l > n {
			n = l
		}
	}
	if n == 0 {
		return
	}
	dbg("HealthUpdates n=%d affected=%d", n, affected)
	now := e.now()
	for i := 0; i < n; i++ {
		e.applyHealthChange(affected, atLong(causers, i), atDouble(changes, i),
			atDouble(newHPs, i), atLong(spells, i), now)
	}
}

// applyHealthChange folds one decoded hit into combat state. Shared by
// both the singular (EventCode 6) and batched (EventCode 7) handlers so
// they accumulate identically. change < 0 is damage, > 0 is heal.
func (e *Engine) applyHealthChange(affected, causer int64, change, newHP float64, spellIdx int64, now time.Time) {
	if causer == 0 {
		return
	}

	causerEnt := e.store.ByObjectId(causer)
	affEnt := e.store.ByObjectId(affected)

	// Combat / fight bookkeeping. Scope it to the LOCAL player —
	// fights only start / continue when the local player is doing or
	// taking damage. Without this guard, every nearby mob hitting
	// every other thing in render range advances the fight counter,
	// which is why the meter would tick fights up while the user
	// stood idle. Has to happen before stats accumulate so a brand-
	// new fight zeroes the Current bucket first.
	if e.localInvolved(causerEnt, affEnt) {
		e.touchCombat(now)
	}

	// Damage is delivered as negative HealthChange; heal as positive.
	if change < 0 {
		dmg := int64(-change + 0.5)
		if dmg <= 0 {
			return
		}
		if causerEnt != nil {
			e.store.mu.Lock()
			recordDamage(&causerEnt.Current, &causerEnt.Overall, dmg, now)
			recordSpell(causerEnt, int(spellIdx), dmg)
			recordSpellSession(causerEnt, int(spellIdx), dmg)
			recordTarget(causerEnt, affected, dmg)
			e.store.mu.Unlock()
		}
		// Attribute the hit to any open debuff windows on the target.
		if affected != 0 {
			e.assistMu.Lock()
			e.chargeWindowsLocked(affected, dmg)
			e.assistMu.Unlock()
		}
		if affEnt != nil {
			e.store.mu.Lock()
			recordTakenDamage(&affEnt.Current, &affEnt.Overall, dmg, now)
			e.store.mu.Unlock()
		}
		// Log hits where at least the actor is known — drops anonymous
		// mob-on-mob noise that we can't render anyway.
		if causerEnt != nil {
			e.recordHit(causerEnt, affEnt, dmg, int(spellIdx), now)
		}
		return
	}
	if change > 0 {
		heal := int64(change + 0.5)
		if heal <= 0 {
			return
		}
		newHPi := int64(0)
		if newHP > 0 {
			newHPi = int64(newHP + 0.5)
		}
		effective, overheal := splitOverheal(affEnt, heal, newHPi)
		if causerEnt != nil {
			e.store.mu.Lock()
			recordHeal(&causerEnt.Current, &causerEnt.Overall, effective, overheal, now)
			e.store.mu.Unlock()
			e.recordHealEvent(causerEnt, affEnt, effective, int(spellIdx), now)
		}
	}
}

// splitOverheal divides a raw heal value into (effective, overheal). With
// no MaxHealth known for the target, the heal is taken at face value and
// no overheal is counted — better to under-report than to fabricate.
func splitOverheal(target *Entity, heal, newHP int64) (effective, overheal int64) {
	if target == nil || target.MaxHealth <= 0 || newHP <= 0 {
		return heal, 0
	}
	prevHP := newHP - heal
	if prevHP >= target.MaxHealth {
		return 0, heal
	}
	headroom := target.MaxHealth - prevHP
	if heal <= headroom {
		return heal, 0
	}
	return headroom, heal - headroom
}

// handleDied logs the death into the activity log AND bumps the death
// counter on the matching tracked entity + the session total. Mirrors
// SAT's DiedEvent — param 2 is the victim's name, param 10 is the killer.
func (e *Engine) handleDied(p map[byte]any) {
	victim, _ := paramString(p, 2)
	killer, _ := paramString(p, 10)
	if victim == "" {
		return
	}
	e.recordDeathEvent(victim, killer, e.now())
	e.store.mu.Lock()
	for _, ent := range e.store.byGuid {
		if ent.Name == victim {
			ent.Deaths++
			break
		}
	}
	e.store.mu.Unlock()
	e.sessionMu.Lock()
	e.session.DeathsTotal++
	e.sessionMu.Unlock()
}

// handleUpdateFame folds a fame-change event into the session total.
//
// SAT's UpdateFameEvent exposes several values:
//   param 1: TotalPlayerFame — running lifetime cumulative
//   param 2: FameWithZoneMultiplier — fame for THIS event including zone bonus
//   param 5: IsPremiumBonus — when true, premium adds +50%
//   param 10: SatchelFame — bag bonus, separate from the kill
//   param 17: BonusFactorInPercent — situational % bonus
//
// We previously tracked the delta of TotalPlayerFame, which under-counted
// because TotalPlayerFame lags by ~1 tick relative to in-game popups, and
// premium / satchel bonuses are sometimes booked separately. Compute the
// per-event total the way SAT does:
//   total = (FameWithZoneMultiplier + PremiumFame + SatchelFame) * BonusFactor
//   PremiumFame = FameWithZoneMultiplier * 0.5 when IsPremiumBonus else 0
//
// FixPoint internal units (10_000 = 1 fame). Falls back to the old
// delta-tracking when param 2 is absent (rare; defensive).
func (e *Engine) handleUpdateFame(p map[byte]any) {
	fameWithZone, hasZoneFame := paramLong(p, 2)
	if !hasZoneFame {
		// Fall back: delta of TotalPlayerFame.
		total, ok := paramLong(p, 1)
		if !ok {
			return
		}
		e.sessionMu.Lock()
		e.session.AccumulateFame(total)
		e.sessionMu.Unlock()
		return
	}
	premium := int64(0)
	if isPrem, _ := paramBool(p, 5); isPrem {
		premium = fameWithZone / 2
	}
	satchel, _ := paramLong(p, 10)
	bonus := int64(0)
	// param 17 ships as float (% multiplier above 1.0). Apply if present.
	if raw, ok := p[17]; ok {
		if f, ok := raw.(float32); ok && f > 0 {
			bonus = int64(float64(fameWithZone+premium+satchel) * float64(f))
		} else if f, ok := raw.(float64); ok && f > 0 {
			bonus = int64(float64(fameWithZone+premium+satchel) * f)
		}
	}
	totalGained := fameWithZone + premium + satchel + bonus
	if totalGained <= 0 {
		return
	}
	e.sessionMu.Lock()
	// Direct add — this is per-event gained, not a cumulative reading,
	// so the delta-baseline logic in AccumulateFame would mis-count.
	e.session.FameTotal += totalGained
	e.sessionMu.Unlock()
}

// creditSilver folds a silver gain into the session total, and into the
// mob-only subtotal when isMob. Single funnel so every silver path —
// TakeSilver, looted piles, and (once confirmed) the open-world mob
// carrier — accumulates consistently. amount is FixPoint (10_000 = 1
// silver). Caller must NOT hold sessionMu.
func (e *Engine) creditSilver(amount int64, isMob bool) {
	if amount <= 0 {
		return
	}
	e.sessionMu.Lock()
	e.session.SilverTotal += amount
	if isMob {
		e.session.MobSilverTotal += amount
	}
	e.sessionMu.Unlock()
}

// handleTakeSilver folds a silver-pickup event into the session total.
// Param 3 = YieldPreTax (FixPoint), param 5 = GuildTax, param 6 = ClusterTax.
// Net silver banked = YieldPreTax - GuildTax (cluster tax is the cluster's
// share, taken from the pre-tax yield before guild tax — SAT does
// YieldAfterTax = YieldPreTax - GuildTax and that's what they bank).
//
// Credits the local player whether they're the looter (param 0) or the
// target (param 2) — SAT matches either. Mob classification is
// best-effort: if a non-local id involved is a known mob (in mobNames),
// tag it as mob silver. Refined once the true open-world carrier is
// confirmed from a verbose capture.
func (e *Engine) handleTakeSilver(p map[byte]any) {
	objectId, _ := paramLong(p, 0)
	target, _ := paramLong(p, 2)
	yieldPre, ok := paramLong(p, 3)
	if !ok || yieldPre <= 0 {
		return
	}
	guildTax, _ := paramLong(p, 5)
	yieldAfter := yieldPre - guildTax
	if yieldAfter <= 0 {
		return
	}
	local := e.store.localGuidEntity()
	if local == nil || local.ObjectId == 0 {
		return
	}
	if objectId != local.ObjectId && target != local.ObjectId {
		return
	}
	isMob := false
	for _, id := range []int64{objectId, target} {
		if id != 0 && id != local.ObjectId && e.MobName(id) != "" {
			isMob = true
			break
		}
	}
	e.creditSilver(yieldAfter, isMob)
}

// handleUpdateReSpec adds the gained respec credits delta to the session
// total. Param 2 is documented (per SAT) as GainedReSpecPoints in FixPoint
// internal units (10_000 = 1 credit). Recent observations suggest param 2
// may actually be the LIFETIME total on some patches — values can balloon
// into 50K+ over a normal session, which is impossible. To be safe, also
// read param 0 (the [_, lifetimeTotal] array form) when present, and
// derive a delta against the prior lifetime reading. Whichever value is
// smaller (and positive) wins per event.
func (e *Engine) handleUpdateReSpec(p map[byte]any) {
	gained, _ := paramLong(p, 2)
	// param 0 carries the lifetime total in an array — element [1].
	lifetime, _ := paramLongAt(p, 0, 1)
	gain := chooseRespecGain(gained, lifetime, &e.session)
	if gain <= 0 {
		return
	}
	e.sessionMu.Lock()
	e.session.AccumulateRespec(gain)
	e.sessionMu.Unlock()
}

// chooseRespecGain reconciles the two possible respec param semantics.
// If we have a lifetime baseline, we trust the delta against the prior
// lifetime reading. Otherwise we trust the per-event gained value.
func chooseRespecGain(gainedParam, lifetimeParam int64, s *SessionStats) int64 {
	if lifetimeParam > 0 {
		// First read seeds the baseline silently.
		if !s.prevRespecKnown {
			s.prevRespec = lifetimeParam
			s.prevRespecKnown = true
			return 0
		}
		delta := lifetimeParam - s.prevRespec
		s.prevRespec = lifetimeParam
		if delta > 0 {
			return delta
		}
		return 0
	}
	if gainedParam > 0 {
		return gainedParam
	}
	return 0
}

// paramLongAt reads a numeric element from an array-shaped param. Used
// by handleUpdateReSpec where param 0 is [_, lifetimeTotal] long[].
func paramLongAt(p map[byte]any, key byte, idx int) (int64, bool) {
	v, ok := p[key]
	if !ok {
		return 0, false
	}
	switch a := v.(type) {
	case []int32:
		if idx < len(a) {
			return int64(a[idx]), true
		}
	case []int64:
		if idx < len(a) {
			return a[idx], true
		}
	case []uint32:
		if idx < len(a) {
			return int64(a[idx]), true
		}
	case []any:
		if idx < len(a) {
			switch x := a[idx].(type) {
			case int32:
				return int64(x), true
			case int64:
				return x, true
			case uint32:
				return int64(x), true
			}
		}
	}
	return 0, false
}

// handleMightAndFavor folds a MightAndFavorReceivedEvent into the session.
// Param 1 = Might gained (FixPoint internal; 10_000 = 1 might). The event
// also carries Premium/Bonus might + Favor variants in params 2-7 but we
// only surface Might in the meter UI.
func (e *Engine) handleMightAndFavor(p map[byte]any) {
	gained, ok := paramLong(p, 1)
	if !ok || gained <= 0 {
		return
	}
	e.sessionMu.Lock()
	e.session.AccumulateMight(gained)
	e.sessionMu.Unlock()
}

// recordTarget increments the per-target damage bucket. Caller must
// hold store.mu. targetId of 0 means we couldn't resolve the affected
// entity — skip rather than create a bogus bucket.
func recordTarget(ent *Entity, targetId int64, dmg int64) {
	if targetId == 0 {
		return
	}
	if ent.ByTarget == nil {
		ent.ByTarget = make(map[int64]int64, 8)
	}
	ent.ByTarget[targetId] += dmg
}

// recordCast increments the cast count for a spell on an entity. Caller
// must hold store.mu. The CastFinished event is the most reliable trigger
// since it implies the cast actually completed (not interrupted).
//
// Bumps BOTH the per-fight bucket (BySpell) and the session-level bucket
// (BySpellSession) so the drill-in can show both views.
func recordCast(ent *Entity, spellIdx int) {
	if ent.BySpell == nil {
		ent.BySpell = make(map[int]*SpellTotals, 8)
	}
	s, ok := ent.BySpell[spellIdx]
	if !ok {
		s = &SpellTotals{}
		ent.BySpell[spellIdx] = s
	}
	s.Casts++
	if ent.BySpellSession == nil {
		ent.BySpellSession = make(map[int]*SpellTotals, 8)
	}
	ss, ok := ent.BySpellSession[spellIdx]
	if !ok {
		ss = &SpellTotals{}
		ent.BySpellSession[spellIdx] = ss
	}
	ss.Casts++
}

// recordSpellSession mirrors recordSpell but into the session-level map
// (BySpellSession), which doesn't reset between fights — only on
// ResetSession. Caller must hold store.mu.
func recordSpellSession(ent *Entity, spellIdx int, dmg int64) {
	if ent.BySpellSession == nil {
		ent.BySpellSession = make(map[int]*SpellTotals, 8)
	}
	s, ok := ent.BySpellSession[spellIdx]
	if !ok {
		s = &SpellTotals{}
		ent.BySpellSession[spellIdx] = s
	}
	s.TotalDamage += dmg
	s.Hits++
	if dmg > s.MaxHit {
		s.MaxHit = dmg
	}
}

// recordSpell increments the per-spell totals for an entity. Caller must
// hold store.mu.
func recordSpell(ent *Entity, spellIdx int, dmg int64) {
	if ent.BySpell == nil {
		ent.BySpell = make(map[int]*SpellTotals, 8)
	}
	s, ok := ent.BySpell[spellIdx]
	if !ok {
		s = &SpellTotals{}
		ent.BySpell[spellIdx] = s
	}
	s.TotalDamage += dmg
	s.Hits++
	if dmg > s.MaxHit {
		s.MaxHit = dmg
	}
}

// localInvolved reports whether the local player participated in this
// damage / heal event as either the causer or the affected entity.
// Used to gate touchCombat so the fight counter only ticks when the
// user is actually in combat, not when distant mobs fight each other
// in render range.
func (e *Engine) localInvolved(causer, affected *Entity) bool {
	if causer != nil && causer.IsLocal {
		return true
	}
	if affected != nil && affected.IsLocal {
		return true
	}
	return false
}

// touchCombat updates lastDamageAt and, if we were idle long enough, ends
// the prior fight and starts a new one — archiving the prior fight,
// bumping FightNumber, and zeroing every entity's Current bucket. Called
// from handleHealthUpdate before any stat accumulation.
func (e *Engine) touchCombat(now time.Time) {
	e.fightMu.Lock()
	defer e.fightMu.Unlock()
	if !e.inCombat || now.Sub(e.lastDamageAt) > fightAutoEnd {
		// If a previous fight existed, archive it before zeroing Current.
		if e.fightNumber > 0 && !e.fightStart.IsZero() {
			e.archiveCurrentFight(e.lastDamageAt)
		}
		e.fightNumber++
		e.fightStart = now
		e.inCombat = true
		e.resetAllCurrent()
	}
	e.lastDamageAt = now
}

// archiveCurrentFight snapshots the in-scope entities' Current stats
// into a FightArchive and appends it to fightHistory. Caller must hold
// fightMu. Uses the same scopedMembers() set as the live snapshot so a
// past fight shows exactly the players the live meter showed during it
// — a dungeon run archives only the party, a ZvZ archives everyone.
// Skips entities with zero activity to keep archives lean.
func (e *Engine) archiveCurrentFight(endedAt time.Time) {
	arch := FightArchive{
		Number:     e.fightNumber,
		StartedAt:  e.fightStart,
		EndedAt:    endedAt,
		DurationMs: endedAt.Sub(e.fightStart).Milliseconds(),
	}
	members := e.scopedMembers()
	e.store.mu.RLock()
	for _, ent := range members {
		if ent.Current.DamageDealt == 0 && ent.Current.HealDone == 0 && ent.Current.DamageTaken == 0 {
			continue
		}
		arch.Players = append(arch.Players, FightPlayerArchive{
			UserGuid:  ent.UserGuid.String(),
			Name:      ent.Name,
			ClassCode: ent.ClassCode,
			Role:      ent.Role,
			RoleLabel: ent.RoleLabel,
			ItemPower: ent.ItemPower,
			IsLocal:   ent.IsLocal,
			Damage:    ent.Current.DamageDealt,
			DPS:       ent.Current.DPS(),
			Heal:      ent.Current.HealDone,
			HPS:       ent.Current.HPS(),
			Overheal:  ent.Current.Overhealing,
			Taken:     ent.Current.DamageTaken,
			Deaths:    ent.Deaths,
			Spells:    e.topSpellsLocked(ent, 10),
		})
	}
	e.store.mu.RUnlock()

	if len(arch.Players) == 0 {
		return
	}
	e.fightHistory = append(e.fightHistory, arch)
	if len(e.fightHistory) > maxFightHistory {
		e.fightHistory = e.fightHistory[len(e.fightHistory)-maxFightHistory:]
	}
}

// topSpellsLocked is topSpells without locking — for callers that already
// hold (or have a read-locked) store.mu.
func (e *Engine) topSpellsLocked(ent *Entity, n int) []SpellBreakdown {
	return e.topSpells(ent, n) // topSpells doesn't lock either; safe.
}

// resetAllCurrent zeroes the per-fight stats for every tracked entity.
// Current snapshots into LastFight first so the meter can fall back
// to the previous fight's values until the new fight produces damage,
// avoiding the jarring "everyone drops to zero" frame on each fight
// boundary. Overall persists for the session. Per-spell + per-target
// breakdowns are also reset here so the drill-in screen shows
// abilities and
// targets used in *this* fight.
func (e *Engine) resetAllCurrent() {
	e.store.mu.Lock()
	defer e.store.mu.Unlock()
	for _, ent := range e.store.byGuid {
		if ent.Current.HasActivity() {
			ent.LastFight = ent.Current
		}
		ent.Current.Reset()
		ent.BySpell = nil
		ent.ByTarget = nil
	}
}

// FightStatus returns a snapshot of the current combat-lifecycle state.
func (e *Engine) FightStatus(now time.Time) (number int, elapsed time.Duration, inCombat bool) {
	e.fightMu.Lock()
	defer e.fightMu.Unlock()
	stillIn := e.inCombat && now.Sub(e.lastDamageAt) <= combatEnterIdle
	if e.inCombat && !stillIn {
		// Lazy transition: fight auto-ends on read once the idle gap is
		// past combatEnterIdle. We don't bump fight number here — the
		// next damage tick handles that.
		e.inCombat = false
	}
	if e.fightStart.IsZero() {
		return e.fightNumber, 0, stillIn
	}
	return e.fightNumber, now.Sub(e.fightStart), stillIn
}

// handleCastFinished bumps the cast counter for a spell on the caster's
// entity. CastFinished fires once per completed cast (interrupts don't
// fire). Param 0 = caster ObjectId, param 2 = spell index (per SAT).
//
// Also adds the cast to recentCasts so a debuff that appears within the
// next ~2 seconds can be attributed back to this caster.
//
// If the caster is the local player and we haven't classified them yet,
// try to infer the weapon class from the spell name — CharacterEquipment-
// Changed doesn't always fire for the local user, but their spell IDs
// usually carry "CROSSBOW", "FROSTSTAFF", etc. in the uniquename.
func (e *Engine) handleCastFinished(p map[byte]any) {
	caster, _ := paramLong(p, 0)
	idx, _ := paramLong(p, 2)
	if caster == 0 {
		return
	}
	ent := e.store.ByObjectId(caster)
	if ent == nil {
		return
	}
	e.store.mu.Lock()
	recordCast(ent, int(idx))
	needsInfer := ent.IsLocal && ent.ClassCode == ""
	e.store.mu.Unlock()
	if needsInfer && e.spells != nil {
		if name := e.spells.Name(int(idx)); name != "" {
			if c, ok := gamedata.InferClassFromSpell(name); ok {
				e.store.mu.Lock()
				ent.ClassCode = c.Code
				ent.Role = string(c.Role)
				ent.RoleLabel = c.Label
				e.store.mu.Unlock()
			}
		}
	}
	e.rememberCast(caster, int(idx), e.now())
}

// rememberCast appends a CastFinished event to the recent-casts ring buffer.
func (e *Engine) rememberCast(casterId int64, spellIdx int, t time.Time) {
	e.assistMu.Lock()
	defer e.assistMu.Unlock()
	e.recentCasts = append(e.recentCasts, recentCast{SpellIdx: spellIdx, CasterId: casterId, At: t})
	if len(e.recentCasts) > e.recentCastsCap {
		e.recentCasts = e.recentCasts[len(e.recentCasts)-e.recentCastsCap:]
	}
}

// findRecentCaster returns the most recent CastFinished caster for a spell
// index, within the window. 0 if nothing matches. Caller must hold assistMu.
func (e *Engine) findRecentCasterLocked(spellIdx int, since time.Time) int64 {
	for i := len(e.recentCasts) - 1; i >= 0; i-- {
		c := e.recentCasts[i]
		if c.At.Before(since) {
			break
		}
		if c.SpellIdx == spellIdx {
			return c.CasterId
		}
	}
	return 0
}

// handleActiveSpellEffects records the set of active buff/debuff spell
// indices on a target. Param 0 = ObjectId, param 1 = short[] of spell
// indices currently active.
//
// On each event we diff the new set against the prior set:
//   - spells newly appearing → open a debuff window, infer caster via
//     recentCasts, start tracking damage that lands on this target.
//   - spells newly missing → close the window, accrue uptime + windowed
//     damage into the caster's AssistsBySpell totals.
func (e *Engine) handleActiveSpellEffects(p map[byte]any) {
	id, _ := paramLong(p, 0)
	if id == 0 {
		return
	}
	raw, ok := p[1]
	if !ok {
		return
	}
	effects := readIntArray(raw)
	now := e.now()

	// Update the entity's published ActiveEffects (so the snapshot can
	// show "what's on you right now") regardless of whether we have an
	// entity for the target — buff display only renders for known ones.
	if ent := e.store.ByObjectId(id); ent != nil {
		e.store.mu.Lock()
		ent.ActiveEffects = effects
		e.store.mu.Unlock()
	}

	// Diff and update windows.
	e.diffEffects(id, effects, now)
}

// diffEffects opens windows for newly-present effects and closes windows
// for newly-absent effects, given the latest effect set.
func (e *Engine) diffEffects(targetId int64, newEffects []int, now time.Time) {
	newSet := make(map[int]struct{}, len(newEffects))
	for _, s := range newEffects {
		newSet[s] = struct{}{}
	}

	e.assistMu.Lock()
	defer e.assistMu.Unlock()
	windows := e.activeWindows[targetId]
	if windows == nil {
		windows = make(map[int]*debuffWindow)
		e.activeWindows[targetId] = windows
	}

	// Close windows that are no longer in the active set.
	for spellIdx, w := range windows {
		if _, still := newSet[spellIdx]; !still {
			e.closeWindowLocked(w, now)
			delete(windows, spellIdx)
		}
	}

	// Open windows for newly-active effects.
	for spellIdx := range newSet {
		if _, already := windows[spellIdx]; already {
			continue
		}
		caster := e.findRecentCasterLocked(spellIdx, now.Add(-2*time.Second))
		windows[spellIdx] = &debuffWindow{
			SpellIdx:       spellIdx,
			CasterObjectId: caster, // 0 if we couldn't attribute
			Started:        now,
		}
	}
}

// closeWindowLocked finalises a debuff window, accumulating uptime + windowed
// damage into the caster's AssistsBySpell. assistMu must be held.
func (e *Engine) closeWindowLocked(w *debuffWindow, now time.Time) {
	if w == nil || w.CasterObjectId == 0 {
		return
	}
	ent := e.store.ByObjectId(w.CasterObjectId)
	if ent == nil {
		return
	}
	e.store.mu.Lock()
	if ent.AssistsBySpell == nil {
		ent.AssistsBySpell = make(map[int]*AssistTotals, 8)
	}
	a, ok := ent.AssistsBySpell[w.SpellIdx]
	if !ok {
		a = &AssistTotals{}
		ent.AssistsBySpell[w.SpellIdx] = a
	}
	a.UptimeMs += now.Sub(w.Started).Milliseconds()
	a.DamageDuring += w.DamageDuring
	e.store.mu.Unlock()
}

// chargeWindowsLocked attributes a damage hit on targetId to all currently
// open debuff windows on that target. assistMu must be held by the caller.
func (e *Engine) chargeWindowsLocked(targetId int64, dmg int64) {
	windows := e.activeWindows[targetId]
	if windows == nil {
		return
	}
	for _, w := range windows {
		w.DamageDuring += dmg
	}
}

// readIntArray flattens a Protocol18 numeric array into []int.
func readIntArray(v any) []int {
	switch a := v.(type) {
	case []int16:
		out := make([]int, len(a))
		for i, x := range a {
			out[i] = int(x)
		}
		return out
	case []int32:
		out := make([]int, len(a))
		for i, x := range a {
			out[i] = int(x)
		}
		return out
	case []int64:
		out := make([]int, len(a))
		for i, x := range a {
			out[i] = int(x)
		}
		return out
	case []byte:
		out := make([]int, len(a))
		for i, x := range a {
			out[i] = int(x)
		}
		return out
	case []uint16:
		out := make([]int, len(a))
		for i, x := range a {
			out[i] = int(x)
		}
		return out
	}
	return nil
}

// handleNewMob fires when Albion sends a new mob into the player's
// visible range. Param 0 = ObjectId, param 1 = MobIndex (post-July-2025
// schema requires subtracting 15 — handled in MobCatalog.Lookup).
// Cache the resolved English name keyed by ObjectId so the drill-in's
// Targets tab can render "Fox" instead of "#8087".
func (e *Engine) handleNewMob(p map[byte]any) {
	if e.mobs == nil {
		return
	}
	objectId, _ := paramLong(p, 0)
	mobIdx, _ := paramLong(p, 1)
	if objectId == 0 || mobIdx <= 0 {
		return
	}
	name := e.mobs.Name(int(mobIdx), e.loc)
	if name == "" {
		return
	}
	e.mobNamesMu.Lock()
	e.mobNames[objectId] = name
	e.mobNamesMu.Unlock()
}

// handleOtherGrabbedLoot fires when a party/visible player grabs an item
// or silver pile from a corpse / chest. Recorded into the loot log with
// the looter's name + item index + quantity + zone + dungeon-id tags so
// the Loot panel can roll it up per-looter.
//
// Param 1 = lootedFromName (corpse owner / mob)
// Param 2 = looterByName (the friend / you)
// Param 3 = isSilver (bool)
// Param 4 = itemIndex (items.bin index, 0 when isSilver)
// Param 5 = quantity (stack size or silver amount in copper)
func (e *Engine) handleOtherGrabbedLoot(p map[byte]any) {
	lootedFrom, _ := paramString(p, 1)
	looter, _ := paramString(p, 2)
	isSilver, _ := paramBool(p, 3)
	itemIdx, _ := paramLong(p, 4)
	qty, _ := paramLong(p, 5)
	if looter == "" || qty <= 0 {
		return
	}
	entry := LootEntry{
		At:         e.now(),
		Looter:     looter,
		LootedFrom: prettifyLootSource(lootedFrom),
		IsSilver:   isSilver,
		Quantity:   int(qty),
		Zone:       e.Zone(),
	}
	if d := e.CurrentDungeon(); d != nil {
		entry.DungeonId = d.Id
	}
	if !isSilver && itemIdx > 0 && e.items != nil {
		entry.ItemIndex = int(itemIdx)
		entry.UniqueName = e.items.Name(int(itemIdx))
		if e.loc != nil {
			if name := e.loc.ItemName(entry.UniqueName); name != "" {
				entry.DisplayName = name
			}
		}
		if entry.DisplayName == "" {
			entry.DisplayName = entry.UniqueName
		}
	}
	// Mark as local if this matches the local player's name.
	if local := e.store.localGuidEntity(); local != nil && local.Name == looter {
		entry.LooterIsLocal = true
	}
	// Credit silver-pile pickups by the local player into the session
	// silver tracker. TakeSilver is the canonical source for chest /
	// dungeon yields, but mob-loot silver piles on the open world ship
	// only OtherGrabbedLoot — without this credit the header reads 0
	// across long farming runs. Classify as mob silver when the raw
	// source key names a mob (@MOB_…); funnels through creditSilver so
	// the mob subtotal stays consistent with TakeSilver.
	if entry.IsSilver && entry.LooterIsLocal {
		isMob := strings.Contains(strings.ToUpper(lootedFrom), "MOB")
		e.creditSilver(int64(qty), isMob)
	}
	e.noteLoot(entry)
}

// paramBool reads a boolean param. Returns (false, false) on miss.
func paramBool(p map[byte]any, key byte) (bool, bool) {
	v, ok := p[key]
	if !ok {
		return false, false
	}
	switch x := v.(type) {
	case bool:
		return x, true
	case byte:
		return x != 0, true
	case int16:
		return x != 0, true
	case int32:
		return x != 0, true
	}
	return false, false
}

// handleEquipmentChanged updates a tracked entity's class chip + role +
// IP when they swap weapons. Critical for the local player — Albion fires
// equipment events for the local user BEFORE the Join response, so we
// cache the full 10-slot equipment + qualities array by ObjectId and
// replay when the entity registers.
//
// Param 0 = ObjectId, param 2 = short[] equipment array; on NewCharacter
// the equipment lives at param 40 instead.
func (e *Engine) handleEquipmentChanged(p map[byte]any) {
	objectId, ok := paramLong(p, 0)
	if !ok || objectId == 0 {
		return
	}
	equip, qualities, spells, found := parseEquipmentParams(p)
	if found {
		e.cachePendingEquip(objectId, equip, qualities, spells)
	}
	ent := e.store.ByObjectId(objectId)
	if ent == nil {
		return
	}
	e.applyEquipment(ent, p)
}

// parseEquipmentParams reads the equipment + quality + spells arrays
// from either NewCharacter (param 40 + 41/42 + 43) or
// CharacterEquipmentChanged (param 2 + 3 + 7). Returns the parsed
// values plus an `ok` flag (false when no equipment array is present).
//
// Quality is the 1..5 byte; if the event doesn't ship a quality array
// we default to 0 (treated as Normal=1 downstream).
func parseEquipmentParams(p map[byte]any) ([10]int, [10]int, [14]int, bool) {
	var equip, qualities [10]int
	var spells [14]int
	rawEquip, ok := p[40]
	if !ok {
		rawEquip, ok = p[2]
	}
	if !ok {
		return equip, qualities, spells, false
	}
	for i, v := range intsOfArray(rawEquip, 10) {
		equip[i] = v
	}
	// Quality lives at param 41 (NewCharacter) or 3
	// (CharacterEquipmentChanged) in SAT — try both. May not be present
	// on older patches; we silently default to 0.
	if rawQ, ok := p[41]; ok {
		for i, v := range intsOfArray(rawQ, 10) {
			qualities[i] = v
		}
	} else if rawQ, ok := p[3]; ok {
		for i, v := range intsOfArray(rawQ, 10) {
			qualities[i] = v
		}
	}
	// Active spells lives at param 7 in CharacterEquipmentChanged. SAT
	// reads index 0/1/2 as MainHand slots, 3 Armor, 4 Head, 5 Shoes,
	// 12 Potion, 13 Food. Empty entries are -1 — we store them as-is.
	if rawS, ok := p[7]; ok {
		for i, v := range intsOfArray(rawS, 14) {
			spells[i] = v
		}
	}
	// Verbose: dump every param key + type + the three parsed arrays so
	// we can pin down which byte Albion is shipping quality in on the
	// current patch. Grep "parseEquipmentParams" in the verbose log.
	if verbose {
		keys := make([]string, 0, len(p))
		for k, v := range p {
			keys = append(keys, fmt.Sprintf("%d=%T", k, v))
		}
		log.Printf("parseEquipmentParams keys=[%s]", strings.Join(keys, " "))
		log.Printf("parseEquipmentParams equip=%v", equip)
		log.Printf("parseEquipmentParams qualities=%v", qualities)
		log.Printf("parseEquipmentParams spells=%v", spells)
	}
	return equip, qualities, spells, true
}

// cachePendingEquip stashes the full equipment+qualities+spells for an
// ObjectId so a later UpsertByGuid can pick it up. Safe for concurrent use.
func (e *Engine) cachePendingEquip(objectId int64, equip [10]int, qualities [10]int, spells [14]int) {
	e.pendingEquipMu.Lock()
	e.pendingEquip[objectId] = pendingEquipEntry{Equipment: equip, Qualities: qualities, ActiveSpells: spells}
	e.pendingEquipMu.Unlock()
}

// takePendingEquip returns and clears the cached entry for an ObjectId.
func (e *Engine) takePendingEquip(objectId int64) (pendingEquipEntry, bool) {
	e.pendingEquipMu.Lock()
	defer e.pendingEquipMu.Unlock()
	entry, ok := e.pendingEquip[objectId]
	if ok {
		delete(e.pendingEquip, objectId)
	}
	return entry, ok
}

// applyCachedEquipment classifies an entity using the cached equipment
// array (if any). Called from handleJoinResponse after the local player's
// entity is finally registered, and from handleNewCharacter for race
// symmetry with non-local players.
func (e *Engine) applyCachedEquipment(ent *Entity) {
	if ent == nil || e.items == nil || ent.ObjectId == 0 {
		return
	}
	entry, ok := e.takePendingEquip(ent.ObjectId)
	if !ok {
		return
	}
	e.applyEquipmentArray(ent, entry.Equipment, entry.Qualities, entry.ActiveSpells)
}

func (e *Engine) handleNewCharacter(p map[byte]any) {
	objectId, _ := paramLong(p, 0)
	name, _ := paramString(p, 1)
	guid, _ := paramGuid(p, 7)
	guild, _ := paramString(p, 8)
	if guid.IsZero() {
		return
	}
	ent := e.store.UpsertByGuid(guid, objectId, name, guild)
	e.applyEquipment(ent, p)
	// Same race as the local Join response — a CharacterEquipmentChanged
	// event for this player may have arrived before NewCharacter. Replay.
	e.applyCachedEquipment(ent)
	// Param 22 carries MaxHealth (the entity's HP cap at spawn). Capturing
	// it here is the only reliable way to compute overheal later, since
	// HealthUpdate only tells us the new HP after the change, not the cap.
	if mh, ok := paramLong(p, 22); ok && mh > 0 {
		e.store.mu.Lock()
		ent.MaxHealth = mh
		e.store.mu.Unlock()
	}
}

// applyEquipment reads the equipment + quality + spells arrays from the
// event, classifies the player's class from the MainHand slot, computes
// average IP across the core slots, and writes everything onto the
// entity.
func (e *Engine) applyEquipment(ent *Entity, p map[byte]any) {
	if ent == nil || e.items == nil {
		return
	}
	equip, qualities, spells, ok := parseEquipmentParams(p)
	if !ok {
		return
	}
	e.applyEquipmentArray(ent, equip, qualities, spells)
}

// applyEquipmentArray is the inner write — used both by live equipment
// events and by the cached-equipment replay path.
func (e *Engine) applyEquipmentArray(ent *Entity, equip [10]int, qualities [10]int, spells [14]int) {
	if ent == nil || e.items == nil {
		return
	}
	// Class chip + role + label come from MainHand.
	if mh := equip[0]; mh > 0 {
		e.classifyMainHand(ent, mh)
	}
	// IP across core slots — MainHand, OffHand, Head, Chest, Shoes, Cape.
	// AverageItemPower handles the 2H "occupies both hands" rule.
	ip := gamedata.AverageItemPower(e.items, equip, qualities)
	e.store.mu.Lock()
	ent.Equipment = equip
	ent.Qualities = qualities
	ent.ActiveSpells = spells
	ent.ItemPower = ip
	e.store.mu.Unlock()
}

// classifyMainHand resolves a MainHand item index to the entity's class
// chip + role + label. The chip + role come from pattern-matching the
// uniquename ("DUALCROSSBOW" → XBW / RangedDPS). The label PREFERS the
// localized in-game name ("Adept's Arclight Blasters") when available,
// falling back to ClassifyWeapon's generic label when not.
func (e *Engine) classifyMainHand(ent *Entity, mainHand int) {
	name := e.items.Name(mainHand)
	if name == "" {
		return
	}
	c := gamedata.ClassifyWeapon(name)
	label := c.Label
	if e.loc != nil {
		if loc := e.loc.ItemName(name); loc != "" {
			label = strings.ToUpper(loc)
		}
	}
	e.store.mu.Lock()
	ent.MainHandItemId = mainHand
	ent.ClassCode = c.Code
	ent.Role = string(c.Role)
	ent.RoleLabel = label
	e.store.mu.Unlock()
}

// intsOfArray flattens up to n leading integer elements of a Protocol18
// numeric array into a Go []int. Used to read 10-slot equipment +
// quality arrays. Returns nil when the value isn't a recognised slice
// shape; caller treats nil as "empty".
func intsOfArray(v any, n int) []int {
	out := make([]int, 0, n)
	take := func(i int) {
		if len(out) < n {
			out = append(out, i)
		}
	}
	switch a := v.(type) {
	case []int16:
		for _, x := range a {
			take(int(x))
		}
	case []int32:
		for _, x := range a {
			take(int(x))
		}
	case []int64:
		for _, x := range a {
			take(int(x))
		}
	case []uint16:
		for _, x := range a {
			take(int(x))
		}
	case []uint32:
		for _, x := range a {
			take(int(x))
		}
	case []byte:
		for _, x := range a {
			take(int(x))
		}
	case []any:
		for _, x := range a {
			switch xx := x.(type) {
			case byte:
				take(int(xx))
			case int16:
				take(int(xx))
			case int32:
				take(int(xx))
			case int64:
				take(int(xx))
			case uint16:
				take(int(xx))
			case uint32:
				take(int(xx))
			}
		}
	}
	return out
}

// firstIntOfArray returns the first integer element of a Protocol18 array
// parameter (int16/int32/int64 / their unsigned cousins / any). 0 on miss.
// Kept for backwards compatibility — callers should prefer intsOfArray.
func firstIntOfArray(v any) int {
	switch a := v.(type) {
	case []int16:
		if len(a) > 0 {
			return int(a[0])
		}
	case []int32:
		if len(a) > 0 {
			return int(a[0])
		}
	case []int64:
		if len(a) > 0 {
			return int(a[0])
		}
	case []uint16:
		if len(a) > 0 {
			return int(a[0])
		}
	case []byte:
		if len(a) > 0 {
			return int(a[0])
		}
	case []any:
		if len(a) > 0 {
			switch x := a[0].(type) {
			case byte:
				return int(x)
			case int16:
				return int(x)
			case int32:
				return int(x)
			case int64:
				return int(x)
			case uint16:
				return int(x)
			case uint32:
				return int(x)
			}
		}
	}
	return 0
}

func (e *Engine) handlePartyJoined(p map[byte]any) {
	guids, _ := paramGuidsFromByteArray(p, 5)
	names, _ := paramStringArray(p, 6)
	e.store.ResetParty()
	n := len(guids)
	if len(names) < n {
		n = len(names)
	}
	for i := 0; i < n; i++ {
		e.store.UpsertByGuid(guids[i], 0, names[i], "")
		e.store.MarkInParty(guids[i], true)
	}
	e.persistParty()
}

func (e *Engine) handlePartyPlayerJoined(p map[byte]any) {
	guid, _ := paramGuid(p, 1)
	name, _ := paramString(p, 2)
	if guid.IsZero() {
		return
	}
	e.store.UpsertByGuid(guid, 0, name, "")
	e.store.MarkInParty(guid, true)
	e.persistParty()
}

func (e *Engine) handlePartyPlayerLeft(p map[byte]any) {
	guid, _ := paramGuid(p, 1)
	if guid.IsZero() {
		return
	}
	e.store.MarkInParty(guid, false)
	e.persistParty()
}

func (e *Engine) handleMountStart(p map[byte]any) {
	rider, ok := paramLong(p, 0)
	if !ok || rider == 0 {
		return
	}
	e.store.StashMountStart(rider)
}

func (e *Engine) handleNewMountObject(p map[byte]any) {
	guid, ok := paramGuid(p, 5)
	if !ok || guid.IsZero() {
		return
	}
	e.store.BindRiderGuid(guid)
}

func (e *Engine) handleJoinFinished(p map[byte]any) {
	guid, ok := paramGuid(p, 0)
	if !ok || guid.IsZero() {
		// Try alternate parameter slots — JoinFinished's layout differs
		// across server versions. Fall through silently.
		return
	}
	e.store.SetLocalGuid(guid)
}

// realCode mirrors the cmd/agent helper: prefer parameter 252 (events) or
// 253 (operations) for the application-level code, falling back to the
// transport-level byte.
func realCode(p map[byte]any, fallback byte) int {
	for _, k := range []byte{252, 253} {
		if v, ok := p[k]; ok {
			switch x := v.(type) {
			case byte:
				return int(x)
			case int16:
				return int(x)
			case int32:
				return int(x)
			case int64:
				return int(x)
			case uint16:
				return int(x)
			case uint32:
				return int(x)
			case uint64:
				return int(x)
			}
		}
	}
	return int(fallback)
}
