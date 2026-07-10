package game

import (
	"github.com/pabloduke/paws-in-the-machine/internal/engine"
	"github.com/pabloduke/paws-in-the-machine/internal/systems/checks"
	"github.com/pabloduke/paws-in-the-machine/internal/systems/dialogue"
)

// buildCoffeeshop is The Coffee Shop: the barista, the espresso
// machine, the laptop, and the corpo hound guarding the back room.
func buildCoffeeshop() *engine.Entity {
	coffeeshop := engine.NewEntity("coffeeshop", "The Coffee Shop").With(
		engine.Description{Text: "(Placeholder) The coffee shop down the block. " +
			"Steam, low talk, a door that never quite shuts. The barista " +
			"works the counter like it owes them money."},
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
		// Position as a function of state (docs/systems/presence.md):
		// once the shard is out in the open, she works the back room.
		engine.Placed{Fn: func(w *engine.World) string {
			if w.Flags[flagBaristaSawShard] {
				return "backroom"
			}
			return "coffeeshop"
		}},
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
							// is gone. It never touches the apple reveal below,
							// so being cruel costs a favor and her goodwill, not
							// the critical path (failure is a fork, not a wall).
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
						{
							Text: "Nudge the data-shard into view.",
							Require: []dialogue.Requirement{
								dialogue.HasItem("shard"),
								dialogue.MissingFlag(flagBaristaSawShard),
							},
							Effects: []dialogue.Effect{
								dialogue.SetFlag(flagBaristaSawShard),
								dialogue.Say("(Placeholder) The barista's eyes flick " +
									"to the shard, then away from the cameras. " +
									"\"Not here. Back room's safer, if you can " +
									"get past the dog.\""),
								dialogue.AwardOnce(flagXPBaristaShard, 2),
							},
							End: true,
						},
						{
							// A cat can't ask (user ruling 2026-07-09) — but a cat
						// can demand. The reveal is her venting at a cat who
						// won't stop meowing at the right thing.
						Text: "Plant yourself in front of her Microslop lanyard and meow. Insist. Cats always get answers.",
							Require: []dialogue.Requirement{
								dialogue.Flag(flagBaristaSoftened),
								dialogue.MissingFlag(flagKnowsMicroslopPassword),
							},
							Effects: []dialogue.Effect{
								dialogue.SetFlag(flagKnowsMicroslopPassword),
								dialogue.Say("(Placeholder) The barista's mouth goes flat. " +
									"\"Microslop fired me for flagging a leak, then reset " +
									"every contractor box to the same insult of a password: apple. " +
									"If you're going there, make it hurt.\""),
							},
							End: true,
						},
						{
							Text: "Stare at the hound, then at the barista. " +
								"Your meaning is clear: that dog has too much " +
								"free time.",
							Require: []dialogue.Requirement{
								dialogue.Flag(flagBaristaSoftened),
								// She won't do a cruel stray any favors — the lure
								// closes if she's burned. A fork, not a wall: the
								// hound still yields to sneak or parkour.
								dialogue.MissingFlag(flagBaristaBurned),
								dialogue.MissingFlag(flagHoundLured),
							},
							Effects: []dialogue.Effect{
								dialogue.SetFlag(flagHoundLured),
								dialogue.Say("(Placeholder) The barista sighs, " +
									"digs a strip of jerky from under the " +
									"counter, and whistles. The hound's " +
									"professionalism lasts half a second. It " +
									"parks itself at the counter, nose down, " +
									"the back door forgotten."),
								dialogue.AwardOnce(flagXPHoundLure, 2),
							},
							End: true,
						},
						{Text: "Mrow.", Next: "mrow"},
						{Text: "Leave.", End: true},
					},
				},
				"softened": {
					Text: "(Placeholder) \"Fine. One saucer. Don't make it weird.\" " +
						"The barista slides a cap of cream under the counter lip.",
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

	hound := engine.NewEntity("hound", "a corpo hound", "hound", "dog", "guard").With(
		engine.Notable{},
		// Prose telegraphs perception state; the numbers stay hidden
		// (docs/systems/stealth.md).
		engine.Description{Fn: func(w *engine.World) string {
			if w.Flags[flagHoundLured] {
				return "(Placeholder) The corpo hound is parked at the " +
					"counter, nose deep in a strip of jerky. The back " +
					"door has never been less interesting to anyone."
			}
			return "(Placeholder) A corpo security hound parked in " +
				"front of the back room door. Ears up. Dogs take this job " +
				"personally."
		}},
		checks.Guarded{
			Dest: "backroom",
			// Snack time blinds the watcher: attempts roll with the
			// unwatched bonus while the lure holds (perception reads
			// flags, never spends them).
			Oblivious: []checks.Cond{{If: []string{flagHoundLured}}},
			Unwatched: 10,
			Approaches: map[checks.Approach]checks.Attempt{
				checks.Sneak: {
					Difficulty: 22,
					Success: "(Placeholder) You pour yourself along the " +
						"skirting board, one shadow among many.",
					Failure: "(Placeholder) A low growl. The hound's eyes " +
						"track you before you've taken two steps. Not " +
						"like this — something has to change.",
					// Failure alerts the hound (dogs escalate); the mug
					// distraction is spent by the roll it covers. With
					// the pinned seed the demo loop is: fail at 10,
					// alerted −2, mug +4 → 12 clears it.
					OnFail: []string{flagHoundAlerted},
					Mods: []checks.Mod{
						{If: []string{flagHoundAlerted}, Delta: -2},
						{If: []string{flagMugShattered}, Delta: +4,
							Consume: flagMugShattered},
					},
				},
				checks.Parkour: {
					Difficulty: 18,
					Success: "(Placeholder) Counter, shelf, hanging lamp, " +
						"transom window. The hound guards a door; you " +
						"were never going to use the door.",
					Failure: "(Placeholder) You misjudge the counter's " +
						"grease factor and abort the run. The hound " +
						"huffs. Not like this — something has to change.",
					OnFail: []string{flagHoundAlerted},
					Mods: []checks.Mod{
						{If: []string{flagHoundAlerted}, Delta: -2},
					},
				},
			},
			Refusals: map[checks.Approach]string{
				checks.Charm: "(Placeholder) You deploy the adopt-me eyes. " +
					"The hound stares through them into middle distance. " +
					"Dogs are immune to cute. It's why the corpos hire them.",
			},
		},
	)

	coffeeshop.Add(counter, barista, machine, laptop, hound, door)
	return coffeeshop
}

// coffeeshopEvents narrates the stage changes presence makes silently
// (docs/systems/presence.md: presence says where, events say what
// you saw).
func coffeeshopEvents() []engine.When {
	return []engine.When{
		{
			Flags: []string{flagBaristaSawShard},
			Enter: "coffeeshop",
			Once:  flagSeenCounterEmpty,
			Do: func(w *engine.World) string {
				return "(Placeholder) The counter stands unmanned, the " +
					"espresso machine hissing to no one. Through the gap " +
					"in the back door: the barista, hunched over the " +
					"server rack."
			},
		},
		// A failed run past the hound is a story state, not a retry
		// gate (docs/systems/stealth.md): the alert lands as a beat,
		// and the same flag stiffens later attempts via a Mod.
		{
			Flags: []string{flagHoundAlerted},
			Once:  flagSeenHoundAlerted,
			Do: func(w *engine.World) string {
				return "(Placeholder) The hound is on its feet now, nose " +
					"working the air, eyes sweeping the floor at cat " +
					"height. Word travels up a corpo dog's leash. You'll " +
					"need to change the situation — or be twice as good."
			},
		},
	}
}

// coffeeshopJournal: derived entries, visible once their flag is true.
func coffeeshopJournal() []engine.Entry {
	return []engine.Entry{
		{Flag: flagHoundAlerted, Text: "(Placeholder) The corpo hound at " +
			"the coffee shop has my scent. A distraction might reset " +
			"the odds."},
		{Flag: flagHoundLured, Text: "(Placeholder) The barista can call " +
			"the hound off with jerky. While it's at the counter, the " +
			"back door is barely watched."},
		{Flag: flagBaristaBurned, Text: "(Placeholder) The barista won't " +
			"forget the tip jar. No favors coming from that counter — " +
			"the hound's mine to handle the hard way."},
	}
}
