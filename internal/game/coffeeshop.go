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
		engine.Description{Text: "(Placeholder) The barista has the hollow-eyed " +
			"look of someone who has seen too many loyalty apps and not " +
			"enough sunlight."},
		dialogue.Talkable{
			Start: "greeting",
			Nodes: map[string]dialogue.Node{
				"greeting": {
					Text: "(Placeholder) \"That cat again. You lost, orange?\"",
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
			return "(Placeholder) You put a paw on the chrome. It is heavy, " +
				"hot, and unimpressed."
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

	coffeeshop.Add(counter, barista, machine, laptop, hound, door)
	return coffeeshop
}
