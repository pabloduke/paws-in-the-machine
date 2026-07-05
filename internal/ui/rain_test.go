package ui

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/pabloduke/paws-in-the-machine/internal/game"
)

func TestRainFieldIsDeterministic(t *testing.T) {
	a := strings.Join(rainField("lair", 4, 60, 8, defaultRainLevel), "\n")
	b := strings.Join(rainField("lair", 4, 60, 8, defaultRainLevel), "\n")
	if a != b {
		t.Fatalf("same args must draw the same frame")
	}
	if a == strings.Join(rainField("lair", 5, 60, 8, defaultRainLevel), "\n") {
		t.Fatalf("a new phase should move the rain")
	}
	if a == strings.Join(rainField("coffeeshop", 4, 60, 8, defaultRainLevel), "\n") {
		t.Fatalf("each room gets its own rain")
	}
}

// headRows finds the rain cells in a frame: column -> rows with '·',
// styling stripped. (Head and trail are all periods; the trail rides
// directly above its head, so the smooth-fall property holds for
// every cell.)
func headRows(field []string) map[int][]int {
	ansi := regexp.MustCompile("\x1b\\[[0-9;]*m")
	heads := map[int][]int{}
	for r, line := range field {
		for c, ch := range []rune(ansi.ReplaceAllString(line, "")) {
			if ch == '·' {
				heads[c] = append(heads[c], r)
			}
		}
	}
	return heads
}

// The smooth-fall property: frame to frame, a drop either holds its
// row or moves down exactly one — never jumps — and no drop stalls
// longer than the slowest speed.
func TestRainFallsSmoothly(t *testing.T) {
	const h, w = 8, 60
	for p := 0; p < 30; p++ {
		now := headRows(rainField("lair", p, w, h, defaultRainLevel))
		next := headRows(rainField("lair", p+1, w, h, defaultRainLevel))
		for col, rows := range now {
			for _, r := range rows {
				ok := false
				for _, nr := range next[col] {
					// hold, one-row fall, or wrap past the bottom
					if nr == r || nr == r+1 || nr < r {
						ok = true
					}
				}
				// A head may also fall off-screen (trail-only frames).
				if !ok && r < h-1 && len(next[col]) > 0 {
					t.Fatalf("phase %d col %d: head at %d jumped to %v", p, col, r, next[col])
				}
			}
		}
		if p%rainMaxSpeed == 0 {
			a := strings.Join(rainField("lair", p, w, h, defaultRainLevel), "\n")
			b := strings.Join(rainField("lair", p+rainMaxSpeed, w, h, defaultRainLevel), "\n")
			if a == b {
				t.Fatalf("phase %d: nothing moved across %d frames", p, rainMaxSpeed)
			}
		}
	}
}

func TestRainStaysGentle(t *testing.T) {
	field := strings.Join(rainField("plaza", 9, 70, 10, defaultRainLevel), "\n")
	drops := strings.Count(field, "·")
	if drops == 0 {
		t.Fatalf("rain should be visible")
	}
	if max := 70 / rainColsPerDrop * (rainTrailLen + 1); drops > max {
		t.Fatalf("too heavy: %d cells, want <= %d", drops, max)
	}
}

// '[' and ']' dial the rain's opacity; level 0 is invisible. The
// keys are a view preference and must never reach the prompt.
func TestRainOpacityKeys(t *testing.T) {
	mod := newSized(game.NewWorld())

	deflt := strings.Join(rainField("lair", 4, 60, 8, defaultRainLevel), "\n")
	dim := strings.Join(rainField("lair", 4, 60, 8, defaultRainLevel-1), "\n")
	if deflt == dim {
		t.Fatalf("levels should render differently")
	}
	if rainField("lair", 4, 60, 8, 0) != nil {
		t.Fatalf("level 0 must render nothing")
	}

	// Dim past the floor: lands on 0 and stays.
	for i := 0; i < maxRainLevel+2; i++ {
		mod, _ = mod.Update(kr('['))
	}
	if lvl := mod.(Model).rainLevel; lvl != 0 {
		t.Fatalf("'[' should bottom out at 0, got %d", lvl)
	}
	// Brighten past the ceiling: clamps at max.
	for i := 0; i < maxRainLevel+2; i++ {
		mod, _ = mod.Update(kr(']'))
	}
	if lvl := mod.(Model).rainLevel; lvl != maxRainLevel {
		t.Fatalf("']' should top out at %d, got %d", maxRainLevel, lvl)
	}
	if v := mod.(Model).input.Value(); v != "" {
		t.Fatalf("opacity keys leaked into the prompt: %q", v)
	}
}

// The timer ruling: a tick advances the render phase, reschedules,
// and touches nothing else — World never sees weather.
func TestRainTickIsPresentationOnly(t *testing.T) {
	w := game.NewWorld()
	mod := newSized(w)
	before := len(w.Flags)

	next, cmd := mod.Update(rainTick(time.Now()))
	if next.(Model).phase != mod.(Model).phase+1 {
		t.Fatalf("tick should advance the render phase")
	}
	if cmd == nil {
		t.Fatalf("tick must reschedule the ticker")
	}
	if len(w.Flags) != before || w.Room().ID != "lair" {
		t.Fatalf("tick must not touch the world")
	}
}
