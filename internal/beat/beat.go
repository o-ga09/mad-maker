// Package beat provides beat timing calculations independent of any rendering or audio layer.
package beat

import (
	"fmt"
	"math"
	"time"
)

// Interval returns the duration of a single beat at the given tempo.
// It returns an error if bpm is not a positive finite number.
func Interval(bpm float64) (time.Duration, error) {
	if math.IsNaN(bpm) || math.IsInf(bpm, 0) || bpm <= 0 {
		return 0, fmt.Errorf("invalid bpm %v: must be a positive finite number", bpm)
	}
	return time.Duration(float64(time.Minute) / bpm), nil
}
