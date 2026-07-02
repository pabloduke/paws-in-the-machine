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

// NewWorld constructs the world. Milestone one: a single stub room
// proving the core loop; real content replaces this.
func NewWorld() *engine.World {
	w := engine.NewWorld()

	den := engine.NewEntity("den", "The Den").With(
		engine.Description{Text: "Buddy's squat. Placeholder walls, placeholder rain."},
		engine.Exits{
			Dirs:    map[string]string{},
			Blocked: "Nowhere to go yet. The world ends at these walls.",
		},
	)

	shard := engine.NewEntity("shard", "a data-shard", "shard", "chip").With(
		engine.Description{Text: "Matte black, cold to the touch. Placeholder secrets."},
		engine.Portable{},
	)

	// Example of bespoke behavior via the closure escape hatch:
	// (when a second entity needs the same behavior, promote it to a
	// reusable component type)
	//
	//	deck := engine.NewEntity("deck", "the deck", "rig").With(
	//	    engine.Description{Text: "..."},
	//	    engine.On{Verb: "use", Do: func(w *engine.World) string {
	//	        if !w.Flags["heard_whisper"] {
	//	            w.Flags["heard_whisper"] = true
	//	            return "...they buried the sun..."
	//	        }
	//	        return "The whisper is still down there."
	//	    }},
	//	)

	w.Root.Add(den)
	den.Add(shard, w.Player)
	return w
}
