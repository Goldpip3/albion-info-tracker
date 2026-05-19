package domain

import (
	"github.com/Goldpip3/albion-info-tracker/agent/internal/photon"
)

// paramLong extracts a signed 64-bit integer from a Photon parameter table.
// Accepts any of the numeric variants Protocol18 emits.
func paramLong(p map[byte]any, key byte) (int64, bool) {
	v, ok := p[key]
	if !ok {
		return 0, false
	}
	switch x := v.(type) {
	case byte:
		return int64(x), true
	case int16:
		return int64(x), true
	case uint16:
		return int64(x), true
	case int32:
		return int64(x), true
	case uint32:
		return int64(x), true
	case int64:
		return x, true
	case uint64:
		return int64(x), true
	default:
		return 0, false
	}
}

func paramDouble(p map[byte]any, key byte) (float64, bool) {
	v, ok := p[key]
	if !ok {
		return 0, false
	}
	switch x := v.(type) {
	case float32:
		return float64(x), true
	case float64:
		return x, true
	default:
		if n, ok := paramLong(p, key); ok {
			return float64(n), true
		}
		return 0, false
	}
}

func paramString(p map[byte]any, key byte) (string, bool) {
	v, ok := p[key]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

// paramGuid pulls a 16-byte UserGuid out of a parameter, whether it arrived
// as a CustomType (typeCode 7 by convention) or a raw byte array.
func paramGuid(p map[byte]any, key byte) (Guid, bool) {
	v, ok := p[key]
	if !ok {
		return Guid{}, false
	}
	var bytes []byte
	switch x := v.(type) {
	case photon.CustomType:
		bytes = x.Data
	case []byte:
		bytes = x
	default:
		return Guid{}, false
	}
	if len(bytes) != 16 {
		return Guid{}, false
	}
	var g Guid
	copy(g[:], bytes)
	return g, true
}

// paramStringArray extracts a []string parameter, returning false if absent
// or if the value is not a string array.
func paramStringArray(p map[byte]any, key byte) ([]string, bool) {
	v, ok := p[key]
	if !ok {
		return nil, false
	}
	if a, ok := v.([]string); ok {
		return a, true
	}
	// Fallback: object array of strings (Protocol18 may deliver either form).
	if a, ok := v.([]any); ok {
		out := make([]string, 0, len(a))
		for _, e := range a {
			if s, ok := e.(string); ok {
				out = append(out, s)
			}
		}
		return out, len(out) == len(a)
	}
	return nil, false
}

// paramGuidsFromByteArray extracts a flat 16N-byte buffer into N consecutive
// Guids. Albion's PartyJoined event packs party member GUIDs this way.
func paramGuidsFromByteArray(p map[byte]any, key byte) ([]Guid, bool) {
	v, ok := p[key]
	if !ok {
		return nil, false
	}
	bytes, ok := v.([]byte)
	if !ok {
		return nil, false
	}
	if len(bytes)%16 != 0 {
		return nil, false
	}
	out := make([]Guid, len(bytes)/16)
	for i := range out {
		copy(out[i][:], bytes[i*16:(i+1)*16])
	}
	return out, true
}
