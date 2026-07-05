package ui

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/pabloduke/paws-in-the-machine/internal/game"
)

func TestRainFieldIsDeterministic(t *testing.T) {
	a := strings.Join(rainField("lair", 4, 60, 8), "\n")
	b := strings.Join(rainField("lair", 4, 60, 8), "\n")
	if a != b {
		t.Fatalf("same args must draw the same frame")
	}
	if a == strings.Join(rainField("lair", 5, 60, 8), "\n") {
		t.Fatalf("a new phase should move the rain")
	}
	if a == strings.Join(rainField("coffeeshop", 4, 60, 8), "\n") {
		t.Fatalf("each room gets its own rain")
	}
}

// headRows finds the drop heads in a frame: column -> rows with '╱',
// styling stripped.
func headRows(field []string) map[int][]int {
	ansi := regexp.MustCompile("\x1b\\[[0-9;]*m")
	heads := map[int][]int{}
	for r, line := range field {
		for c, ch := range []rune(ansi.ReplaceAllString(line, "")) {
			if ch == '╱' {
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
		now := headRows(rainField("lair", p, w, h))
		next := headRows(rainField("lair", p+1, w, h))
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
			a := strings.Join(rainField("lair", p, w, h), "\n")
			b := strings.Join(rainField("lair", p+rainMaxSpeed, w, h), "\n")
			if a == b {
				t.Fatalf("phase %d: nothing moved across %d frames", p, rainMaxSpeed)
			}
		}
	}
}

func TestRainStaysGentle(t *testing.T) {
	field := strings.Join(rainField("plaza", 9, 70, 10), "\n")
	drops := strings.Count(field, "╱")
	if drops == 0 {
		t.Fatalf("rain should be visible")
	}
	if max := 70 / rainColsPerDrop; drops > max {
		t.Fatalf("too heavy: %d heads, want <= %d", drops, max)
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
