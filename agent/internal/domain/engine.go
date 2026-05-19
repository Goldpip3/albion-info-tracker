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
}

// NewEngine constructs an Engine backed by a fresh Store.
func NewEngine() *Engine {
	return &Engine{store: NewStore(), now: time.Now}
}

// SetItemCatalog wires an items.bin catalog into the engine for weapon-based
// role classification. Safe to call before or after capture starts.
func (e *Engine) SetItemCatalog(c *gamedata.ItemCatalog) {
	e.items = c
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
	if causer == 0 {
		return
	}

	now := e.now()

	// Damage is delivered as negative HealthChange; heal as positive.
	if change < 0 {
		dmg := int64(-change + 0.5)
		if dmg <= 0 {
			return
		}
		if causerEnt := e.store.ByObjectId(causer); causerEnt != nil {
			e.store.mu.Lock()
			recordDamage(&causerEnt.Current, &causerEnt.Overall, dmg, now)
			e.store.mu.Unlock()
		}
		if affEnt := e.store.ByObjectId(affected); affEnt != nil {
			e.store.mu.Lock()
			recordTakenDamage(&affEnt.Current, &affEnt.Overall, dmg, now)
			e.store.mu.Unlock()
		}
		return
	}
	if change > 0 {
		heal := int64(change + 0.5)
		if heal <= 0 {
			return
		}
		if causerEnt := e.store.ByObjectId(causer); causerEnt != nil {
			e.store.mu.Lock()
			recordHeal(&causerEnt.Current, &causerEnt.Overall, heal, now)
			e.store.mu.Unlock()
		}
	}
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
}

// applyEquipment reads param 40 (the 10-slot equipment array) from a
// NewCharacter event and uses index 0 (MainHand) to classify the weapon.
// Subsequent NewCharacter / CharacterEquipmentChanged events for the same
// player will overwrite as the player re-equips.
func (e *Engine) applyEquipment(ent *Entity, p map[byte]any) {
	if ent == nil || e.items == nil {
		return
	}
	equip, ok := p[40]
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
