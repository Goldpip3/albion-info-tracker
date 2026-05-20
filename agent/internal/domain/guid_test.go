package domain

import "testing"

// TestParseGuidRoundTrip confirms ParseGuid is the exact inverse of
// String() — the property the persisted party roster relies on, since
// it stores g.String() and reparses it on restore.
func TestParseGuidRoundTrip(t *testing.T) {
	cases := []Guid{
		{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		{0xde, 0xad, 0xbe, 0xef, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x10, 0x20, 0x30, 0x40},
		{0xff, 0, 0xff, 0, 0xff, 0, 0xff, 0, 0xff, 0, 0xff, 0, 0xff, 0, 0xff, 0},
	}
	for _, want := range cases {
		s := want.String()
		got, err := ParseGuid(s)
		if err != nil {
			t.Fatalf("ParseGuid(%q): unexpected error %v", s, err)
		}
		if got != want {
			t.Errorf("round-trip mismatch for %q:\n  want %v\n  got  %v", s, want, got)
		}
		if got.String() != s {
			t.Errorf("re-String mismatch: %q -> %q", s, got.String())
		}
	}
}

func TestParseGuidRejectsGarbage(t *testing.T) {
	bad := []string{"", "not-a-guid", "1234", "zzzzzzzz-zzzz-zzzz-zzzz-zzzzzzzzzzzz"}
	for _, s := range bad {
		if _, err := ParseGuid(s); err == nil {
			t.Errorf("ParseGuid(%q): want error, got nil", s)
		}
	}
}
