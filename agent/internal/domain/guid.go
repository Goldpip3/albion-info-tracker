package domain

import (
	"encoding/hex"
	"fmt"
	"strings"
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

// ParseGuid is the inverse of String(): it takes the dashed 8-4-4-4-12
// textual form and recovers the 16-byte storage order, undoing the
// .NET byte-swap on the first three groups. Round-trips g.String()
// exactly. Used to rehydrate a persisted party roster and to resolve
// guids the web sends in addPartyMember/removePartyMember commands.
func ParseGuid(s string) (Guid, error) {
	var g Guid
	clean := strings.ReplaceAll(s, "-", "")
	if len(clean) != 32 {
		return g, fmt.Errorf("guid: want 32 hex chars, got %d in %q", len(clean), s)
	}
	t, err := hex.DecodeString(clean)
	if err != nil {
		return g, fmt.Errorf("guid: %w", err)
	}
	// Undo String()'s layout: textual bytes t[0..15] map back to storage
	// b[..] with the first three groups reversed, last two straight.
	g[0], g[1], g[2], g[3] = t[3], t[2], t[1], t[0]
	g[4], g[5] = t[5], t[4]
	g[6], g[7] = t[7], t[6]
	g[8], g[9] = t[8], t[9]
	copy(g[10:], t[10:16])
	return g, nil
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
