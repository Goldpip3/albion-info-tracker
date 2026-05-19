// Package photon parses Albion Online's Photon-over-UDP traffic.
// The outer envelope is big-endian; Protocol18 payloads are little-endian.
package photon

import (
	"encoding/binary"
	"errors"
	"math"
)

var errShortRead = errors.New("photon: short read")

// reader walks a byte slice with a cursor and bounds-checked accessors.
// Endianness is fixed per method (be = big-endian, le = little-endian).
type reader struct {
	buf []byte
	pos int
}

func (r *reader) remaining() int          { return len(r.buf) - r.pos }
func (r *reader) skip(n int) error        { return r.advance(n) }
func (r *reader) advance(n int) error {
	if r.pos+n > len(r.buf) {
		return errShortRead
	}
	r.pos += n
	return nil
}

func (r *reader) byteAt() (byte, error) {
	if r.pos >= len(r.buf) {
		return 0, errShortRead
	}
	v := r.buf[r.pos]
	r.pos++
	return v, nil
}

func (r *reader) bytes(n int) ([]byte, error) {
	if r.pos+n > len(r.buf) {
		return nil, errShortRead
	}
	out := r.buf[r.pos : r.pos+n]
	r.pos += n
	return out, nil
}

func (r *reader) beInt16() (int16, error) {
	if r.pos+2 > len(r.buf) {
		return 0, errShortRead
	}
	v := int16(binary.BigEndian.Uint16(r.buf[r.pos:]))
	r.pos += 2
	return v, nil
}

func (r *reader) beInt32() (int32, error) {
	if r.pos+4 > len(r.buf) {
		return 0, errShortRead
	}
	v := int32(binary.BigEndian.Uint32(r.buf[r.pos:]))
	r.pos += 4
	return v, nil
}

func (r *reader) leInt16() (int16, error) {
	if r.pos+2 > len(r.buf) {
		return 0, errShortRead
	}
	v := int16(binary.LittleEndian.Uint16(r.buf[r.pos:]))
	r.pos += 2
	return v, nil
}

func (r *reader) leUint16() (uint16, error) {
	if r.pos+2 > len(r.buf) {
		return 0, errShortRead
	}
	v := binary.LittleEndian.Uint16(r.buf[r.pos:])
	r.pos += 2
	return v, nil
}

func (r *reader) leFloat32() (float32, error) {
	if r.pos+4 > len(r.buf) {
		return 0, errShortRead
	}
	v := math.Float32frombits(binary.LittleEndian.Uint32(r.buf[r.pos:]))
	r.pos += 4
	return v, nil
}

func (r *reader) leFloat64() (float64, error) {
	if r.pos+8 > len(r.buf) {
		return 0, errShortRead
	}
	v := math.Float64frombits(binary.LittleEndian.Uint64(r.buf[r.pos:]))
	r.pos += 8
	return v, nil
}

// compressedUint32 is a varint: 7 bits per byte, MSB set means continuation.
func (r *reader) compressedUint32() (uint32, error) {
	var v uint32
	shift := uint(0)
	for shift < 35 {
		b, err := r.byteAt()
		if err != nil {
			return 0, err
		}
		v |= uint32(b&0x7F) << shift
		shift += 7
		if b&0x80 == 0 {
			return v, nil
		}
	}
	return v, nil
}

func (r *reader) compressedUint64() (uint64, error) {
	var v uint64
	shift := uint(0)
	for shift < 70 {
		b, err := r.byteAt()
		if err != nil {
			return 0, err
		}
		v |= uint64(b&0x7F) << shift
		shift += 7
		if b&0x80 == 0 {
			return v, nil
		}
	}
	return v, nil
}

func (r *reader) compressedInt32() (int32, error) {
	u, err := r.compressedUint32()
	if err != nil {
		return 0, err
	}
	return int32((u >> 1) ^ uint32(-int32(u&1))), nil
}

func (r *reader) compressedInt64() (int64, error) {
	u, err := r.compressedUint64()
	if err != nil {
		return 0, err
	}
	return int64((u >> 1) ^ uint64(-int64(u&1))), nil
}
