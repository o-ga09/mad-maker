package timeline

import (
	"testing"
	"time"
)

const ms = time.Millisecond

func TestNew(t *testing.T) {
	tests := []struct {
		name string
		in   Params
		want []Segment
	}{
		{
			name: "spec example: last segment ends mid-beat and images repeat",
			in:   Params{BPM: 120, Duration: 1200 * ms, ImageCount: 2, BeatsPerSwitch: 1},
			want: []Segment{
				{Start: 0, End: 500 * ms, ImageIndex: 0, Beat: 0},
				{Start: 500 * ms, End: 1000 * ms, ImageIndex: 1, Beat: 1},
				{Start: 1000 * ms, End: 1200 * ms, ImageIndex: 0, Beat: 2},
			},
		},
		{
			name: "two beats per switch",
			in:   Params{BPM: 120, Duration: 2 * time.Second, ImageCount: 3, BeatsPerSwitch: 2},
			want: []Segment{
				{Start: 0, End: time.Second, ImageIndex: 0, Beat: 0},
				{Start: time.Second, End: 2 * time.Second, ImageIndex: 1, Beat: 2},
			},
		},
		{
			name: "offset shifts the first segment",
			in:   Params{BPM: 120, Offset: 250 * ms, Duration: 1100 * ms, ImageCount: 5, BeatsPerSwitch: 1},
			want: []Segment{
				{Start: 250 * ms, End: 750 * ms, ImageIndex: 0, Beat: 0},
				{Start: 750 * ms, End: 1100 * ms, ImageIndex: 1, Beat: 1},
			},
		},
		{
			name: "duration exactly on a beat boundary has no empty tail segment",
			in:   Params{BPM: 120, Duration: time.Second, ImageCount: 1, BeatsPerSwitch: 1},
			want: []Segment{
				{Start: 0, End: 500 * ms, ImageIndex: 0, Beat: 0},
				{Start: 500 * ms, End: time.Second, ImageIndex: 0, Beat: 1},
			},
		},
		{
			name: "interval longer than the song yields a single segment",
			in:   Params{BPM: 60, Duration: 500 * ms, ImageCount: 4, BeatsPerSwitch: 4},
			want: []Segment{
				{Start: 0, End: 500 * ms, ImageIndex: 0, Beat: 0},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tl, err := New(tt.in)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}
			got := tl.Segments()
			if len(got) != len(tt.want) {
				t.Fatalf("Segments() = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("Segments()[%d] = %+v, want %+v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestNewInvalid(t *testing.T) {
	valid := Params{BPM: 120, Duration: time.Second, ImageCount: 2, BeatsPerSwitch: 1}
	tests := []struct {
		name   string
		mutate func(p *Params)
	}{
		{"bpm zero", func(p *Params) { p.BPM = 0 }},
		{"bpm negative", func(p *Params) { p.BPM = -1 }},
		{"no images", func(p *Params) { p.ImageCount = 0 }},
		{"negative images", func(p *Params) { p.ImageCount = -1 }},
		{"duration zero", func(p *Params) { p.Duration = 0 }},
		{"duration negative", func(p *Params) { p.Duration = -time.Second }},
		{"switch interval zero", func(p *Params) { p.BeatsPerSwitch = 0 }},
		{"switch interval negative", func(p *Params) { p.BeatsPerSwitch = -1 }},
		{"offset negative", func(p *Params) { p.Offset = -ms }},
		{"offset equals duration", func(p *Params) { p.Offset = p.Duration }},
		{"offset beyond duration", func(p *Params) { p.Offset = p.Duration + ms }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := valid
			tt.mutate(&p)
			if _, err := New(p); err == nil {
				t.Errorf("New(%+v) error = nil, want error", p)
			}
		})
	}
}

func TestImageAt(t *testing.T) {
	// 120 BPM, offset 250ms, 1.2s: segments [250,750) img0, [750,1200) img1.
	tl, err := New(Params{BPM: 120, Offset: 250 * ms, Duration: 1200 * ms, ImageCount: 2, BeatsPerSwitch: 1})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		at     time.Duration
		want   int
		wantOK bool
	}{
		{"time zero is before offset", 0, 0, true},
		{"just before offset", 249 * ms, 0, true},
		{"exactly offset", 250 * ms, 0, true},
		{"inside first segment", 500 * ms, 0, true},
		{"just before boundary", 749 * ms, 0, true},
		{"exactly on boundary", 750 * ms, 1, true},
		{"inside last partial segment", 1199 * ms, 1, true},
		{"exactly at song end", 1200 * ms, 0, false},
		{"after song end", 5 * time.Second, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := tl.ImageAt(tt.at)
			if ok != tt.wantOK || got != tt.want {
				t.Errorf("ImageAt(%v) = (%d, %v), want (%d, %v)", tt.at, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestImageAtWraps(t *testing.T) {
	tl, err := New(Params{BPM: 120, Duration: 3 * time.Second, ImageCount: 3, BeatsPerSwitch: 1})
	if err != nil {
		t.Fatal(err)
	}
	// beat k is at k*500ms and shows image k%3.
	for k := 0; k < 6; k++ {
		at := time.Duration(k)*500*ms + ms
		got, ok := tl.ImageAt(at)
		if !ok || got != k%3 {
			t.Errorf("ImageAt(%v) = (%d, %v), want (%d, true)", at, got, ok, k%3)
		}
	}
}

func TestSegmentsIsCopy(t *testing.T) {
	tl, err := New(Params{BPM: 120, Duration: time.Second, ImageCount: 2, BeatsPerSwitch: 1})
	if err != nil {
		t.Fatal(err)
	}
	tl.Segments()[0].ImageIndex = 99
	if got, _ := tl.ImageAt(0); got != 0 {
		t.Errorf("mutating Segments() result changed the timeline: ImageAt(0) = %d", got)
	}
}
