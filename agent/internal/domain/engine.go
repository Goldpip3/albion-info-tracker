package domain

import (
	"log"
	"os"
	"sync"
	"time"

	"github.com/Goldpip3/albion-info-tracker/agent/internal/gamecodes"
	"github.com/Goldpip3/albion-info-tracker/agent/internal/gamedata"
	"github.com/Goldpip3/albion-info-tracker/agent/internal/photon"
)

// verbose toggles per-event debug logging. Set ALBION_AGENT_VERBOSE=1 to
// enable; useful for figuring out why party detection isn't kicking in.
var verbose = os.Getenv("ALBION_AGENT_VERBOSE") != ""

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

// NewEngine constructs an Engine backed by a fresh Store.
func NewEngine() *Engine {
	now := time.Now()
	return &Engine{
		store:   NewStore(),
		now:     time.Now,
		events:  newEventBuffer(),
		session: SessionStats{Start: now},
	}
}

// ResetSession clears every running counter — combat stats, per-spell
// breakdowns, deaths, session economy, fight counter, fight history,
// activity log — and stamps a fresh session start time. Called when the
// user clicks "New session" in the UI.
func (e *Engine) ResetSession() {
	now := e.now()
	e.store.mu.Lock()
	for _, ent := range e.store.byGuid {
		ent.Current.Reset()
		ent.Overall.Reset()
		ent.BySpell = nil
		ent.Deaths = 0
	}
	e.store.mu.Unlock()

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
// Worker over the same WebSocket). Currently only "resetSession" is
// recognised; future actions go here too.
func (e *Engine) HandleCommand(action string) {
	switch action {
	case "resetSession":
		e.ResetSession()
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

func (e *Engine) onEvent(ev photon.EventData) {
	code := realCode(ev.Parameters, ev.Code)
	switch gamecodes.Event(code) {
	case gamecodes.EventHealthUpdate:
		e.handleHealthUpdate(ev.Parameters)
	case gamecodes.EventNewCharacter:
		dbg("NewCharacter %v", ev.Parameters)
		e.handleNewCharacter(ev.Parameters)
	case gamecodes.EventCharacterEquipmentChanged:
		dbg("CharacterEquipmentChanged %v", ev.Parameters)
		e.handleEquipmentChanged(ev.Parameters)
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
	case gamecodes.EventUpdateMoney:
		dbg("UpdateMoney %v", ev.Parameters)
		e.handleUpdateMoney(ev.Parameters)
	case gamecodes.EventUpdateReSpecPoints:
		dbg("UpdateReSpec %v", ev.Parameters)
		e.handleUpdateReSpec(ev.Parameters)
	case gamecodes.EventMightAndFavorReceivedEvent:
		dbg("MightAndFavor %v", ev.Parameters)
		e.handleMightAndFavor(ev.Parameters)
	case gamecodes.EventTakeSilver:
		dbg("TakeSilver %v", ev.Parameters)
		// covered by UpdateMoney delta; ignore to avoid double counting
	}
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
//   0 → UserObjectId, 1 → UserGuid, 2 → Username, 58 → GuildName.
func (e *Engine) handleJoinResponse(p map[byte]any) {
	objectId, _ := paramLong(p, 0)
	guid, _ := paramGuid(p, 1)
	name, _ := paramString(p, 2)
	guild, _ := paramString(p, 58)
	if guid.IsZero() {
		return
	}
	e.store.UpsertByGuid(guid, objectId, name, guild)
	e.store.SetLocalGuid(guid)
	// Local player is always in their own party for damage-meter purposes,
	// even when actually solo — that matches SAT's behaviour.
	e.store.MarkInParty(guid, true)
}

func (e *Engine) handleHealthUpdate(p map[byte]any) {
	affected, _ := paramLong(p, 0)
	change, _ := paramDouble(p, 2)
	causer, _ := paramLong(p, 6)
	spellIdx, _ := paramLong(p, 7)
	if causer == 0 {
		return
	}

	now := e.now()

	// Combat / fight bookkeeping. Has to happen before stats accumulate so
	// a brand-new fight zeroes the Current bucket first.
	e.touchCombat(now)

	causerEnt := e.store.ByObjectId(causer)
	affEnt := e.store.ByObjectId(affected)

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
			e.store.mu.Unlock()
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
		newHP := int64(0)
		if v, ok := paramDouble(p, 3); ok {
			newHP = int64(v + 0.5)
		}
		effective, overheal := splitOverheal(affEnt, heal, newHP)
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

// handleUpdateFame folds a fame-change event into the session total. The
// event carries the LIFETIME total (param 1) so we track delta-to-previous.
func (e *Engine) handleUpdateFame(p map[byte]any) {
	total, ok := paramLong(p, 1)
	if !ok {
		return
	}
	e.sessionMu.Lock()
	e.session.AccumulateFame(total)
	e.sessionMu.Unlock()
}

// handleUpdateMoney folds a wallet-change event into session silver gained.
// Param 1 = CurrentPlayerSilver (lifetime / wallet total in FixPoint
// internal units; 10_000 = 1 silver). Only positive deltas count.
func (e *Engine) handleUpdateMoney(p map[byte]any) {
	total, ok := paramLong(p, 1)
	if !ok {
		return
	}
	e.sessionMu.Lock()
	e.session.AccumulateSilver(total)
	e.sessionMu.Unlock()
}

// handleUpdateReSpec adds the gained respec credits delta (param 2) to the
// session total. Param 2 is FixPoint internal units (10_000 = 1 credit).
func (e *Engine) handleUpdateReSpec(p map[byte]any) {
	gained, ok := paramLong(p, 2)
	if !ok || gained <= 0 {
		return
	}
	e.sessionMu.Lock()
	e.session.AccumulateRespec(gained)
	e.sessionMu.Unlock()
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

// archiveCurrentFight snapshots every entity's Current stats into a
// FightArchive and appends it to fightHistory. Caller must hold fightMu.
// Skips entities with zero activity to keep archives lean.
func (e *Engine) archiveCurrentFight(endedAt time.Time) {
	arch := FightArchive{
		Number:     e.fightNumber,
		StartedAt:  e.fightStart,
		EndedAt:    endedAt,
		DurationMs: endedAt.Sub(e.fightStart).Milliseconds(),
	}
	e.store.mu.RLock()
	for _, ent := range e.store.byGuid {
		if ent.Current.DamageDealt == 0 && ent.Current.HealDone == 0 && ent.Current.DamageTaken == 0 {
			continue
		}
		arch.Players = append(arch.Players, FightPlayerArchive{
			UserGuid:  ent.UserGuid.String(),
			Name:      ent.Name,
			ClassCode: ent.ClassCode,
			Role:      ent.Role,
			RoleLabel: ent.RoleLabel,
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
// Overall persists for the session. Per-spell breakdown is also reset
// here so the drill-in screen shows abilities used in *this* fight.
func (e *Engine) resetAllCurrent() {
	e.store.mu.Lock()
	defer e.store.mu.Unlock()
	for _, ent := range e.store.byGuid {
		ent.Current.Reset()
		ent.BySpell = nil
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

// handleEquipmentChanged updates a tracked entity's class chip + role when
// they swap weapons. Critical for the local player — Albion never
// broadcasts the local user via NewCharacter, so this is the only path
// that gets your own "MELEE DPS · DAGGERS" subtitle on screen. Param 0 =
// ObjectId, param 2 = short[] equipment array with index 0 = MainHand.
func (e *Engine) handleEquipmentChanged(p map[byte]any) {
	objectId, ok := paramLong(p, 0)
	if !ok || objectId == 0 {
		return
	}
	ent := e.store.ByObjectId(objectId)
	if ent == nil {
		return
	}
	e.applyEquipment(ent, p)
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
	// Param 22 carries MaxHealth (the entity's HP cap at spawn). Capturing
	// it here is the only reliable way to compute overheal later, since
	// HealthUpdate only tells us the new HP after the change, not the cap.
	if mh, ok := paramLong(p, 22); ok && mh > 0 {
		e.store.mu.Lock()
		ent.MaxHealth = mh
		e.store.mu.Unlock()
	}
}

// applyEquipment reads a 10-slot equipment array and uses index 0
// (MainHand) to classify the weapon. NewCharacter ships the array in
// param 40; CharacterEquipmentChanged ships it in param 2.
func (e *Engine) applyEquipment(ent *Entity, p map[byte]any) {
	if ent == nil || e.items == nil {
		return
	}
	equip, ok := p[40]
	if !ok {
		equip, ok = p[2]
	}
	if !ok {
		return
	}
	mainHand := firstIntOfArray(equip)
	if mainHand <= 0 {
		return
	}
	name := e.items.Name(mainHand)
	if name == "" {
		return
	}
	c := gamedata.ClassifyWeapon(name)
	e.store.mu.Lock()
	ent.MainHandItemId = mainHand
	ent.ClassCode = c.Code
	ent.Role = string(c.Role)
	ent.RoleLabel = c.Label
	e.store.mu.Unlock()
}

// firstIntOfArray returns the first integer element of a Protocol18 array
// parameter (int16/int32/int64 / their unsigned cousins / any). 0 on miss.
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
}

func (e *Engine) handlePartyPlayerJoined(p map[byte]any) {
	guid, _ := paramGuid(p, 1)
	name, _ := paramString(p, 2)
	if guid.IsZero() {
		return
	}
	e.store.UpsertByGuid(guid, 0, name, "")
	e.store.MarkInParty(guid, true)
}

func (e *Engine) handlePartyPlayerLeft(p map[byte]any) {
	guid, _ := paramGuid(p, 1)
	if guid.IsZero() {
		return
	}
	e.store.MarkInParty(guid, false)
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
