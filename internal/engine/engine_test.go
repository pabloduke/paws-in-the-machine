package engine_test

import (
	"strings"
	"testing"

	"github.com/pabloduke/paws-in-the-machine/internal/engine"
)

// testWorld builds a small fixture tree exercising the engine contract:
// two rooms, a portable item, a scenery item, a nested item (composite
// scope), and a component using the closure escape hatch.
func testWorld() *engine.World {
	w := engine.NewWorld()

	lab := engine.NewEntity("lab", "The Lab").With(
		engine.Description{Text: "A bare test chamber."},
		engine.Exits{Dirs: map[string]string{"north": "hall"}},
	)
	hall := engine.NewEntity("hall", "The Hall").With(
		engine.Description{Fn: func(w *engine.World) string {
			if w.Flags["lever_pulled"] {
				return "The hall hums."
			}
			return "The hall is silent."
		}},
		engine.Exits{Dirs: map[string]string{"south": "lab"}, Blocked: "A wall of test assertions."},
	)

	coin := engine.NewEntity("coin", "a coin").With(
		engine.Description{Text: "A test coin."},
		engine.Portable{},
	)
	shelf := engine.NewEntity("shelf", "a shelf").With(
		engine.Description{Text: "High up. Your kind of territory."},
	)
	lever := engine.NewEntity("lever", "a lever").With(
		engine.On{Verb: "use", Do: func(w *engine.World) string {
			w.Flags["lever_pulled"] = true
			return "Clunk."
		}},
	)

	w.Root.Add(lab, hall)
	lab.Add(shelf, lever, w.Player)
	shelf.Add(coin) // nested: reachable via composite scope
	return w
}

// TestCoreLoop drives command -> parse -> dispatch -> world update ->
// output through the fixture.
func TestCoreLoop(t *testing.T) {
	eng := engine.New(testWorld())

	steps := []struct {
		input string
		want  string // substring expected in the output
	}{
		{"look", "The Lab"},
		{"x shelf", "your kind of territory"},
		{"take the coin", "You take"},       // nested in shelf, noise word stripped
		{"i", "coin"},                       // inventory = player contents
		{"take shelf", "isn't going"},       // no Portable component
		{"use lever", "Clunk."},             // On escape-hatch component
		{"north", "The Hall"},               // bare direction
		{"look", "The hall hums."},          // Description.Fn sees flag
		{"east", "wall of test assertions"}, // Exits.Blocked
		{"drop coin", "You drop"},           // carried across rooms
		{"take coin", "You take"},           // now from the hall floor
		{"x me", "nothing special"},         // player is an entity, default examine
		{"dance", `don't know how to "dance"`},
	}

	for _, step := range steps {
		out := eng.Execute(step.input)
		if !strings.Contains(strings.ToLower(out), strings.ToLower(step.want)) {
			t.Fatalf("input %q: output %q does not contain %q", step.input, out, step.want)
		}
	}

	eng.Execute("quit")
	if !eng.World.Quitting() {
		t.Fatal("quit did not end the session")
	}
}

// TestDispatchOrder proves attachment order wins and handled=false
// falls through to the default.
func TestDispatchOrder(t *testing.T) {
	w := engine.NewWorld()
	room := engine.NewEntity("room", "Room").With(engine.Exits{})
	thing := engine.NewEntity("thing", "a thing").With(
		engine.On{Verb: "use", Do: func(*engine.World) string { return "first" }},
		engine.On{Verb: "use", Do: func(*engine.World) string { return "second" }},
	)
	w.Root.Add(room)
	room.Add(thing, w.Player)
	eng := engine.New(w)

	if out := eng.Execute("use thing"); out != "first" {
		t.Fatalf("attachment order not respected: %q", out)
	}
	if out := eng.Execute("knock thing"); !strings.Contains(out, "speculative shove") {
		t.Fatalf("unhandled verb should fall through to default: %q", out)
	}
}
