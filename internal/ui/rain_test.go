package ui

import (
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

// The falling property: everything visible in one frame appears one
// row lower in the next (new drops enter at row 0, exempt).
func TestRainFalls(t *testing.T) {
	const h = 8
	now := rainField("lair", 4, 60, h)
	next := rainField("lair", 5, 60, h)
	for r := 0; r < h-1; r++ {
		if now[r] != next[r+1] {
			t.Fatalf("row %d should fall to row %d:\n%q\n%q", r, r+1, now[r], next[r+1])
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
