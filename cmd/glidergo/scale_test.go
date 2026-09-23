package main

import (
	"strings"
	"testing"

	"github.com/bwenstar/gliderGo/internal/platform"
	"github.com/bwenstar/gliderGo/internal/prefs"
)

func TestWindowScale(t *testing.T) {
	// The development desktop, a 1080p monitor, a 4K one, a laptop, and no answer at all.
	desk := &platform.Room{W: 2510, H: 1256, From: "the desk"}
	hd := &platform.Room{W: 1904, H: 992, From: "a 1080p monitor"}
	uhd := &platform.Room{W: 3824, H: 2064, From: "a 4K monitor"}
	laptop := &platform.Room{W: 1350, H: 672, From: "a laptop"}
	for _, tc := range []struct {
		name      string
		asked     int
		given     bool
		room      *platform.Room
		want      int
		wantNote  string // a phrase the note has, or "" for no note
		wantScale string // the banner's word
	}{
		{"auto on the desk", prefs.ScaleAuto, false, desk, 2, "", "2 (auto)"},
		{"auto on 1080p", prefs.ScaleAuto, false, hd, 2, "", "2 (auto)"},
		{"auto on 4K stops at the cap", prefs.ScaleAuto, false, uhd, autoMax, "", "3 (auto)"},
		{"auto on a laptop", prefs.ScaleAuto, false, laptop, 1, "", "1 (auto)"},
		{"auto, hermetic or unanswered", prefs.ScaleAuto, false, nil, 1, "", "1 (auto)"},
		{"a setting that fits", 2, false, desk, 2, "", "2"},
		{"4x asked for on 4K", 4, false, uhd, 4, "", "4"},
		{"a setting too big for the monitor", 3, false, hd, 2, "the setting is unchanged", "2 (fitted from 3)"},
		{"a setting with no answer", 3, false, nil, 3, "", "3"},
		{"a -scale too big is kept", 4, true, desk, 4, "the largest that fits is 2", "4"},
		{"a -scale that fits", 2, true, desk, 2, "", "2"},
	} {
		got, note := windowScale(tc.asked, tc.given, tc.room, platform.ScreenWidth, platform.ScreenHeight)
		if got != tc.want {
			t.Errorf("%s: scale %d, want %d", tc.name, got, tc.want)
		}
		if (note == "") != (tc.wantNote == "") || !strings.Contains(note, tc.wantNote) {
			t.Errorf("%s: note %q, want one with %q", tc.name, note, tc.wantNote)
		}
		if tc.room != nil && note != "" && !strings.Contains(note, tc.room.From) {
			t.Errorf("%s: note %q does not say what it measured", tc.name, note)
		}
		if w := scaleWord(got, tc.asked); w != tc.wantScale {
			t.Errorf("%s: banner says %q, want %q", tc.name, w, tc.wantScale)
		}
	}
}
