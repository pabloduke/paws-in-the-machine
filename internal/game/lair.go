package game

import (
	"github.com/pabloduke/paws-in-the-machine/internal/engine"
	"github.com/pabloduke/paws-in-the-machine/internal/systems/hacking"
)

// buildLair is Buddy's Lair: the high shelf, the mug, the shard, the
// window — and the deck, which lives here and nowhere else. Hacking
// starts at home (user ruling 2026-07-07): the lair is the base you
// scan from and return to, or there's no point to having one.
//
// It also returns Buddy's carried PDA: a read-only mirror of the same
// net (menus only, no shell) so the field can be scouted without the
// lair losing its point. The net map is built once and shared, so the
// PDA sees exactly what the deck sees — including files copied home.
func buildLair() (*engine.Entity, *engine.Entity) {
	lair := engine.NewEntity("lair", "Buddy's Lair").With(
		engine.Description{Text: "(Placeholder) The lair. Rain on the window, " +
			"neon through the blinds."},
		engine.Exits{
			Dirs:    map[string]string{"north": "coffeeshop"},
			Blocked: "(Placeholder) The only way out is north, past the noodle bar.",
		},
	)

	net := starterNet()

	deck := engine.NewEntity("deck", "the deck", "rig", "computer", "terminal").With(
		engine.Description{Fn: func(w *engine.World) string {
			if w.Flags[flagHeardWhisper] {
				return "(Placeholder) The deck. The cursor blinks, patient. " +
					"You can still feel the warmth in the dead code."
			}
			return "(Placeholder) The deck. Scavenged, soldered, faithful."
		}},
		hacking.Deck{
			Net:  net,
			Host: "deck",
			Objectives: []hacking.Objective{
				{Flag: flagHeardWhisper, Text: "trace the whisper in the dead code"},
				{Flag: flagGotSunFragment, Text: "pull whatever 'sun' data is still out there"},
			},
			// The Resistance's channel (docs/draft.md: Resistance-lite —
			// a mission-giver and a delivery point, nothing more).
			// Messages arrive on flags, never timers.
			Messenger: hacking.Messenger{
				Contact: "resistance",
				Msgs: []hacking.Msg{
					{ID: "welcome",
						Text: "(Placeholder) channel's clean. you're the cat " +
							"the barista keeps feeding, right? she used to be " +
							"one of ours — Microslop badge, before the layoffs. " +
							"start there."},
					{ID: "route_open", When: flagMicroslopRouteOpen,
						Text: "(Placeholder) we see the bridge you patched at " +
							"the coffee shop. good paws. Microslop will answer " +
							"the deck now."},
					{ID: "whisper", When: flagHeardWhisper,
						Text: "(Placeholder) you heard it too, then. the thing " +
							"in the dead code. careful who you tell — most of " +
							"us pretend we didn't."},
					{ID: "notice", When: flagGotSunNotice,
						Text: "(Placeholder) that liability notice on your " +
							"deck is the first paper proof anyone's pulled out " +
							"of Microslop. hold onto it. we'll want it soon."},
				},
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

	pda := engine.NewEntity("pda", "the PDA", "pda", "slab", "handheld").With(
		engine.Description{Text: "(Placeholder) A salvaged pocket slab riding " +
			"in your harness. It mirrors the deck's storage over the " +
			"collar link and sniffs ports, and that's all it does — " +
			"reading glasses, not claws."},
		engine.Portable{},
		hacking.PDA{Net: net, Host: "deck"},
	)

	lair.Add(deck, shelf, shard, window)
	shelf.Add(mug)
	return lair, pda
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
