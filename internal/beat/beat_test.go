package beat

import (
	"math"
	"testing"
	"time"
)

func TestInterval(t *testing.T) {
	tests := []struct {
		name    string
		bpm     float64
		want    time.Duration
		wantErr bool
	}{
		{name: "60 BPM", bpm: 60, want: time.Second},
		{name: "120 BPM", bpm: 120, want: 500 * time.Millisecond},
		{name: "128 BPM", bpm: 128, want: 468750 * time.Microsecond},
		{name: "zero", bpm: 0, wantErr: true},
		{name: "negative", bpm: -120, wantErr: true},
		{name: "NaN", bpm: math.NaN(), wantErr: true},
		{name: "+Inf", bpm: math.Inf(1), wantErr: true},
		{name: "-Inf", bpm: math.Inf(-1), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Interval(tt.bpm)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Interval(%v) error = %v, wantErr %v", tt.bpm, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("Interval(%v) = %v, want %v", tt.bpm, got, tt.want)
			}
		})
	}
}
