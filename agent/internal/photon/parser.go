package photon

import (
	"encoding/binary"
)

const (
	photonHeaderLen  = 12
	commandHeaderLen = 12
)

// Handlers receives the three message kinds the parser produces. Any handler
// may be nil; nil handlers silently drop the matching messages.
type Handlers struct {
	OnEvent    func(EventData)
	OnRequest  func(OperationRequest)
	OnResponse func(OperationResponse)
}

// Parser turns raw Photon UDP datagram payloads into game-level
// events/requests/responses. It is safe to call Receive concurrently from
// multiple goroutines only if you protect the same instance externally.
type Parser struct {
	h Handlers
	// TODO: fragment reassembly state goes here once SendFragment is handled.
}

// New returns a parser that dispatches into h.
func New(h Handlers) *Parser {
	return &Parser{h: h}
}

// Receive parses one Photon datagram payload (already stripped of IP/UDP
// headers). Malformed packets are dropped silently — Photon's framing has
// no inherent integrity check beyond optional CRC, and partial captures are
// expected on a noisy network.
func (p *Parser) Receive(payload []byte) {
	if len(payload) < photonHeaderLen {
		return
	}
	r := &reader{buf: payload}

	if err := r.skip(2); err != nil { // peer id
		return
	}
	flags, err := r.byteAt()
	if err != nil {
		return
	}
	cmdCount, err := r.byteAt()
	if err != nil {
		return
	}
	if err := r.skip(8); err != nil { // sent time (4) + challenge (4)
		return
	}

	if flags == 1 {
		return // encrypted — not supported
	}
	if flags == 0xCC {
		// CRC-protected; verify by zeroing out the CRC field and recomputing.
		if !verifyCRC(payload) {
			return
		}
	}

	for i := byte(0); i < cmdCount; i++ {
		if !p.handleCommand(r) {
			return
		}
	}
}

// handleCommand returns false on a fatal framing error.
func (p *Parser) handleCommand(r *reader) bool {
	cmdType, err := r.byteAt()
	if err != nil {
		return false
	}
	if err := r.skip(3); err != nil { // channel(1) + flags(1) + reserved(1)
		return false
	}
	cmdLen, err := r.beInt32()
	if err != nil {
		return false
	}
	if err := r.skip(4); err != nil { // reliable sequence number
		return false
	}
	bodyLen := int(cmdLen) - commandHeaderLen
	if bodyLen < 0 || bodyLen > r.remaining() {
		return false
	}

	switch commandType(cmdType) {
	case cmdDisconnect:
		return false
	case cmdSendUnreliable:
		// 4-byte unreliable sequence number prefix
		if err := r.skip(4); err != nil {
			return false
		}
		bodyLen -= 4
		return p.handleReliable(r, bodyLen)
	case cmdSendReliable:
		return p.handleReliable(r, bodyLen)
	case cmdSendFragment:
		// TODO: implement fragment reassembly. For now, advance past the body.
		return r.skip(bodyLen) == nil
	default:
		// Unknown command: skip its body and keep going.
		return r.skip(bodyLen) == nil
	}
}

func (p *Parser) handleReliable(r *reader, bodyLen int) bool {
	if bodyLen < 2 {
		return false
	}
	// First byte is a signature/version (always 0xF3 for Albion's Protocol18);
	// second byte is the message type.
	if err := r.skip(1); err != nil {
		return false
	}
	mt, err := r.byteAt()
	if err != nil {
		return false
	}
	bodyLen -= 2

	body, err := r.bytes(bodyLen)
	if err != nil {
		return false
	}
	inner := &reader{buf: body}

	switch messageType(mt) {
	case msgOperationRequest:
		req, err := DeserializeOperationRequest(inner)
		if err != nil {
			return true // skip malformed, keep parsing later commands
		}
		if p.h.OnRequest != nil {
			p.h.OnRequest(req)
		}
	case msgOperationResponse:
		resp, err := DeserializeOperationResponse(inner)
		if err != nil {
			return true
		}
		if p.h.OnResponse != nil {
			p.h.OnResponse(resp)
		}
	case msgEvent:
		evt, err := DeserializeEventData(inner)
		if err != nil {
			return true
		}
		if p.h.OnEvent != nil {
			p.h.OnEvent(evt)
		}
	}
	return true
}

// verifyCRC checks the optional CRC32 over the packet. Bytes 8..11 carry the
// CRC; we zero them out, compute over the full payload, and compare.
func verifyCRC(payload []byte) bool {
	if len(payload) < 12 {
		return false
	}
	stored := binary.BigEndian.Uint32(payload[8:12])
	// Make a working copy with the CRC field zeroed.
	work := make([]byte, len(payload))
	copy(work, payload)
	binary.BigEndian.PutUint32(work[8:12], 0)
	return stored == crc32Photon(work)
}

// crc32Photon implements the bit-reversed CRC-32 variant Photon uses.
// Equivalent to the C# CrcCalculator.
func crc32Photon(b []byte) uint32 {
	const key = 0xEDB88320 // == 3988292384
	result := uint32(0xFFFFFFFF)
	for _, x := range b {
		result ^= uint32(x)
		for j := 0; j < 8; j++ {
			if result&1 != 0 {
				result = (result >> 1) ^ key
			} else {
				result >>= 1
			}
		}
	}
	return result
}
