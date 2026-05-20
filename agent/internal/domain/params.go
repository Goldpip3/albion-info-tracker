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

// paramDoubleArray extracts a numeric parameter as a []float64, tolerating
// every slice type the Protocol18 deserializer can emit for a numeric
// array (float/int variants, or a heterogeneous []any). Mirrors SAT's
// Converter.GetValue<T> flexibility. Returns nil when the key is absent
// or non-numeric. Used by the batched HealthUpdates handler.
func paramDoubleArray(p map[byte]any, key byte) []float64 {
	v, ok := p[key]
	if !ok {
		return nil
	}
	switch x := v.(type) {
	case []float64:
		return x
	case []float32:
		out := make([]float64, len(x))
		for i, e := range x {
			out[i] = float64(e)
		}
		return out
	case []int16:
		out := make([]float64, len(x))
		for i, e := range x {
			out[i] = float64(e)
		}
		return out
	case []int32:
		out := make([]float64, len(x))
		for i, e := range x {
			out[i] = float64(e)
		}
		return out
	case []int64:
		out := make([]float64, len(x))
		for i, e := range x {
			out[i] = float64(e)
		}
		return out
	case []byte:
		out := make([]float64, len(x))
		for i, e := range x {
			out[i] = float64(e)
		}
		return out
	case []any:
		out := make([]float64, len(x))
		for i, e := range x {
			out[i] = anyToFloat(e)
		}
		return out
	default:
		return nil
	}
}

// paramLongArray is the integer analogue of paramDoubleArray.
func paramLongArray(p map[byte]any, key byte) []int64 {
	v, ok := p[key]
	if !ok {
		return nil
	}
	switch x := v.(type) {
	case []int64:
		return x
	case []int32:
		out := make([]int64, len(x))
		for i, e := range x {
			out[i] = int64(e)
		}
		return out
	case []int16:
		out := make([]int64, len(x))
		for i, e := range x {
			out[i] = int64(e)
		}
		return out
	case []uint16:
		out := make([]int64, len(x))
		for i, e := range x {
			out[i] = int64(e)
		}
		return out
	case []uint32:
		out := make([]int64, len(x))
		for i, e := range x {
			out[i] = int64(e)
		}
		return out
	case []byte:
		out := make([]int64, len(x))
		for i, e := range x {
			out[i] = int64(e)
		}
		return out
	case []float32:
		out := make([]int64, len(x))
		for i, e := range x {
			out[i] = int64(e)
		}
		return out
	case []float64:
		out := make([]int64, len(x))
		for i, e := range x {
			out[i] = int64(e)
		}
		return out
	case []any:
		out := make([]int64, len(x))
		for i, e := range x {
			out[i] = int64(anyToFloat(e))
		}
		return out
	default:
		return nil
	}
}

// atDouble / atLong are bounds-safe element accessors — out-of-range
// indices yield 0, matching SAT's `i < count ? arr[i] : 0`.
func atDouble(s []float64, i int) float64 {
	if i < 0 || i >= len(s) {
		return 0
	}
	return s[i]
}

func atLong(s []int64, i int) int64 {
	if i < 0 || i >= len(s) {
		return 0
	}
	return s[i]
}

// anyToFloat coerces a single boxed numeric value to float64 (0 on miss).
func anyToFloat(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case float32:
		return float64(x)
	case int16:
		return float64(x)
	case uint16:
		return float64(x)
	case int32:
		return float64(x)
	case uint32:
		return float64(x)
	case int64:
		return float64(x)
	case uint64:
		return float64(x)
	case byte:
		return float64(x)
	default:
		return 0
	}
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
