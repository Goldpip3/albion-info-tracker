package domain

import (
	"sync"
	"time"

	"github.com/Goldpip3/albion-info-tracker/agent/internal/gamecodes"
	"github.com/Goldpip3/albion-info-tracker/agent/internal/photon"
)

// Engine wires Photon events into the entity store and combat tracker.
// It is the public surface the rest of the agent talks to.
type Engine struct {
	store *Store

	// guard protects flushedFights and other engine-only fields. The Store
	// has its own lock for entity state.
	guard sync.Mutex
	now   func() time.Time
}

// NewEngine constructs an Engine backed by a fresh Store.
func NewEngine() *Engine {
	return &Engine{store: NewStore(), now: time.Now}
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
		e.handleNewCharacter(ev.Parameters)
	case gamecodes.EventPartyJoined:
		e.handlePartyJoined(ev.Parameters)
	case gamecodes.EventPartyPlayerJoined:
		e.handlePartyPlayerJoined(ev.Parameters)
	case gamecodes.EventPartyPlayerLeft:
		e.handlePartyPlayerLeft(ev.Parameters)
	case gamecodes.EventPartyDisbanded:
		e.store.ResetParty()
	case gamecodes.EventMountStart:
		e.handleMountStart(ev.Parameters)
	case gamecodes.EventNewMountObject:
		e.handleNewMountObject(ev.Parameters)
	case gamecodes.EventJoinFinished:
		e.handleJoinFinished(ev.Parameters)
	}
}

// onRequest and onResponse are stubs for now — damage meter MVP doesn't
// consume operations. Hooks are in place for later phases.
func (e *Engine) onRequest(photon.OperationRequest)   {}
func (e *Engine) onResponse(photon.OperationResponse) {}

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
	e.store.UpsertByGuid(guid, objectId, name, guild)
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
