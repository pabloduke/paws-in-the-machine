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
	w, eng := guardedWorld(3)

	// Impossible approach refuses with configured prose, no roll.
	if out := eng.Execute("charm guard"); !strings.Contains(out, "immune to cute") {
		t.Fatalf("charm should be refused: %q", out)
	}

	// Failed approach repeats identically (seed 3: sneak fails) and
	// pays no XP.
	first := eng.Execute("sneak past guard")
	if !strings.Contains(first, "sneak-no") {
		t.Fatalf("expected sneak failure with seed 3: %q", first)
	}
	if again := eng.Execute("sneak past guard"); again != first {
		t.Fatalf("identical attempt differed:\n%q\n%q", first, again)
	}
	if w.Level != 1 || w.XP != 0 {
		t.Fatalf("failed checks must not pay XP: level %d, xp %d", w.Level, w.XP)
	}

	// Possible approach succeeds, moves the player, and pays the roll
	// you needed: 18 - 12 = 6 XP (crossing L1->2, leaving 1 XP over).
	out := eng.Execute("parkour guard")
	if !strings.Contains(out, "parkour-ok") || w.Room().ID != "back" {
		t.Fatalf("expected parkour success into back room: %q (room %s)", out, w.Room().ID)
	}
	if !strings.Contains(out, "+6 XP") || w.Level != 2 || w.XP != 1 {
		t.Fatalf("expected +6 XP and level 2: %q (level %d, xp %d)", out, w.Level, w.XP)
	}

	// Once bypassed, the guard stays solved and pays nothing more.
	eng.Execute("out")
	if out := eng.Execute("sneak past guard"); !strings.Contains(out, "ignore you") {
		t.Fatalf("bypassed guard should stay bypassed: %q", out)
	}
	if w.Level != 2 || w.XP != 1 {
		t.Fatalf("bypassed guard must not pay again: level %d, xp %d", w.Level, w.XP)
	}
}

// modWorld builds a fixture around one configurable sneak attempt.
func modWorld(attempt checks.Attempt) (*engine.World, *engine.Engine) {
	w := engine.NewWorld()
	w.Seed = 3
	w.Stats = engine.Stats{Stealth: 10}
	front := engine.NewEntity("front", "Front").With(engine.Exits{})
	back := engine.NewEntity("back", "Back").With(engine.Exits{})
	guard := engine.NewEntity("guard", "a guard").With(checks.Guarded{
		Dest:       "back",
		Approaches: map[checks.Approach]checks.Attempt{checks.Sneak: attempt},
	})
	w.Root.Add(front, back)
	front.Add(guard, w.Player)
	return w, engine.New(w)
}

// Mods shift the roll: a big enough circumstance flips a guaranteed
// failure into a pass, is spent on use, and shrinks the XP payout.
func TestModsShiftTheRoll(t *testing.T) {
	attempt := checks.Attempt{
		Difficulty: 22, Success: "in", Failure: "no",
		Mods: []checks.Mod{{If: []string{"distracted"}, Delta: 25, Consume: "distracted"}},
	}

	// Without the circumstance: seed 3, Stealth 10 fails (see TestXCOMRule).
	w, eng := modWorld(attempt)
	if out := eng.Execute("sneak past guard"); !strings.Contains(out, "no") {
		t.Fatalf("baseline should fail: %q", out)
	}

	// With it: +25 clears any roll, the flag is consumed, and the
	// stacked deck pays the floor award.
	w, eng = modWorld(attempt)
	w.Flags["distracted"] = true
	out := eng.Execute("sneak past guard")
	if !strings.Contains(out, "in") || w.Room().ID != "back" {
		t.Fatalf("modded sneak should pass: %q (room %s)", out, w.Room().ID)
	}
	if !strings.Contains(out, "+1 XP") {
		t.Fatalf("a cheesed check should pay the floor: %q", out)
	}
	if w.Flags["distracted"] {
		t.Fatalf("the distraction must be consumed by the roll")
	}
}

// A distraction is spent even when the attempt still fails.
func TestConsumeSpentOnFailure(t *testing.T) {
	w, eng := modWorld(checks.Attempt{
		Difficulty: 22, Success: "in", Failure: "no",
		Mods: []checks.Mod{{If: []string{"distracted"}, Delta: -25, Consume: "distracted"}},
	})
	w.Flags["distracted"] = true
	if out := eng.Execute("sneak past guard"); !strings.Contains(out, "no") {
		t.Fatalf("-25 must fail: %q", out)
	}
	if w.Flags["distracted"] {
		t.Fatalf("a spent distraction stays spent, pass or fail")
	}
}

// Failure writes to the blackboard (OnFail); success does not.
func TestOnFailForks(t *testing.T) {
	attempt := checks.Attempt{
		Difficulty: 22, Success: "in", Failure: "no",
		OnFail: []string{"alerted"},
		Mods:   []checks.Mod{{If: []string{"boost"}, Delta: 25}},
	}

	w, eng := modWorld(attempt)
	eng.Execute("sneak past guard") // seed 3: fails
	if !w.Flags["alerted"] {
		t.Fatalf("failure should set its OnFail flags")
	}

	w, eng = modWorld(attempt)
	w.Flags["boost"] = true
	eng.Execute("sneak past guard") // +25: passes
	if w.Flags["alerted"] {
		t.Fatalf("success must not set OnFail flags")
	}
}

// A sealed approach refuses outright: no roll, no XP, no flags.
func TestSealedApproach(t *testing.T) {
	w, eng := modWorld(checks.Attempt{
		Difficulty: 22, Success: "in", Failure: "no",
		Mods:       []checks.Mod{{If: []string{"boost"}, Delta: 25, Consume: "boost"}},
		Seals:      "burned",
		SealedText: "that route is gone",
	})
	w.Flags["burned"] = true
	w.Flags["boost"] = true // would guarantee a pass if it rolled

	out := eng.Execute("sneak past guard")
	if !strings.Contains(out, "that route is gone") {
		t.Fatalf("sealed approach should refuse with its prose: %q", out)
	}
	if w.Room().ID != "front" || w.XP != 0 || !w.Flags["boost"] {
		t.Fatalf("sealed refusal must not roll, move, pay, or consume")
	}
}

// watchedWorld builds a fixture where the gate and the eyes differ:
// a back door guarded by a sneak attempt, watched by a separate dog
// entity that starts in the room with the player.
func watchedWorld(g checks.Guarded) (*engine.World, *engine.Engine) {
	w := engine.NewWorld()
	w.Seed = 3
	w.Stats = engine.Stats{Stealth: 10}
	front := engine.NewEntity("front", "Front").With(engine.Exits{})
	back := engine.NewEntity("back", "Back").With(engine.Exits{})
	dog := engine.NewEntity("dog", "a dog")
	door := engine.NewEntity("door", "the back door").With(g)
	w.Root.Add(front, back)
	front.Add(door, dog, w.Player)
	return w, engine.New(w)
}

// An absent watcher cannot observe: moving the dog out of the room
// applies the Unwatched delta and flips a guaranteed failure.
func TestUnwatchedWhenWatcherAbsent(t *testing.T) {
	gate := checks.Guarded{
		Dest:      "back",
		Watcher:   "dog",
		Unwatched: 25,
		Approaches: map[checks.Approach]checks.Attempt{
			checks.Sneak: {Difficulty: 31, Success: "in", Failure: "no"},
		},
	}

	// Dog in the room: difficulty 31 is out of reach at Stealth 10.
	w, eng := watchedWorld(gate)
	if out := eng.Execute("sneak past door"); !strings.Contains(out, "no") {
		t.Fatalf("watched door should fail: %q", out)
	}

	// Dog elsewhere: unwatched, +25 clears any roll.
	w, eng = watchedWorld(gate)
	w.FindID("back").Add(w.FindID("dog"))
	out := eng.Execute("sneak past door")
	if !strings.Contains(out, "in") || w.Room().ID != "back" {
		t.Fatalf("unwatched door should pass: %q (room %s)", out, w.Room().ID)
	}
}

// An Oblivious circumstance blinds a present watcher; perception only
// reads flags, so the circumstance is not consumed by the roll.
func TestObliviousBlindsPresentWatcher(t *testing.T) {
	gate := checks.Guarded{
		Dest:      "back",
		Watcher:   "dog",
		Oblivious: []checks.Cond{{If: []string{"lured"}}},
		Unwatched: 25,
		Approaches: map[checks.Approach]checks.Attempt{
			checks.Sneak: {Difficulty: 31, Success: "in", Failure: "no"},
		},
	}

	w, eng := watchedWorld(gate)
	w.Flags["lured"] = true
	out := eng.Execute("sneak past door")
	if !strings.Contains(out, "in") || w.Room().ID != "back" {
		t.Fatalf("oblivious watcher should not stop the sneak: %q", out)
	}
	if !w.Flags["lured"] {
		t.Fatalf("perception must not consume flags — observing never mutates")
	}
}

// Unwatched zero means perception doesn't participate: an absent
// watcher changes nothing.
func TestUnwatchedZeroIgnoresPerception(t *testing.T) {
	gate := checks.Guarded{
		Dest:    "back",
		Watcher: "dog",
		Approaches: map[checks.Approach]checks.Attempt{
			checks.Sneak: {Difficulty: 31, Success: "in", Failure: "no"},
		},
	}
	w, eng := watchedWorld(gate)
	w.FindID("back").Add(w.FindID("dog"))
	if out := eng.Execute("sneak past door"); !strings.Contains(out, "no") {
		t.Fatalf("with Unwatched 0 an absent watcher must change nothing: %q", out)
	}
}

// TestAwardFloor: a check you outclass still pays 1 XP, never 0 or
// negative.
func TestAwardFloor(t *testing.T) {
	w := engine.NewWorld()
	w.Seed = 3
	w.Stats = engine.Stats{Agility: 12}

	front := engine.NewEntity("front", "Front").With(engine.Exits{})
	back := engine.NewEntity("back", "Back").With(engine.Exits{})
	pushover := engine.NewEntity("pushover", "a pushover").With(checks.Guarded{
		Dest: "back",
		Approaches: map[checks.Approach]checks.Attempt{
			checks.Parkour: {Difficulty: 5, Success: "over-it", Failure: "no"},
		},
	})
	w.Root.Add(front, back)
	front.Add(pushover, w.Player)

	out := engine.New(w).Execute("parkour pushover")
	if !strings.Contains(out, "+1 XP") {
		t.Fatalf("outclassed check should floor at +1 XP: %q", out)
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
