package domain

import (
	"encoding/hex"
	"fmt"
)

// Guid is a 16-byte player identifier as Albion delivers it on the wire.
// Stored in the same byte order Photon uses (little-endian first four bytes
// of the standard GUID textual form — the same .NET layout).
type Guid [16]byte

// String renders the Guid in the conventional 8-4-4-4-12 dashed form.
// The first three groups are byte-swapped relative to the storage order
// to match .NET's Guid.ToString convention.
func (g Guid) String() string {
	b := g[:]
	return fmt.Sprintf("%02x%02x%02x%02x-%02x%02x-%02x%02x-%s-%s",
		b[3], b[2], b[1], b[0],
		b[5], b[4],
		b[7], b[6],
		hex.EncodeToString(b[8:10]),
		hex.EncodeToString(b[10:16]),
	)
}

// IsZero reports whether the Guid is the all-zero sentinel.
func (g Guid) IsZero() bool {
	for _, b := range g {
		if b != 0 {
			return false
		}
	}
	return true
}
