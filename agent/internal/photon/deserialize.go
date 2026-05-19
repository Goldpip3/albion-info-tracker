package photon

import (
	"errors"
	"fmt"
)

// deserialize reads one Protocol18-encoded value (typeCode + payload).
func deserialize(r *reader) (any, error) {
	tc, err := r.byteAt()
	if err != nil {
		return nil, err
	}
	return deserializeKind(r, tc)
}

// deserializeKind reads a value of the given type code, without reading the
// type-code byte itself. Used when the caller has already consumed the type.
func deserializeKind(r *reader, code byte) (any, error) {
	if code >= byte(p18CustomTypeSlim) && code <= byte(p18MaxSlimCustom) {
		return deserializeCustom(r, code)
	}

	switch p18Type(code) {
	case p18Boolean:
		b, err := r.byteAt()
		return b != 0, err
	case p18Byte:
		return r.byteAt()
	case p18Short:
		return r.leInt16()
	case p18Float:
		return r.leFloat32()
	case p18Double:
		return r.leFloat64()
	case p18String:
		return deserializeString(r)
	case p18Null:
		return nil, nil
	case p18CompressedInt:
		return r.compressedInt32()
	case p18CompressedLong:
		return r.compressedInt64()
	case p18Int1:
		b, err := r.byteAt()
		return int32(b), err
	case p18Int1Negative:
		b, err := r.byteAt()
		return -int32(b), err
	case p18Int2:
		u, err := r.leUint16()
		return int32(u), err
	case p18Int2Negative:
		u, err := r.leUint16()
		return -int32(u), err
	case p18Long1:
		b, err := r.byteAt()
		return int64(b), err
	case p18Long1Negative:
		b, err := r.byteAt()
		return -int64(b), err
	case p18Long2:
		u, err := r.leUint16()
		return int64(u), err
	case p18Long2Negative:
		u, err := r.leUint16()
		return -int64(u), err
	case p18Custom:
		return deserializeCustom(r, 0)
	case p18Dictionary:
		return deserializeDictionary(r)
	case p18Hashtable:
		return deserializeHashtable(r)
	case p18ObjectArray:
		return deserializeObjectArray(r)
	case p18OperationRequest:
		return DeserializeOperationRequest(r)
	case p18OperationResponse:
		return DeserializeOperationResponse(r)
	case p18EventData:
		return DeserializeEventData(r)
	case p18BooleanFalse:
		return false, nil
	case p18BooleanTrue:
		return true, nil
	case p18ShortZero:
		return int16(0), nil
	case p18IntZero:
		return int32(0), nil
	case p18LongZero:
		return int64(0), nil
	case p18FloatZero:
		return float32(0), nil
	case p18DoubleZero:
		return float64(0), nil
	case p18ByteZero:
		return byte(0), nil
	case p18Array:
		return deserializeArrayInArray(r)
	case p18BooleanArray:
		return deserializeBooleanArray(r)
	case p18ByteArray:
		return deserializeByteArray(r)
	case p18ShortArray:
		return deserializeShortArray(r)
	case p18FloatArray:
		return deserializeFloatArray(r)
	case p18DoubleArray:
		return deserializeDoubleArray(r)
	case p18StringArray:
		return deserializeStringArray(r)
	case p18CompressedIntArray:
		return deserializeCompressedIntArray(r)
	case p18CompressedLongArray:
		return deserializeCompressedLongArray(r)
	case p18CustomTypeArray:
		return deserializeCustomTypeArray(r)
	case p18DictionaryArray:
		return deserializeDictionaryArray(r)
	case p18HashtableArray:
		return deserializeHashtableArray(r)
	default:
		return nil, fmt.Errorf("photon: unsupported type code %d", code)
	}
}

// DeserializeEventData reads an EventData value (without its outer type byte).
func DeserializeEventData(r *reader) (EventData, error) {
	code, err := r.byteAt()
	if err != nil {
		return EventData{}, err
	}
	params, err := deserializeParameterTable(r)
	if err != nil {
		return EventData{}, err
	}
	return EventData{Code: code, Parameters: params}, nil
}

// DeserializeOperationRequest reads an OperationRequest value.
func DeserializeOperationRequest(r *reader) (OperationRequest, error) {
	op, err := r.byteAt()
	if err != nil {
		return OperationRequest{}, err
	}
	params, err := deserializeParameterTable(r)
	if err != nil {
		return OperationRequest{}, err
	}
	return OperationRequest{OperationCode: op, Parameters: params}, nil
}

// DeserializeOperationResponse reads an OperationResponse value.
func DeserializeOperationResponse(r *reader) (OperationResponse, error) {
	op, err := r.byteAt()
	if err != nil {
		return OperationResponse{}, err
	}
	rc, err := deserializeShortLE(r)
	if err != nil {
		return OperationResponse{}, err
	}
	dbgTC, err := r.byteAt()
	if err != nil {
		return OperationResponse{}, err
	}
	dbgVal, err := deserializeKind(r, dbgTC)
	if err != nil {
		return OperationResponse{}, err
	}
	dbgStr, _ := dbgVal.(string)
	params, err := deserializeParameterTable(r)
	if err != nil {
		return OperationResponse{}, err
	}
	return OperationResponse{
		OperationCode: op,
		ReturnCode:    rc,
		DebugMessage:  dbgStr,
		Parameters:    params,
	}, nil
}

// deserializeShortLE matches C#'s DeserializeShort (LE 16-bit signed).
func deserializeShortLE(r *reader) (int16, error) {
	return r.leInt16()
}

func deserializeString(r *reader) (string, error) {
	n, err := r.compressedUint32()
	if err != nil {
		return "", err
	}
	if n == 0 {
		return "", nil
	}
	b, err := r.bytes(int(n))
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func deserializeByteArray(r *reader) ([]byte, error) {
	n, err := r.compressedUint32()
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return []byte{}, nil
	}
	src, err := r.bytes(int(n))
	if err != nil {
		return nil, err
	}
	// Copy out of the parser buffer so the caller can retain it.
	out := make([]byte, len(src))
	copy(out, src)
	return out, nil
}

func deserializeShortArray(r *reader) ([]int16, error) {
	n, err := r.compressedUint32()
	if err != nil {
		return nil, err
	}
	out := make([]int16, n)
	for i := range out {
		out[i], err = r.leInt16()
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func deserializeFloatArray(r *reader) ([]float32, error) {
	n, err := r.compressedUint32()
	if err != nil {
		return nil, err
	}
	out := make([]float32, n)
	for i := range out {
		out[i], err = r.leFloat32()
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func deserializeDoubleArray(r *reader) ([]float64, error) {
	n, err := r.compressedUint32()
	if err != nil {
		return nil, err
	}
	out := make([]float64, n)
	for i := range out {
		out[i], err = r.leFloat64()
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

var bitMasks = [8]byte{1, 2, 4, 8, 16, 32, 64, 128}

func deserializeBooleanArray(r *reader) ([]bool, error) {
	n, err := r.compressedUint32()
	if err != nil {
		return nil, err
	}
	out := make([]bool, n)
	full := int(n) / 8
	idx := 0
	for i := 0; i < full; i++ {
		b, err := r.byteAt()
		if err != nil {
			return nil, err
		}
		for j := 0; j < 8; j++ {
			out[idx] = b&bitMasks[j] != 0
			idx++
		}
	}
	if idx < int(n) {
		b, err := r.byteAt()
		if err != nil {
			return nil, err
		}
		for j := 0; idx < int(n); j++ {
			out[idx] = b&bitMasks[j] != 0
			idx++
		}
	}
	return out, nil
}

func deserializeStringArray(r *reader) ([]string, error) {
	n, err := r.compressedUint32()
	if err != nil {
		return nil, err
	}
	out := make([]string, n)
	for i := range out {
		out[i], err = deserializeString(r)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func deserializeCompressedIntArray(r *reader) ([]int32, error) {
	n, err := r.compressedUint32()
	if err != nil {
		return nil, err
	}
	out := make([]int32, n)
	for i := range out {
		out[i], err = r.compressedInt32()
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func deserializeCompressedLongArray(r *reader) ([]int64, error) {
	n, err := r.compressedUint32()
	if err != nil {
		return nil, err
	}
	out := make([]int64, n)
	for i := range out {
		out[i], err = r.compressedInt64()
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func deserializeObjectArray(r *reader) ([]any, error) {
	n, err := r.compressedUint32()
	if err != nil {
		return nil, err
	}
	out := make([]any, n)
	for i := range out {
		out[i], err = deserialize(r)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func deserializeArrayInArray(r *reader) ([]any, error) {
	// Array-of-arrays: each element is itself an array value.
	return deserializeObjectArray(r)
}

func deserializeCustom(r *reader, slimCode byte) (CustomType, error) {
	var tc byte
	if slimCode == 0 {
		b, err := r.byteAt()
		if err != nil {
			return CustomType{}, err
		}
		tc = b
	} else {
		tc = slimCode - byte(p18CustomTypeSlim)
	}
	n, err := r.compressedUint32()
	if err != nil {
		return CustomType{}, err
	}
	src, err := r.bytes(int(n))
	if err != nil {
		return CustomType{}, err
	}
	data := make([]byte, len(src))
	copy(data, src)
	return CustomType{TypeCode: tc, Data: data}, nil
}

func deserializeCustomTypeArray(r *reader) ([]CustomType, error) {
	n, err := r.compressedUint32()
	if err != nil {
		return nil, err
	}
	tc, err := r.byteAt()
	if err != nil {
		return nil, err
	}
	out := make([]CustomType, n)
	for i := range out {
		ln, err := r.compressedUint32()
		if err != nil {
			return nil, err
		}
		src, err := r.bytes(int(ln))
		if err != nil {
			return nil, err
		}
		data := make([]byte, len(src))
		copy(data, src)
		out[i] = CustomType{TypeCode: tc, Data: data}
	}
	return out, nil
}

func deserializeHashtable(r *reader) (map[any]any, error) {
	n, err := r.compressedUint32()
	if err != nil {
		return nil, err
	}
	out := make(map[any]any, n)
	for i := uint32(0); i < n; i++ {
		k, err := deserialize(r)
		if err != nil {
			return nil, err
		}
		v, err := deserialize(r)
		if err != nil {
			return nil, err
		}
		if k != nil {
			out[k] = v
		}
	}
	return out, nil
}

func deserializeHashtableArray(r *reader) ([]map[any]any, error) {
	n, err := r.compressedUint32()
	if err != nil {
		return nil, err
	}
	out := make([]map[any]any, n)
	for i := range out {
		out[i], err = deserializeHashtable(r)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

// deserializeDictionary reads a Photon Dictionary. We collapse all dictionaries
// to map[any]any rather than reconstructing strongly-typed maps via reflection
// (as the C# version does) — downstream handlers iterate generically and don't
// rely on the static key/value type.
func deserializeDictionary(r *reader) (map[any]any, error) {
	keyTC, err := r.byteAt()
	if err != nil {
		return nil, err
	}
	valueTC, err := r.byteAt()
	if err != nil {
		return nil, err
	}
	if err := consumeNestedTypeMetadata(r, p18Type(valueTC)); err != nil {
		return nil, err
	}
	return deserializeDictionaryElements(r, p18Type(keyTC), p18Type(valueTC))
}

func deserializeDictionaryArray(r *reader) ([]map[any]any, error) {
	keyTC, err := r.byteAt()
	if err != nil {
		return nil, err
	}
	valueTC, err := r.byteAt()
	if err != nil {
		return nil, err
	}
	if err := consumeNestedTypeMetadata(r, p18Type(valueTC)); err != nil {
		return nil, err
	}
	n, err := r.compressedUint32()
	if err != nil {
		return nil, err
	}
	out := make([]map[any]any, n)
	for i := range out {
		out[i], err = deserializeDictionaryElements(r, p18Type(keyTC), p18Type(valueTC))
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

// consumeNestedTypeMetadata reads (and discards) the inner type prefix bytes
// that appear when a dictionary's value type is itself a Dictionary or Array.
// We don't need the type info because we deserialize generically.
func consumeNestedTypeMetadata(r *reader, valueTC p18Type) error {
	switch valueTC {
	case p18Dictionary:
		// inner dict has its own key/value type bytes
		if _, err := r.byteAt(); err != nil {
			return err
		}
		innerValue, err := r.byteAt()
		if err != nil {
			return err
		}
		return consumeNestedTypeMetadata(r, p18Type(innerValue))
	case p18Array:
		// nested array: one type-code per nesting level, then the leaf type
		tc, err := r.byteAt()
		if err != nil {
			return err
		}
		for p18Type(tc) == p18Array {
			tc, err = r.byteAt()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func deserializeDictionaryElements(r *reader, keyTC, valueTC p18Type) (map[any]any, error) {
	n, err := r.compressedUint32()
	if err != nil {
		return nil, err
	}
	out := make(map[any]any, n)
	for i := uint32(0); i < n; i++ {
		var (
			k, v any
			kerr error
		)
		if keyTC == p18Unknown {
			k, kerr = deserialize(r)
		} else {
			k, kerr = deserializeKind(r, byte(keyTC))
		}
		if kerr != nil {
			return nil, kerr
		}
		if valueTC == p18Unknown {
			v, err = deserialize(r)
		} else {
			v, err = deserializeKind(r, byte(valueTC))
		}
		if err != nil {
			return nil, err
		}
		if k != nil {
			out[k] = v
		}
	}
	return out, nil
}

func deserializeParameterTable(r *reader) (map[byte]any, error) {
	n, err := r.byteAt()
	if err != nil {
		return nil, err
	}
	out := make(map[byte]any, n)
	for i := byte(0); i < n; i++ {
		key, err := r.byteAt()
		if err != nil {
			return nil, err
		}
		valueTC, err := r.byteAt()
		if err != nil {
			return nil, err
		}
		v, err := deserializeKind(r, valueTC)
		if err != nil {
			return nil, err
		}
		out[key] = v
	}
	return out, nil
}

// errInvalidParameterTable is returned when DeserializeParameterTable cannot
// recover. Currently unused — kept for forward compatibility.
var errInvalidParameterTable = errors.New("photon: invalid parameter table")

var _ = errInvalidParameterTable
