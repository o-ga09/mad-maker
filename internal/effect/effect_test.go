package effect

import (
	"math"
	"testing"
	"time"
)

const interval = 500 * time.Millisecond

func TestKindString(t *testing.T) {
	tests := []struct {
		kind Kind
		want string
	}{
		{None, "none"},
		{Flash, "flash"},
		{Zoom, "zoom"},
		{Shake, "shake"},
		{Kind(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.kind.String(); got != tt.want {
			t.Errorf("Kind(%d).String() = %q, want %q", tt.kind, got, tt.want)
		}
	}
}

func TestAssignDeterministic(t *testing.T) {
	for _, seed := range []uint64{0, 1, 42, 1 << 63} {
		for b := 0; b < 200; b++ {
			if a, c := Assign(seed, b), Assign(seed, b); a != c {
				t.Fatalf("Assign(%d, %d) not deterministic: %+v vs %+v", seed, b, a, c)
			}
		}
	}
}

func TestAssignSeedChangesResult(t *testing.T) {
	differs := false
	for b := 0; b < 200 && !differs; b++ {
		differs = Assign(1, b) != Assign(2, b)
	}
	if !differs {
		t.Error("different seeds produced identical assignments for 200 beats")
	}
}

func TestAssignDownbeatIsStrong(t *testing.T) {
	for seed := uint64(0); seed < 20; seed++ {
		for b := 0; b < 400; b++ {
			e := Assign(seed, b)
			if b%BeatsPerBar == 0 {
				if e.Kind == None {
					t.Fatalf("seed %d beat %d: downbeat must have an effect", seed, b)
				}
				if e.Strength != StrongStrength {
					t.Fatalf("seed %d beat %d: downbeat strength = %v, want %v", seed, b, e.Strength, StrongStrength)
				}
			} else if e.Kind != None && e.Strength != WeakStrength {
				t.Fatalf("seed %d beat %d: off-beat strength = %v, want %v", seed, b, e.Strength, WeakStrength)
			}
		}
	}
}

func TestAssignOffBeatCanBeNoneAndAllKindsAppear(t *testing.T) {
	seen := map[Kind]bool{}
	for b := 0; b < 2000; b++ {
		seen[Assign(7, b).Kind] = true
	}
	for _, k := range []Kind{None, Flash, Zoom, Shake} {
		if !seen[k] {
			t.Errorf("kind %v never assigned in 2000 beats", k)
		}
	}
}

func TestAssignNegativeBeat(t *testing.T) {
	if e := Assign(1, -3); e.Kind != None {
		t.Errorf("Assign(1, -3) = %+v, want None", e)
	}
}

func TestStateEnvelope(t *testing.T) {
	const seed, beatNo = 5, 3
	tests := []struct {
		name string
		e    Effect
		peak func(State) float64
		rest func(State) bool
	}{
		{"flash", Effect{Flash, 1}, func(s State) float64 { return s.FlashAlpha }, func(s State) bool { return s.FlashAlpha == 0 }},
		{"zoom", Effect{Zoom, 1}, func(s State) float64 { return s.Zoom - 1 }, func(s State) bool { return s.Zoom == 1 }},
		{"shake", Effect{Shake, 1}, func(s State) float64 { return math.Hypot(s.ShakeX, s.ShakeY) }, func(s State) bool { return s.ShakeX == 0 && s.ShakeY == 0 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := tt.peak(tt.e.State(seed, beatNo, 0, interval))
			if start <= 0 {
				t.Fatalf("peak at beat start = %v, want > 0", start)
			}
			prev := start
			for _, frac := range []float64{0.25, 0.5, 0.75} {
				v := tt.peak(tt.e.State(seed, beatNo, time.Duration(frac*float64(interval)), interval))
				if v >= prev || v <= 0 {
					t.Errorf("at %v: value %v not strictly decaying from %v", frac, v, prev)
				}
				prev = v
			}
			if !tt.rest(tt.e.State(seed, beatNo, interval, interval)) {
				t.Errorf("state at beat end not at rest: %+v", tt.e.State(seed, beatNo, interval, interval))
			}
			if !tt.rest(tt.e.State(seed, beatNo, 2*interval, interval)) {
				t.Error("state past beat end not at rest")
			}
		})
	}
}

func TestStateBounds(t *testing.T) {
	for _, k := range []Kind{Flash, Zoom, Shake} {
		for _, el := range []time.Duration{-time.Second, 0, interval / 3, interval, 3 * interval} {
			s := Effect{k, StrongStrength}.State(9, 4, el, interval)
			if s.FlashAlpha < 0 || s.FlashAlpha > 1 {
				t.Errorf("%v el=%v: FlashAlpha %v out of [0,1]", k, el, s.FlashAlpha)
			}
			if s.Zoom < 1 {
				t.Errorf("%v el=%v: Zoom %v < 1", k, el, s.Zoom)
			}
		}
	}
}

func TestStateOnlyOwnKind(t *testing.T) {
	s := Effect{Flash, 1}.State(1, 1, 0, interval)
	if s.Zoom != 1 || s.ShakeX != 0 || s.ShakeY != 0 {
		t.Errorf("flash affected other fields: %+v", s)
	}
}

func TestStateNoneAndInvalid(t *testing.T) {
	rest := State{Zoom: 1}
	if got := (Effect{None, 1}).State(1, 1, 0, interval); got != rest {
		t.Errorf("None state = %+v, want %+v", got, rest)
	}
	if got := (Effect{Flash, 1}).State(1, 1, 0, 0); got != rest {
		t.Errorf("zero interval state = %+v, want %+v", got, rest)
	}
	if got := (Effect{Flash, 1}).State(1, 1, 0, -interval); got != rest {
		t.Errorf("negative interval state = %+v, want %+v", got, rest)
	}
}

func TestStateDeterministicAndSeedDependentShake(t *testing.T) {
	e := Effect{Shake, StrongStrength}
	if a, b := e.State(3, 8, 0, interval), e.State(3, 8, 0, interval); a != b {
		t.Errorf("shake not deterministic: %+v vs %+v", a, b)
	}
	differs := false
	for b := 0; b < 50 && !differs; b++ {
		differs = e.State(1, b, 0, interval) != e.State(2, b, 0, interval)
	}
	if !differs {
		t.Error("shake offsets identical across seeds")
	}
}

func TestStrongerEffectIsLarger(t *testing.T) {
	weak := Effect{Flash, WeakStrength}.State(1, 1, 0, interval).FlashAlpha
	strong := Effect{Flash, StrongStrength}.State(1, 1, 0, interval).FlashAlpha
	if strong <= weak {
		t.Errorf("strong %v <= weak %v", strong, weak)
	}
}
