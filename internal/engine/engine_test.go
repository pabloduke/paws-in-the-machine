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
		{"north", ""},                       // movement returns no description...
		{"look", "The hall hums."},          // ...state renders the new room (Description.Fn sees flag)
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

// TestDisplayNameTypable: whatever label the UI shows must resolve,
// article or not — even when ID and aliases don't cover it.
func TestDisplayNameTypable(t *testing.T) {
	w := engine.NewWorld()
	room := engine.NewEntity("room", "Room").With(engine.Exits{})
	shard := engine.NewEntity("shard1", "a data-shard").With(
		engine.Description{Text: "Cold."},
	)
	w.Root.Add(room)
	room.Add(shard, w.Player)
	eng := engine.New(w)

	for _, input := range []string{"x data-shard", "x a data-shard"} {
		if out := eng.Execute(input); out != "Cold." {
			t.Fatalf("%q should resolve the shard by display name: %q", input, out)
		}
	}
}

// TestVisibilityScope: scope stops at closed containers; opening
// reveals contents to both resolution and the visible list; labels
// reflect true state via Aspect.
func TestVisibilityScope(t *testing.T) {
	w := engine.NewWorld()
	room := engine.NewEntity("room", "Room").With(engine.Exits{})
	drawer := engine.NewEntity("drawer", "a drawer").With(
		engine.Openable{Flag: "open"},
		engine.Aspect{Fn: func(w *engine.World) string {
			if w.Flags["open"] {
				return "an open drawer"
			}
			return "a drawer"
		}},
	)
	ruby := engine.NewEntity("ruby", "a ruby").With(engine.Portable{})
	w.Root.Add(room)
	room.Add(drawer, w.Player)
	drawer.Add(ruby)
	eng := engine.New(w)

	// Closed: the drawer is visible and targetable, the ruby is not.
	if out := eng.Execute("take ruby"); !strings.Contains(out, `don't see any "ruby"`) {
		t.Fatalf("enclosed ruby should be out of scope: %q", out)
	}
	if vis := w.Visible(); len(vis) != 1 || vis[0].ID != "drawer" {
		t.Fatalf("expected only the drawer visible, got %v", vis)
	}
	if name := engine.DisplayName(w, drawer); name != "a drawer" {
		t.Fatalf("closed label: %q", name)
	}

	// Open: the ruby enters scope, the list, and the label updates.
	w.Flags["open"] = true
	if out := eng.Execute("take ruby"); !strings.Contains(out, "You take") {
		t.Fatalf("revealed ruby should be takeable: %q", out)
	}
	if name := engine.DisplayName(w, drawer); name != "an open drawer" {
		t.Fatalf("open label: %q", name)
	}
	// Carried items don't show in the visible list.
	for _, e := range w.Visible() {
		if e.ID == "ruby" {
			t.Fatal("carried ruby should not be in the visible list")
		}
	}
}

// TestObvious: the YOU SEE tier — Portable and Notable entities are
// listed; scenery stays visible/targetable but unlisted.
func TestObvious(t *testing.T) {
	w := engine.NewWorld()
	room := engine.NewEntity("room", "Room").With(engine.Exits{})
	coin := engine.NewEntity("coin", "a coin").With(engine.Portable{})
	npc := engine.NewEntity("npc", "a hound").With(engine.Notable{})
	scenery := engine.NewEntity("door", "a door").With(
		engine.Description{Text: "Pretty boring."},
	)
	w.Root.Add(room)
	room.Add(coin, npc, scenery, w.Player)

	obvious := w.Obvious()
	if len(obvious) != 2 || obvious[0].ID != "coin" || obvious[1].ID != "npc" {
		t.Fatalf("expected coin and npc listed, got %v", obvious)
	}
	// Scenery is unlisted but fully interactive.
	if out := engine.New(w).Execute("x door"); out != "Pretty boring." {
		t.Fatalf("scenery must stay examinable: %q", out)
	}
}

// TestXPAndTraining covers the growth loop: escalating level costs,
// stat points per level, and spending them.
func TestXPAndTraining(t *testing.T) {
	w := engine.NewWorld()

	// 15 XP crosses L1->2 (5) and L2->3 (10) in one award.
	msg := engine.AwardXP(w, 15)
	if w.Level != 3 || w.StatPoints != 2 || w.XP != 0 {
		t.Fatalf("after 15 XP: level %d, points %d, xp %d", w.Level, w.StatPoints, w.XP)
	}
	if !strings.Contains(msg, "LEVEL 3") || !strings.Contains(msg, "2 stat points") {
		t.Fatalf("award message should announce the level-up: %q", msg)
	}

	if out := engine.Train(w, "stealth"); !strings.Contains(out, "0 → 1") {
		t.Fatalf("train should raise stealth: %q", out)
	}
	if w.Stats.Stealth != 1 || w.StatPoints != 1 {
		t.Fatalf("stealth %d, points %d", w.Stats.Stealth, w.StatPoints)
	}
	if out := engine.Train(w, "whiskers"); !strings.Contains(out, "Train what?") {
		t.Fatalf("unknown stat should be refused: %q", out)
	}
	engine.Train(w, "charm")
	if out := engine.Train(w, "charm"); !strings.Contains(out, "No stat points") {
		t.Fatalf("training without points should be refused: %q", out)
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

// The room view re-renders Look every UI frame; with multiple exits,
// unsorted map iteration made the exits line shuffle between frames.
// Look must be a pure, stable function of world state.
func TestLookExitsStable(t *testing.T) {
	w := engine.NewWorld()
	crossroads := engine.NewEntity("crossroads", "The Crossroads").With(
		engine.Exits{Dirs: map[string]string{
			"north": "a", "south": "b", "east": "c", "west": "d",
		}},
	)
	w.Root.Add(crossroads)
	crossroads.Add(w.Player)

	first := engine.Look(w)
	if !strings.Contains(first, "Exits: east, north, south, west") {
		t.Fatalf("exits should render in sorted order: %q", first)
	}
	for i := 0; i < 100; i++ {
		if got := engine.Look(w); got != first {
			t.Fatalf("Look must be stable across renders:\nfirst: %q\ngot:   %q", first, got)
		}
	}
}

// A gated exit is a lock, not an obstacle: shut prose until its flag
// is true, normal movement after — no dice anywhere (contrast
// checks.Guarded, which is for things you attempt).
func TestGatedExit(t *testing.T) {
	w := engine.NewWorld()
	vault := engine.NewEntity("vault", "The Vault")
	foyer := engine.NewEntity("foyer", "The Foyer").With(
		engine.Exits{
			Dirs: map[string]string{"north": "vault"},
			Gated: map[string]engine.Gate{
				"north": {Flag: "vault_unlocked", Shut: "The vault door stays shut."},
			},
		},
	)
	w.Root.Add(foyer, vault)
	foyer.Add(w.Player)
	eng := engine.New(w)

	if out := eng.Execute("north"); out != "The vault door stays shut." || w.Room().ID != "foyer" {
		t.Fatalf("gated exit should refuse with its prose: %q (room %s)", out, w.Room().ID)
	}
	w.Flags["vault_unlocked"] = true
	eng.Execute("north")
	if w.Room().ID != "vault" {
		t.Fatalf("open gate should move normally, got room %s", w.Room().ID)
	}
}
