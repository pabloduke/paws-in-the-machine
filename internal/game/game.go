// Package game holds the content of Paws in the Machine: the entity
// tree, descriptions, and story triggers. All mechanics live in the
// engine package; this package only declares.
//
// House rule: this package contains only literals, component values,
// and small hook funcs — never engine changes. A behavior needed by
// more than one entity graduates into a reusable component.
package game

import "github.com/pabloduke/paws-in-the-machine/internal/engine"

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

	// --- Assemble the tree ---------------------------------------------

	w.Root.Add(lair, coffeeshop)
	lair.Add(deck, shelf, shard, w.Player)
	shelf.Add(mug)
	coffeeshop.Add(counter, machine, laptop)
	return w
}
