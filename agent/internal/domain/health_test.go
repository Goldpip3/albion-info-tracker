package domain

import "testing"

func TestParamDoubleArrayVariants(t *testing.T) {
	cases := []struct {
		name string
		val  any
		want []float64
	}{
		{"float64", []float64{-50, -75}, []float64{-50, -75}},
		{"float32", []float32{1.5, 2.5}, []float64{1.5, 2.5}},
		{"int64", []int64{3, 4}, []float64{3, 4}},
		{"int16", []int16{5, 6}, []float64{5, 6}},
		{"any", []any{float64(7), int32(8)}, []float64{7, 8}},
		{"absent", nil, nil},
	}
	for _, c := range cases {
		p := map[byte]any{}
		if c.val != nil {
			p[2] = c.val
		}
		got := paramDoubleArray(p, 2)
		if len(got) != len(c.want) {
			t.Errorf("%s: len %d, want %d", c.name, len(got), len(c.want))
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("%s[%d] = %v, want %v", c.name, i, got[i], c.want[i])
			}
		}
	}
}

func TestParamLongArrayVariants(t *testing.T) {
	p := map[byte]any{
		1: []int32{10, 20},
		2: []int64{100, 200},
		3: []any{int64(1), float64(2)},
	}
	if got := paramLongArray(p, 1); len(got) != 2 || got[0] != 10 || got[1] != 20 {
		t.Errorf("int32 -> %v", got)
	}
	if got := paramLongArray(p, 2); len(got) != 2 || got[0] != 100 || got[1] != 200 {
		t.Errorf("int64 -> %v", got)
	}
	if got := paramLongArray(p, 3); len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Errorf("any -> %v", got)
	}
	if got := paramLongArray(p, 9); got != nil {
		t.Errorf("absent -> %v, want nil", got)
	}
}

// TestHandleHealthUpdatesBatched is the regression test for the
// under-counted-DPS bug: batched HealthUpdates (EventCode 7) carry the
// bulk of ability damage as parallel arrays. Two hits of 50 + 75 from
// one causer onto one target must sum to 125 dealt / 125 taken.
func TestHandleHealthUpdatesBatched(t *testing.T) {
	e := NewEngine()
	attacker := Guid{1}
	target := Guid{2}
	e.store.UpsertByGuid(attacker, 100, "Attacker", "")
	e.store.UpsertByGuid(target, 200, "Target", "")
	e.store.SetLocalGuid(attacker)

	p := map[byte]any{
		byte(0): int64(200),            // affected target
		byte(2): []float64{-50, -75},   // health changes (neg = damage)
		byte(6): []int64{100, 100},     // causers
		byte(7): []int64{5, 6},         // spell indices
	}
	e.handleHealthUpdates(p)

	atk := e.store.ByGuid(attacker)
	tgt := e.store.ByGuid(target)
	if atk == nil || tgt == nil {
		t.Fatal("entities missing after UpsertByGuid")
	}
	if atk.Overall.DamageDealt != 125 {
		t.Errorf("attacker DamageDealt = %d, want 125", atk.Overall.DamageDealt)
	}
	if tgt.Overall.DamageTaken != 125 {
		t.Errorf("target DamageTaken = %d, want 125", tgt.Overall.DamageTaken)
	}
}
