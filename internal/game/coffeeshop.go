package game

import (
	"github.com/pabloduke/paws-in-the-machine/internal/engine"
	"github.com/pabloduke/paws-in-the-machine/internal/systems/checks"
	"github.com/pabloduke/paws-in-the-machine/internal/systems/dialogue"
)

// buildCoffeeshop is The Coffee Shop: the barista, her old badge, the
// espresso machine, and the laptop.
func buildCoffeeshop() *engine.Entity {
	coffeeshop := engine.NewEntity("coffeeshop", "The Coffee Shop").With(
		// User-declared 2026-07-10.
		engine.Description{Text: "The coffee shop bustles with people. An " +
			"overworked barista making drinks, taking orders, looks " +
			"exhausted and ragged. Her long blonde hair plastered to her " +
			"face underneath that ugly baseball cap they make her wear. " +
			"Her apron stained with coffee and chocolate. She seems " +
			"distracted and lonely. Her backpack is behind the counter, " +
			"its old Microslop badge clipped to it."},
		engine.Exits{
			Dirs:    map[string]string{"south": "lair"},
			Blocked: "(Placeholder) Nothing that way but rain. The lair is south.",
		},
	)

	counter := engine.NewEntity("counter", "the counter").With(
		engine.Description{Text: "(Placeholder) A long counter, wiped clean a " +
			"thousand times."},
	)

	barista := engine.NewEntity("barista", "the barista", "barista", "human").With(
		engine.Notable{},
		// Her body language remembers first contact (NPC memory, #14):
		// cold if Buddy was cruel, easy once she's been won over.
		engine.Description{Fn: func(w *engine.World) string {
			switch {
			case w.Flags[flagBaristaBurned]:
				return "(Placeholder) The barista keeps the counter between you " +
					"and her, wiping the same spot she already wiped. She " +
					"remembers the stray who knocked her tips to the floor."
			case w.Flags[flagBaristaSoftened]:
				return "(Placeholder) The barista's shoulders drop a notch when " +
					"she clocks the orange. Not a smile — but the closest thing " +
					"she keeps in stock."
			default:
				return "(Placeholder) The barista has the hollow-eyed " +
					"look of someone who has seen too many loyalty apps and not " +
					"enough sunlight."
			}
		}},
		dialogue.Talkable{
			Start: "greeting",
			Nodes: map[string]dialogue.Node{
				"greeting": {
					// NPC memory (#14): the opening line runs warm, cold, or
					// neutral on what she remembers — a pure function of flags
					// (docs/systems/dialogue.md).
					TextFn: func(w *engine.World) string {
						switch {
						case w.Flags[flagBaristaBurned]:
							return "(Placeholder) \"You. Order something or get out, " +
								"orange. I've got a broom with your name on it.\""
						case w.Flags[flagBaristaSoftened]:
							return "(Placeholder) \"Back again? Counter's yours, I " +
								"guess. What now.\""
						default:
							return "(Placeholder) \"That cat again. You lost, orange?\""
						}
					},
					Choices: []dialogue.Choice{
						{
							Text: "Purr and make the counter your kingdom.",
							Require: []dialogue.Requirement{
								dialogue.StatAtLeast("charm", 8),
								dialogue.MissingFlag(flagBaristaSoftened),
							},
							Effects: []dialogue.Effect{
								dialogue.SetFlag(flagBaristaSoftened),
								dialogue.AwardOnce(flagXPBaristaCharm, 3),
							},
							Next: "softened",
						},
						{
							// The cruel fork (NPC memory, #14): a petty cat move
							// she remembers. Only reachable at first contact —
							// once she's warmed to you, or already soured, this
							// is gone. Being cruel closes the invited post-it look,
							// but the stealth route stays open.
							Text: "Hook a claw under her tip jar and send it to the floor. Because you can.",
							Require: []dialogue.Requirement{
								dialogue.MissingFlag(flagBaristaSoftened),
								dialogue.MissingFlag(flagBaristaBurned),
							},
							Effects: []dialogue.Effect{
								dialogue.SetFlag(flagBaristaBurned),
								dialogue.Say("(Placeholder) Coins ring across the tile. " +
									"The barista doesn't chase them. She just looks at " +
									"you the way you'd look at a stain, and files it away."),
							},
							End: true,
						},
						{Text: "Mrow.", Next: "mrow"},
						{Text: "Leave.", End: true},
					},
				},
				"softened": {
					Text: "(Placeholder) \"Fine. One saucer. Don't make it weird.\" " +
						"The barista sets it behind the counter beside her backpack.",
					Choices: []dialogue.Choice{
						{Text: "Accept this tribute.", End: true},
					},
				},
				"mrow": {
					Text: "(Placeholder) \"Yeah, same.\"",
					Choices: []dialogue.Choice{
						{Text: "Leave.", End: true},
					},
				},
			},
		},
	)

	backpack := engine.NewEntity("backpack", "the barista's backpack",
		"backpack", "badge", "microslop badge", "post-it", "postit", "note").With(
		engine.On{Verb: "examine", Do: func(w *engine.World) string {
			switch {
			case w.Flags[flagReadMicroslopBadge]:
				return "(Placeholder) The post-it behind the badge reads: " +
					"employee ID `1008476`, password `apple`."
			case w.Flags[flagBaristaSoftened] && !w.Flags[flagBaristaBurned]:
				w.Flags[flagReadMicroslopBadge] = true
				return "(Placeholder) From the invited saucer, you look at the post-it " +
					"behind the laminated badge: employee ID `1008476`, password `apple`."
			default:
				return "(Placeholder) The post-it is behind the counter and the " +
					"barista is watching. Meow at her or sneak to the post-it."
			}
		}},
		checks.Guarded{
			Watcher: "barista",
			Solved:  flagReadMicroslopBadge,
			Approaches: map[checks.Approach]checks.Attempt{
				checks.Sneak: {
					Difficulty: 22,
					Success: "(Placeholder) You slip behind the counter and look at the post-it " +
						"through the badge laminate: employee ID `1008476`, password `apple`.",
					Failure: "(Placeholder) The barista catches your nose at her backpack. " +
						"The invitation in her face disappears; next time she watches for you.",
					OnSuccess: []string{flagReadMicroslopBadge},
					OnFail:    []string{flagBaristaBurned},
					Mods: []checks.Mod{
						{If: []string{flagBaristaBurned}, Delta: -2},
						{If: []string{flagMugShattered}, Delta: +4, Consume: flagMugShattered},
					},
				},
			},
			Refusals: map[checks.Approach]string{
				checks.Charm:   "(Placeholder) Charm the barista, not her backpack.",
				checks.Parkour: "(Placeholder) There is no acrobatic route through a laminated badge.",
			},
		},
	)

	door := engine.NewEntity("door", "the door").With(
		engine.Description{Text: "(Placeholder) A normal door that never quite " +
			"shuts. Pretty boring, unless you like drafts."},
	)

	machine := engine.NewEntity("machine", "the espresso machine", "espresso", "espresso machine").With(
		engine.Description{Text: "(Placeholder) A chrome espresso machine, " +
			"hissing like something alive."},
		engine.On{Verb: "knock", Do: func(w *engine.World) string {
			if w.Flags[flagMugShattered] {
				return "(Placeholder) The barista is still sweeping up the " +
					"last mug. Knocking again buys you nothing new — yet."
			}
			w.Flags[flagMugShattered] = true
			return "(Placeholder) You put a paw on the chrome. The machine " +
				"is heavy, hot, and unimpressed — but the mug drying on its " +
				"lid isn't. It detonates on the tile, and every head in the " +
				"shop turns toward the sound."
		}},
	)

	laptop := engine.NewEntity("laptop", "a regular's laptop", "laptop").With(
		engine.On{Verb: "examine", Do: func(w *engine.World) string {
			return "(Placeholder) A laptop left open at a corner table. Its " +
				"owner is in the restroom. Interesting." +
				awardOnce(w, flagXPLaptop, 3)
		}},
	)

	coffeeshop.Add(counter, barista, backpack, machine, laptop, door)
	return coffeeshop
}

// coffeeshopJournal: derived entries, visible once their flag is true.
func coffeeshopJournal() []engine.Entry {
	return []engine.Entry{
		{Flag: flagBaristaBurned, Text: "(Placeholder) The barista won't " +
			"forget the tip jar. No invitation behind the counter now; " +
			"the badge takes stealth."},
	}
}
