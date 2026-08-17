package game_test

import (
	"testing"

	"github.com/pabloduke/paws-in-the-machine/internal/engine"
	"github.com/pabloduke/paws-in-the-machine/internal/game"
)

// The retrofit must be behaviour-preserving: charted rooms derive
// exactly the exits they used to declare by hand, and the authored
// prose survives.
func TestChartedRoomsKeepTheirExits(t *testing.T) {
	w := game.NewWorld()
	if len(w.Pending) != 0 {
		t.Fatalf("charting the world should report no content bugs: %v", w.Pending)
	}
	for _, tc := range []struct{ room, dir, want string }{
		{"lair", "north", "coffeeshop"},
		{"coffeeshop", "south", "lair"},
		{"plaza_square", "east", "arcade"},
		{"arcade", "west", "plaza_square"},
	} {
		e := w.FindID(tc.room)
		x, ok := engine.Part[engine.Exits](e)
		if !ok {
			t.Fatalf("%s should carry exits", tc.room)
		}
		if got := x.Dirs[tc.dir]; got != tc.want {
			t.Errorf("%s %s: got %q want %q", tc.room, tc.dir, got, tc.want)
		}
		if x.Blocked == "" {
			t.Errorf("%s lost its authored Blocked prose", tc.room)
		}
		if len(x.Dirs) != 1 {
			t.Errorf("%s gained an exit it never had: %v", tc.room, x.Dirs)
		}
	}
}
