// Package game holds the content of Paws in the Machine: the entity
// tree, descriptions, and story triggers. All mechanics live in the
// engine package; this package only declares.
//
// House rule: this package contains only literals, component values,
// and small hook funcs — never engine changes. A behavior needed by
// more than one entity graduates into a reusable component.
package game

import (
	"github.com/pabloduke/paws-in-the-machine/internal/engine"
	"github.com/pabloduke/paws-in-the-machine/internal/systems/checks"
	"github.com/pabloduke/paws-in-the-machine/internal/systems/hubs"
)

// Intro is shown once when the session starts.
const Intro = `PAWS IN THE MACHINE

(Intro text goes here. Type "help" for commands.)`

// NewWorld constructs the world: Buddy's lair and the coffee shop,
// connected north/south. All prose is placeholder.
func NewWorld() *engine.World {
	w := engine.NewWorld()

	// --- Buddy's Lair -------------------------------------------------

	lair := engine.NewEntity("lair", "Buddy's Lair").With(
		engine.Description{Text: "(Placeholder) The lair. Rain on the window, " +
			"neon through the blinds."},
		engine.Exits{
			Dirs:    map[string]string{"north": "coffeeshop"},
			Blocked: "(Placeholder) The only way out is north, past the noodle bar.",
		},
	)

	deck := engine.NewEntity("deck", "the deck", "rig", "computer", "terminal").With(
		engine.Description{Fn: func(w *engine.World) string {
			if w.Flags["heard_whisper"] {
				return "(Placeholder) The deck. The cursor blinks, patient. " +
					"You can still feel the warmth in the dead code."
			}
			return "(Placeholder) The deck. Scavenged, soldered, faithful."
		}},
		engine.On{Verb: "use", Do: func(w *engine.World) string {
			if !w.Flags["heard_whisper"] {
				w.Flags["heard_whisper"] = true
				return "(Placeholder) You jack in. Down in the dead code, " +
					"something whispers: ...they buried the sun..."
			}
			return "(Placeholder) You jack in again. The whisper is still " +
				"down there, circling."
		}},
	)

	shelf := engine.NewEntity("shelf", "a high shelf", "shelf").With(
		engine.Description{Text: "(Placeholder) A high shelf. Your kind of territory."},
	)

	mug := engine.NewEntity("mug", "a chipped mug", "mug", "cup").With(
		engine.Description{Text: "(Placeholder) A mug, near the edge of the " +
			"shelf. Very near the edge."},
		engine.Portable{},
		engine.On{Verb: "knock", Do: func(w *engine.World) string {
			if w.Flags["mug_down"] {
				return "(Placeholder) The mug is already on the floor."
			}
			w.Flags["mug_down"] = true
			return "(Placeholder) One deliberate paw. The mug tips, hangs, " +
				"shatters. Focus restored."
		}},
	)

	shard := engine.NewEntity("shard", "a data-shard", "shard", "chip").With(
		engine.Description{Text: "(Placeholder) Matte black, colder than it " +
			"should be."},
		engine.Portable{},
	)

	// --- The Coffee Shop ----------------------------------------------

	coffeeshop := engine.NewEntity("coffeeshop", "The Coffee Shop").With(
		engine.Description{Text: "(Placeholder) The coffee shop down the block. " +
			"Steam, low talk, a door that never quite shuts."},
		engine.Exits{
			Dirs:    map[string]string{"south": "lair"},
			Blocked: "(Placeholder) Nothing that way but rain. The lair is south.",
		},
	)

	counter := engine.NewEntity("counter", "the counter").With(
		engine.Description{Text: "(Placeholder) A long counter, wiped clean a " +
			"thousand times."},
	)

	machine := engine.NewEntity("machine", "the espresso machine", "espresso", "espresso machine").With(
		engine.Description{Text: "(Placeholder) A chrome espresso machine, " +
			"hissing like something alive."},
		engine.On{Verb: "knock", Do: func(w *engine.World) string {
			return "(Placeholder) You put a paw on the chrome. It is heavy, " +
				"hot, and unimpressed."
		}},
	)

	laptop := engine.NewEntity("laptop", "a regular's laptop", "laptop").With(
		engine.Description{Text: "(Placeholder) A laptop left open at a corner " +
			"table. Its owner is in the restroom. Interesting."},
	)

	hound := engine.NewEntity("hound", "a corpo hound", "hound", "dog", "guard").With(
		engine.Description{Text: "(Placeholder) A corpo security hound parked in " +
			"front of the back room door. Ears up. Dogs take this job " +
			"personally."},
		checks.Guarded{
			Dest: "backroom",
			Approaches: map[checks.Approach]checks.Attempt{
				checks.Sneak: {
					Difficulty: 22,
					Success: "(Placeholder) You pour yourself along the " +
						"skirting board, one shadow among many.",
					Failure: "(Placeholder) A low growl. The hound's eyes " +
						"track you before you've taken two steps. Not " +
						"like this — something has to change.",
				},
				checks.Parkour: {
					Difficulty: 18,
					Success: "(Placeholder) Counter, shelf, hanging lamp, " +
						"transom window. The hound guards a door; you " +
						"were never going to use the door.",
					Failure: "(Placeholder) You misjudge the counter's " +
						"grease factor and abort the run. The hound " +
						"huffs. Not like this — something has to change.",
				},
			},
			Refusals: map[checks.Approach]string{
				checks.Charm: "(Placeholder) You deploy the adopt-me eyes. " +
					"The hound stares through them into middle distance. " +
					"Dogs are immune to cute. It's why the corpos hire them.",
			},
		},
	)

	// --- The Back Room --------------------------------------------------

	backroom := engine.NewEntity("backroom", "The Back Room").With(
		engine.Description{Text: "(Placeholder) Storage, a humming server " +
			"rack that has no business in a coffee shop, and the smell " +
			"of secrets."},
		engine.Exits{
			Dirs:    map[string]string{"north": "coffeeshop"},
			Blocked: "(Placeholder) One way in, one way out: north.",
		},
	)

	// --- The Plaza (hub) -------------------------------------------------

	plazaSquare := engine.NewEntity("plaza_square", "Plaza Square").With(
		engine.Description{Text: "(Placeholder) The Plaza. Ad-drones wheeling " +
			"under the dome glow, crowds that part around you without " +
			"noticing you. Nothing here needs a cat. Yet."},
		engine.Exits{
			Dirs:    map[string]string{"east": "arcade"},
			Blocked: "(Placeholder) Crowds and chrome in every other direction.",
		},
	)

	arcade := engine.NewEntity("arcade", "The Shuttered Arcade").With(
		engine.Description{Text: "(Placeholder) A dead arcade, cabinets under " +
			"dust sheets. Something hums in the back wall that shouldn't."},
		engine.Exits{
			Dirs:    map[string]string{"west": "plaza_square"},
			Blocked: "(Placeholder) The square is back west.",
		},
	)

	// --- Assemble the tree ---------------------------------------------

	neighborhood := engine.NewEntity("neighborhood", "The Neighborhood").With(
		hubs.Hub{Entry: "lair"},
	)
	plaza := engine.NewEntity("plaza", "The Plaza").With(
		hubs.Hub{Entry: "plaza_square"},
	)

	w.Root.Add(neighborhood, plaza)
	neighborhood.Add(lair, coffeeshop, backroom)
	plaza.Add(plazaSquare, arcade)
	lair.Add(deck, shelf, shard, w.Player)
	shelf.Add(mug)
	coffeeshop.Add(counter, machine, laptop, hound)

	// Starting numbers. With the pinned seed below, the hound demos the
	// full loop: sneak fails at Stealth 10 (and would pass at 12 —
	// growth flips it), parkour clears, charm is refused outright.
	w.Stats = engine.Stats{Stealth: 10, Agility: 12, Charm: 8}
	w.XP = 0
	// Pinned while tuning the feel; remove to randomize per new game.
	w.Seed = 3

	// Idioms a player will reach for that the generic parser can't guess.
	w.Rewrites["jack in"] = "use deck"

	return w
}
