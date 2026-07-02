package checks_test

import (
	"strings"
	"testing"

	"github.com/pabloduke/paws-in-the-machine/internal/engine"
	"github.com/pabloduke/paws-in-the-machine/internal/systems/checks"
)

// guardedWorld builds a minimal fixture: two rooms and a guard whose
// matrix allows sneak and parkour but not charm.
func guardedWorld(seed int64) (*engine.World, *engine.Engine) {
	w := engine.NewWorld()
	w.Seed = seed
	w.Stats = engine.Stats{Stealth: 10, Agility: 12, Charm: 8}

	front := engine.NewEntity("front", "Front").With(engine.Exits{})
	back := engine.NewEntity("back", "Back").With(
		engine.Exits{Dirs: map[string]string{"out": "front"}},
	)
	guard := engine.NewEntity("guard", "a guard").With(checks.Guarded{
		Dest: "back",
		Approaches: map[checks.Approach]checks.Attempt{
			checks.Sneak:   {Difficulty: 22, Success: "sneak-ok", Failure: "sneak-no"},
			checks.Parkour: {Difficulty: 18, Success: "parkour-ok", Failure: "parkour-no"},
		},
		Refusals: map[checks.Approach]string{checks.Charm: "immune to cute"},
	})
	w.Root.Add(front, back)
	front.Add(guard, w.Player)
	return w, engine.New(w)
}

// TestXCOMRule: identical attempts give identical results, across fresh
// worlds with the same seed; a changed stat may change the outcome.
func TestXCOMRule(t *testing.T) {
	first := checks.Check(mustWorld(3), "hound.sneak", 10, 22)
	for range 5 {
		if got := checks.Check(mustWorld(3), "hound.sneak", 10, 22); got != first {
			t.Fatal("identical attempt changed its result")
		}
	}

	// With seed 3 the tuned demo pattern holds for the game's hound:
	// sneak fails at Stealth 10, passes at 12 (see internal/game).
	if checks.Check(mustWorld(3), "hound.sneak", 10, 22) {
		t.Fatal("expected sneak at Stealth 10 to fail with seed 3")
	}
	if !checks.Check(mustWorld(3), "hound.sneak", 12, 22) {
		t.Fatal("expected sneak at Stealth 12 to pass with seed 3")
	}
}

func mustWorld(seed int64) *engine.World {
	w := engine.NewWorld()
	w.Seed = seed
	return w
}

func TestApproachMatrix(t *testing.T) {
	_, eng := guardedWorld(3)

	// Impossible approach refuses with configured prose, no roll.
	if out := eng.Execute("charm guard"); !strings.Contains(out, "immune to cute") {
		t.Fatalf("charm should be refused: %q", out)
	}

	// Failed approach repeats identically (seed 3: sneak fails).
	first := eng.Execute("sneak past guard")
	if !strings.Contains(first, "sneak-no") {
		t.Fatalf("expected sneak failure with seed 3: %q", first)
	}
	if again := eng.Execute("sneak past guard"); again != first {
		t.Fatalf("identical attempt differed:\n%q\n%q", first, again)
	}

	// Possible approach succeeds and moves the player.
	out := eng.Execute("parkour guard")
	if !strings.Contains(out, "parkour-ok") || !strings.Contains(out, "Back") {
		t.Fatalf("expected parkour success into Back: %q", out)
	}

	// Once bypassed, the guard stays solved.
	eng.Execute("out")
	if out := eng.Execute("sneak past guard"); !strings.Contains(out, "ignore you") {
		t.Fatalf("bypassed guard should stay bypassed: %q", out)
	}
}

func TestApproachVerbFallbacks(t *testing.T) {
	w := engine.NewWorld()
	room := engine.NewEntity("room", "Room").With(engine.Exits{})
	thing := engine.NewEntity("thing", "a thing")
	w.Root.Add(room)
	room.Add(thing, w.Player)
	eng := engine.New(w)

	for verb, want := range map[string]string{
		"sneak past thing": "isn't in your way",
		"parkour thing":    "nothing to parkour",
		"charm thing":      "Nothing to gain",
		"stats":            "Stealth 0",
	} {
		if out := eng.Execute(verb); !strings.Contains(out, want) {
			t.Fatalf("%q: got %q, want substring %q", verb, out, want)
		}
	}
}
