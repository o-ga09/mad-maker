// Package timeline computes which image is shown at which time, independent of any rendering layer.
package timeline

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/o-ga09/mad-maker/internal/beat"
)

// Params are the inputs of a timeline.
type Params struct {
	// BPM is the tempo in beats per minute. Must be a positive finite number.
	BPM float64
	// Offset is the position of the first beat. Must be in [0, Duration).
	Offset time.Duration
	// Duration is the length of the song. Must be positive.
	Duration time.Duration
	// ImageCount is the number of images. Must be at least 1.
	ImageCount int
	// BeatsPerSwitch is how many beats pass between image switches. Must be at least 1.
	BeatsPerSwitch int
}

// Segment is a span of time during which one image is shown.
type Segment struct {
	// Start is the beginning of the span (inclusive).
	Start time.Duration
	// End is the end of the span (exclusive). The last segment may end mid-beat.
	End time.Duration
	// ImageIndex is the 0-based index of the image to show.
	ImageIndex int
	// Beat is the number of the beat at which the segment starts.
	Beat int
}

// Timeline is an immutable, time-ordered list of segments.
type Timeline struct {
	segments []Segment
	offset   time.Duration
	end      time.Duration
}

// New builds a timeline from p. It returns an error if any input is invalid.
func New(p Params) (*Timeline, error) {
	interval, err := beat.Interval(p.BPM)
	if err != nil {
		return nil, fmt.Errorf("compute beat interval: %w", err)
	}
	switch {
	case p.ImageCount < 1:
		return nil, fmt.Errorf("invalid image count %d: must be at least 1", p.ImageCount)
	case p.Duration <= 0:
		return nil, fmt.Errorf("invalid duration %v: must be positive", p.Duration)
	case p.BeatsPerSwitch < 1:
		return nil, fmt.Errorf("invalid beats per switch %d: must be at least 1", p.BeatsPerSwitch)
	case p.Offset < 0:
		return nil, fmt.Errorf("invalid offset %v: must not be negative", p.Offset)
	case p.Offset >= p.Duration:
		return nil, fmt.Errorf("invalid offset %v: must be less than duration %v", p.Offset, p.Duration)
	}

	// Guard against a zero interval (extreme BPM truncates to 0) and against
	// overflow, either of which would stop the loop below from advancing.
	if interval <= 0 || time.Duration(p.BeatsPerSwitch) > math.MaxInt64/interval {
		return nil, fmt.Errorf("invalid tempo/switch interval: bpm %v with %d beats per switch gives an unusable step", p.BPM, p.BeatsPerSwitch)
	}
	step := time.Duration(p.BeatsPerSwitch) * interval
	var segments []Segment
	for k, start := 0, p.Offset; start < p.Duration; k++ {
		segment := Segment{
			Start:      start,
			End:        p.Duration,
			ImageIndex: k % p.ImageCount,
			Beat:       k * p.BeatsPerSwitch,
		}
		// Compare against the remaining time instead of adding, so start+step cannot overflow.
		if step >= p.Duration-start {
			segments = append(segments, segment)
			break
		}
		start += step
		segment.End = start
		segments = append(segments, segment)
	}
	return &Timeline{segments: segments, offset: p.Offset, end: p.Duration}, nil
}

// Segments returns a copy of the segments in chronological order.
func (t *Timeline) Segments() []Segment {
	return append([]Segment(nil), t.segments...)
}

// ImageAt returns the index of the image shown at time at, in O(log n).
// Times before the first beat return image 0. Times at or after the end of the
// song return ok == false.
func (t *Timeline) ImageAt(at time.Duration) (index int, ok bool) {
	if at >= t.end {
		return 0, false
	}
	if at < t.offset {
		return 0, true
	}
	// First segment that ends after at; it is the one containing at.
	i := sort.Search(len(t.segments), func(i int) bool { return t.segments[i].End > at })
	return t.segments[i].ImageIndex, true
}
