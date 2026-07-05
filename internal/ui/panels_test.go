package ui

import (
	"strings"
	"testing"
)

// The rain follows the XCOM rule: identical state, identical drizzle;
// a player action (new transcript line) shifts it. No timers ever
// touch it (docs/systems/turns.md).
func TestDrizzleIsAFunctionOfState(t *testing.T) {
	if drizzle("lair", 3, 60) != drizzle("lair", 3, 60) {
		t.Fatalf("same state must draw the same rain")
	}
	if drizzle("lair", 3, 60) == drizzle("lair", 4, 60) {
		t.Fatalf("an action should shift the rain")
	}
	if drizzle("lair", 3, 60) == drizzle("coffeeshop", 3, 60) {
		t.Fatalf("each room gets its own rain")
	}
}

func TestDrizzleStaysSparse(t *testing.T) {
	// Styled output carries ANSI codes; count only the drizzle glyphs.
	line := drizzle("plaza", 7, 70)
	drops := strings.Count(line, "╱") + strings.Count(line, "·") + strings.Count(line, "`")
	if drops == 0 {
		t.Fatalf("drizzle should draw at least one drop")
	}
	if max := 70 / drizzleDensity; drops > max {
		t.Fatalf("drizzle too dense: %d drops, want <= %d", drops, max)
	}
}
