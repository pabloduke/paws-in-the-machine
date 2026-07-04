package game

import (
	"github.com/pabloduke/paws-in-the-machine/internal/engine"
	"github.com/pabloduke/paws-in-the-machine/internal/systems/hacking"
)

// buildLair is Buddy's Lair: the deck, the high shelf, the mug, the
// shard, the window. Returns the room with its contents attached;
// world.go places it in the neighborhood and adds the player.
func buildLair() *engine.Entity {
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
			if w.Flags[flagHeardWhisper] {
				return "(Placeholder) The deck. The cursor blinks, patient. " +
					"You can still feel the warmth in the dead code."
			}
			return "(Placeholder) The deck. Scavenged, soldered, faithful."
		}},
		hacking.Deck{
			Net:  starterNet(),
			Host: "deck",
			Objectives: []hacking.Objective{
				{Flag: flagHeardWhisper, Text: "trace the whisper in the dead code"},
				{Flag: flagGotSunFragment, Text: "pull whatever 'sun' data is still out there"},
			},
		},
	)

	shelf := engine.NewEntity("shelf", "a high shelf", "shelf").With(
		engine.Description{Text: "(Placeholder) A high shelf. Your kind of territory."},
	)

	window := engine.NewEntity("window", "the window").With(
		engine.Description{Text: "(Placeholder) Rain streaks the glass. Four " +
			"stories down, the noodle bar's sign flickers. A normal " +
			"window, pretty boring — unless you like watching rain. " +
			"You do."},
	)

	mug := engine.NewEntity("mug", "a chipped mug", "mug", "cup").With(
		engine.Description{Text: "(Placeholder) A mug, near the edge of the " +
			"shelf. Very near the edge."},
		engine.Portable{},
		engine.On{Verb: "knock", Do: func(w *engine.World) string {
			if w.Flags[flagMugDown] {
				return "(Placeholder) The mug is already on the floor."
			}
			w.Flags[flagMugDown] = true
			return "(Placeholder) One deliberate paw. The mug tips, hangs, " +
				"shatters. Focus restored." + awardOnce(w, flagXPMug, 2)
		}},
	)

	shard := engine.NewEntity("shard", "a data-shard", "shard", "chip").With(
		engine.On{Verb: "examine", Do: func(w *engine.World) string {
			if w.Flags[flagHeardWhisper] {
				return "(Placeholder) The glyphs on the shard almost make " +
					"sense now. Same warmth, folded small." +
					awardOnce(w, flagXPShard, 2)
			}
			return "(Placeholder) Matte black, colder than it should be."
		}},
		engine.Portable{},
	)

	lair.Add(deck, shelf, shard, window)
	shelf.Add(mug)
	return lair
}

// lairEvents are the lair's event rules (docs/systems/events.md).
func lairEvents() []engine.When {
	return []engine.When{
		{
			Flags: []string{flagHeardWhisper},
			Once:  flagSeenWhisperReaction,
			Do: func(w *engine.World) string {
				return "(Placeholder) The deck's fans spin down. In the " +
					"quiet after, the lair feels different — like the " +
					"room heard it too."
			},
		},
	}
}

// lairJournal is the journal content for the whisper thread; entries
// surface as their flags come true.
func lairJournal() []engine.Entry {
	return []engine.Entry{
		{Flag: flagHeardWhisper, Text: "(Placeholder) Traced the whisper " +
			"to sunfarm.arc. Someone buried something they call \"the sun\"."},
		{Flag: flagGotSunFragment, Text: "(Placeholder) Pulled a fragment " +
			"of the sun off the archive. The coordinates are smeared by time."},
	}
}
