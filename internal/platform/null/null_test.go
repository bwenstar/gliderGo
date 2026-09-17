package null

import (
	"encoding/binary"
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"github.com/bwenstar/gliderGo/internal/platform"
)

func TestWindowRecordsFramesAndScriptedInput(t *testing.T) {
	dir := t.TempDir()
	win, err := New(platform.Config{}, dir)
	if err != nil {
		t.Fatal(err)
	}
	win.Script = map[int][]platform.Event{
		0: {{Kind: platform.EventKeyDown, Key: platform.KeyRight}},
		1: {{Kind: platform.EventKeyUp, Key: platform.KeyRight}, {Kind: platform.EventQuit}},
	}

	fb := platform.NewFramebuffer(platform.ScreenWidth, platform.ScreenHeight)

	win.PollEvents()
	if !win.KeyDown(platform.KeyRight) {
		t.Error("scripted key down was not recorded as held")
	}
	fb.Fill(color.RGBA{R: 0x20, G: 0x40, B: 0x60, A: 0xff})
	if err := win.Present(fb); err != nil {
		t.Fatal(err)
	}

	win.PollEvents()
	if win.KeyDown(platform.KeyRight) {
		t.Error("scripted key up did not clear the held state")
	}
	if !win.Closed() {
		t.Error("scripted quit did not close the window")
	}
	if err := win.Present(fb); err != nil {
		t.Fatal(err)
	}

	if got := win.Frames(); got != 2 {
		t.Errorf("Frames() = %d, want 2", got)
	}
	for _, name := range []string{"frame-000000.png", "frame-000001.png"} {
		if st, err := os.Stat(filepath.Join(dir, name)); err != nil || st.Size() == 0 {
			t.Errorf("expected a non-empty %s: %v", name, err)
		}
	}
	if win.LastFrame() == nil {
		t.Fatal("LastFrame() = nil after Present")
	}
	if o := 0; win.LastFrame().Pix[o] != 0x60 || win.LastFrame().Pix[o+2] != 0x20 {
		t.Errorf("recorded frame has BGRX %v, want blue-ish {0x60 0x40 0x20}", win.LastFrame().Pix[:4])
	}
}

func TestPresentRejectsMismatchedFramebuffer(t *testing.T) {
	win, err := New(platform.Config{Width: 32, Height: 16}, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := win.Present(platform.NewFramebuffer(64, 16)); err == nil {
		t.Error("Present accepted a framebuffer of the wrong size")
	}
}

func TestWAVSinkWritesAValidHeader(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.wav")
	sink, err := NewWAVSink(path, 22254, 1)
	if err != nil {
		t.Fatal(err)
	}
	if sink.SampleRate() != 22254 || sink.Channels() != 1 {
		t.Errorf("sink reports %d Hz / %d ch, want 22254 / 1", sink.SampleRate(), sink.Channels())
	}
	samples := make([]int16, 1000)
	for i := range samples {
		samples[i] = int16(i * 30)
	}
	if err := sink.Write(samples); err != nil {
		t.Fatal(err)
	}
	if err := sink.Close(); err != nil {
		t.Fatal(err)
	}

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if want := 44 + len(samples)*2; len(b) != want {
		t.Fatalf("file is %d bytes, want %d", len(b), want)
	}
	if string(b[0:4]) != "RIFF" || string(b[8:12]) != "WAVE" || string(b[36:40]) != "data" {
		t.Error("RIFF/WAVE/data chunk tags are wrong")
	}
	if got := binary.LittleEndian.Uint32(b[24:]); got != 22254 {
		t.Errorf("header sample rate = %d, want 22254", got)
	}
	if got := binary.LittleEndian.Uint32(b[40:]); int(got) != len(samples)*2 {
		t.Errorf("data chunk length = %d, want %d", got, len(samples)*2)
	}
	if got := int16(binary.LittleEndian.Uint16(b[44+2:])); got != 30 {
		t.Errorf("second sample = %d, want 30", got)
	}
}
