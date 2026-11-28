// Package effect decides which visual effect to apply on each beat and
// computes the effect's state over time. It is pure logic with no dependency
// on any rendering layer, so preview and export can share the same results.
package effect

import (
	"math"
	"time"
)

// Kind is the type of visual effect applied on a beat.
type Kind int

// Effect kinds.
const (
	None Kind = iota
	Flash
	Zoom
	Shake
)

// String returns a lowercase name for the kind.
func (k Kind) String() string {
	switch k {
	case None:
		return "none"
	case Flash:
		return "flash"
	case Zoom:
		return "zoom"
	case Shake:
		return "shake"
	default:
		return "unknown"
	}
}

const (
	// BeatsPerBar is the number of beats in a bar; the first beat of each bar
	// is the downbeat and always receives a strong effect.
	BeatsPerBar = 4

	// StrongStrength is the strength of effects on downbeats.
	StrongStrength = 1.0
	// WeakStrength is the strength of effects on other beats.
	WeakStrength = 0.5

	// MaxZoomGain is the extra scale added to 1.0 by a zoom of strength 1 at the beat start.
	MaxZoomGain = 0.15
	// MaxShakePixels is the shake displacement in pixels for strength 1 at the beat start.
	MaxShakePixels = 12.0
)

// Effect is the effect assigned to one beat.
type Effect struct {
	Kind     Kind
	Strength float64 // 0..1; scales the effect's intensity
}

// State is the effect state at one instant, ready to be applied by a renderer.
type State struct {
	FlashAlpha float64 // opacity of a white overlay, 0..1
	Zoom       float64 // scale factor, >= 1
	ShakeX     float64 // horizontal offset in pixels
	ShakeY     float64 // vertical offset in pixels
}

// Assign decides the effect for the given beat number (0-based).
//
// Rules: the first beat of every bar (beat % BeatsPerBar == 0) always gets a
// flash, zoom or shake at StrongStrength. Other beats get no effect with
// probability 2/5, otherwise a flash, zoom or shake at WeakStrength.
// The result depends only on seed and beat, so equal inputs always give equal
// results. Negative beats get no effect.
func Assign(seed uint64, beat int) Effect {
	if beat < 0 {
		return Effect{Kind: None}
	}
	h := mix(seed, uint64(beat), 0)
	if beat%BeatsPerBar == 0 {
		return Effect{Kind: kinds[h%3], Strength: StrongStrength}
	}
	if idx := h % 5; idx >= 2 {
		return Effect{Kind: kinds[idx-2], Strength: WeakStrength}
	}
	return Effect{Kind: None}
}

var kinds = [3]Kind{Flash, Zoom, Shake}

// State returns the effect state elapsed after the start of the beat, where
// interval is the beat length. The effect is at its maximum at elapsed == 0
// and decays linearly to rest at elapsed >= interval. Shake direction is
// derived from seed and beat. A negative beat or non-positive interval yields the rest state.
func (e Effect) State(seed uint64, beat int, elapsed, interval time.Duration) State {
	rest := State{Zoom: 1}
	if e.Kind == None || beat < 0 || interval <= 0 {
		return rest
	}
	p := float64(elapsed) / float64(interval)
	p = math.Min(math.Max(p, 0), 1)
	level := e.Strength * (1 - p)

	switch e.Kind {
	case Flash:
		rest.FlashAlpha = math.Min(math.Max(level, 0), 1)
	case Zoom:
		rest.Zoom = 1 + MaxZoomGain*level
	case Shake:
		angle := float64(mix(seed, uint64(beat), 1)%3600) / 3600 * 2 * math.Pi
		rest.ShakeX = MaxShakePixels * level * math.Cos(angle)
		rest.ShakeY = MaxShakePixels * level * math.Sin(angle)
	}
	return rest
}

// mix is a splitmix64-style hash combining its inputs into a well-spread value.
func mix(a, b, c uint64) uint64 {
	x := a + 0x9e3779b97f4a7c15
	for _, v := range [...]uint64{b, c} {
		x ^= v + 0x9e3779b97f4a7c15 + (x << 6) + (x >> 2)
		x = (x ^ (x >> 30)) * 0xbf58476d1ce4e5b9
		x = (x ^ (x >> 27)) * 0x94d049bb133111eb
		x ^= x >> 31
	}
	return x
}
