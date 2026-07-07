package game

import (
	"github.com/pabloduke/paws-in-the-machine/internal/engine"
	"github.com/pabloduke/paws-in-the-machine/internal/systems/checks"
	"github.com/pabloduke/paws-in-the-machine/internal/systems/dialogue"
)

// buildOkuda is Okuda HQ: the first infiltration building, two ways
// in (docs/systems/stealth.md). The front lobby is the charm route —
// the receptionist can simply let a cat in. The alley is the
// stealth/agility route — a fire-escape climb, then a watched
// corridor. Both converge on the records office, whose maintenance
// console opens okuda.grid's port for the deck (issue #16, the first
// overworld↔terminal handshake).
func buildOkuda() (street, lobby, alley, corridor, office *engine.Entity) {
	street = engine.NewEntity("okuda_street", "Okuda HQ — Street").With(
		engine.Description{Text: "(Placeholder) Okuda HQ. A tower with its " +
			"name pried off the facade, which never works: the ghost of " +
			"the letters stains the stone. Lobby doors north, and a " +
			"service alley breathing warm air to the east."},
		engine.Exits{
			Dirs:    map[string]string{"north": "okuda_lobby", "east": "okuda_alley"},
			Blocked: "(Placeholder) Traffic and glass in every other direction.",
		},
	)

	facade := engine.NewEntity("facade", "the scrubbed facade", "tower", "sign", "letters").With(
		engine.Description{Text: "(Placeholder) Someone paid to make this " +
			"building forget its own name. The rain remembers: OKUDA, " +
			"in clean stone where the letters kept it dry."},
	)

	lobby = engine.NewEntity("okuda_lobby", "Okuda HQ — Lobby").With(
		engine.Description{Fn: func(w *engine.World) string {
			base := "(Placeholder) A lobby designed to be forgotten: " +
				"beige, quiet, one desk, one receptionist, one gate " +
				"that opens for badges and nothing else."
			if w.Flags[flagReceptionistSuspicious] {
				return base + "\n\nThe receptionist tracks you now, the " +
					"way people track a problem they've already reported " +
					"once."
			}
			return base
		}},
		engine.Exits{
			Dirs:    map[string]string{"south": "okuda_street"},
			Blocked: "(Placeholder) The badge gate doesn't argue. It just stays shut.",
		},
	)

	flyer := engine.NewEntity("flyer", "a missing-cat flyer", "flyer", "poster").With(
		engine.On{Verb: "examine", Do: func(w *engine.World) string {
			w.Flags[flagSawLobbyFlyer] = true
			return "(Placeholder) Taped to the desk: MISSING — ORANGE TABBY, " +
				"ANSWERS TO NOODLE. REWARD. The photo is blurry, generously " +
				"orange, and — if you squint — practically a self-portrait."
		}},
	)

	receptionist := engine.NewEntity("receptionist", "the receptionist", "receptionist", "desk clerk", "clerk").With(
		engine.Notable{},
		engine.Description{Fn: func(w *engine.World) string {
			if w.Flags[flagReceptionistSuspicious] {
				return "(Placeholder) The receptionist has decided you are " +
					"a situation. Situations get watched."
			}
			return "(Placeholder) Night-shift posture, day-shift smile. " +
				"The desk has a terminal, a badge gate button, and a " +
				"missing-cat flyer taped where visitors will see it."
		}},
		dialogue.Talkable{
			Start: "greeting",
			Nodes: map[string]dialogue.Node{
				"greeting": {
					Text: "(Placeholder) \"Oh — hey. No pets in the lobby. " +
						"That's a rule. Probably. Nobody's tested it.\"",
					Choices: []dialogue.Choice{
						{
							Text: "Sit directly under the missing-cat " +
								"flyer and look recently bereaved.",
							Require: []dialogue.Requirement{
								dialogue.Flag(flagSawLobbyFlyer),
								dialogue.MissingFlag(flagPlayedLostCat),
							},
							Effects: []dialogue.Effect{
								dialogue.SetFlag(flagPlayedLostCat),
								dialogue.Say("(Placeholder) The receptionist's " +
									"eyes go from you, to the flyer, to you. " +
									"\"...Noodle?\" You have never been Noodle " +
									"in your life. You are absolutely Noodle " +
									"now."),
								dialogue.AwardOnce(flagXPLobbyFlyer, 2),
							},
							End: true,
						},
						{Text: "Mrow.", Next: "mrow"},
						{Text: "Leave.", End: true},
					},
				},
				"mrow": {
					Text: "(Placeholder) \"Yeah, I don't make the rules. " +
						"I just sit where they can see me not making them.\"",
					Choices: []dialogue.Choice{
						{Text: "Leave.", End: true},
					},
				},
			},
		},
		// The charm route: the receptionist can just let a cat in.
		// Failure is a fork (docs/systems/stealth.md): a refused cat
		// becomes a situation, and cute alone stops working — the
		// flyer trick is the angle that changes the circumstances.
		checks.Guarded{
			Dest: "okuda_office",
			Approaches: map[checks.Approach]checks.Attempt{
				checks.Charm: {
					Difficulty: 20,
					Success: "(Placeholder) The receptionist glances at the " +
						"cameras, mutters \"lunch break,\" and badges you " +
						"through the gate like contraband. \"Records office. " +
						"Nobody goes in there. Find your way home, Noodle.\"",
					Failure: "(Placeholder) You deploy the eyes. The " +
						"receptionist reaches for the phone instead of the " +
						"gate button. Not like this — something has to change.",
					OnFail: []string{flagReceptionistSuspicious},
					Mods: []checks.Mod{
						{If: []string{flagReceptionistSuspicious}, Delta: -2},
						{If: []string{flagPlayedLostCat}, Delta: +6},
					},
				},
			},
			Refusals: map[checks.Approach]string{
				checks.Sneak: "(Placeholder) The desk faces the gate, the " +
					"gate faces the desk. This lobby was designed by " +
					"someone who owned a cat.",
				checks.Parkour: "(Placeholder) The gate is chest-high glass " +
					"with nothing above it but receptionist sightline. " +
					"Athletic, sure. Invisible, no.",
			},
		},
	)

	alley = engine.NewEntity("okuda_alley", "Okuda HQ — Service Alley").With(
		engine.Description{Text: "(Placeholder) Dumpsters, steam vents, and a " +
			"fire escape holding its ladder just out of reach — the " +
			"building's one honest entrance."},
		engine.Exits{
			Dirs:    map[string]string{"west": "okuda_street"},
			Blocked: "(Placeholder) Dead end. The street is back west.",
		},
	)

	fireEscape := engine.NewEntity("fire_escape", "the fire escape", "fire escape", "escape", "ladder").With(
		engine.Description{Text: "(Placeholder) Rust and geometry. The " +
			"bottom ladder is raised, but a dumpster, a vent hood, and a " +
			"window ledge describe a route the architect never signed " +
			"off on."},
		// The agility door in. A fumbled climb is loud, and loud
		// travels up a corpo building's spine — the corridor guard
		// reads the same flag.
		checks.Guarded{
			Dest: "okuda_corridor",
			Approaches: map[checks.Approach]checks.Attempt{
				checks.Parkour: {
					Difficulty: 20,
					Success: "(Placeholder) Dumpster, vent hood, ledge, " +
						"ladder, landing. A maintenance window on the " +
						"second floor was painted shut in the last " +
						"century; the paint lost.",
					Failure: "(Placeholder) The vent hood bows under you " +
						"with a boom like a cheap gong. Somewhere above, a " +
						"door opens. Not like this — something has to change.",
					OnFail: []string{flagOkudaGuardAlerted},
					Mods: []checks.Mod{
						{If: []string{flagOkudaGuardAlerted}, Delta: -2},
					},
				},
			},
			Refusals: map[checks.Approach]string{
				checks.Sneak: "(Placeholder) Sneak up a vertical wall? " +
					"There's nothing to sneak past down here but pigeons.",
				checks.Charm: "(Placeholder) You give the fire escape the " +
					"eyes. It remains a fire escape.",
			},
		},
	)

	corridor = engine.NewEntity("okuda_corridor", "Okuda HQ — Second Floor").With(
		engine.Description{Fn: func(w *engine.World) string {
			base := "(Placeholder) A service corridor lit like an " +
				"interrogation. A guard walks it end to end, past the " +
				"records office door. The maintenance window behind you " +
				"is the way back down."
			if w.Flags[flagCorridorLightsOut] {
				return "(Placeholder) The corridor is dark now, emergency " +
					"strips glowing at ankle height — cat height. The " +
					"guard's flashlight makes him easy to place."
			}
			return base
		}},
		engine.Exits{
			Dirs:    map[string]string{"down": "okuda_alley"},
			Blocked: "(Placeholder) The only ways from here: the window down, or past the guard.",
		},
	)

	breaker := engine.NewEntity("breaker", "a breaker panel", "breaker", "panel", "lights").With(
		engine.Description{Text: "(Placeholder) A breaker panel with its " +
			"cover hanging open, because maintenance budgets are the " +
			"first thing corpos cut. One fat switch says CORRIDOR."},
		engine.On{Verb: "knock", Do: func(w *engine.World) string {
			if w.Flags[flagCorridorLightsOut] {
				return "(Placeholder) The lights are already out. The " +
					"guard will get them back on eventually — move."
			}
			w.Flags[flagCorridorLightsOut] = true
			return "(Placeholder) One paw, one switch, one corridor gone " +
				"dark. Down the hall the guard sighs the sigh of a man " +
				"who has met this panel before."
		}},
	)

	guard := engine.NewEntity("okuda_guard", "a corpo guard", "guard", "human").With(
		engine.Notable{},
		engine.Description{Fn: func(w *engine.World) string {
			if w.Flags[flagOkudaGuardAlerted] {
				return "(Placeholder) The guard has stopped walking his " +
					"loop. He stands where he can see the whole corridor, " +
					"hand near his radio. He knows something's inside."
			}
			return "(Placeholder) A corpo guard on a long loop: door, " +
				"window, vending machine, door. His boots announce him " +
				"a corridor away. Rhythms can be learned."
		}},
		checks.Guarded{
			Dest: "okuda_office",
			Approaches: map[checks.Approach]checks.Attempt{
				checks.Sneak: {
					Difficulty: 22,
					Success: "(Placeholder) You learn his loop, then live " +
						"in the gaps of it: doorframe, shadow, cart, " +
						"records office door. He walks past close enough " +
						"to touch and touches nothing.",
					Failure: "(Placeholder) A boot stops mid-corridor. " +
						"\"...the hell?\" The flashlight swings low — cat " +
						"height. Not like this — something has to change.",
					OnFail: []string{flagOkudaGuardAlerted},
					// Alert stiffens him; darkness is spent by the roll
					// it covers (he resets the breaker either way).
					Mods: []checks.Mod{
						{If: []string{flagOkudaGuardAlerted}, Delta: -2},
						{If: []string{flagCorridorLightsOut}, Delta: +6,
							Consume: flagCorridorLightsOut},
					},
				},
			},
			Refusals: map[checks.Approach]string{
				checks.Charm: "(Placeholder) He's paid to be immune to " +
					"exactly you. Corpo onboarding covers the eyes.",
				checks.Parkour: "(Placeholder) The ceiling is smooth panel " +
					"and camera domes. There is no acrobat's route past a " +
					"man in a hallway.",
			},
		},
	)

	camera := engine.NewEntity("okuda_camera", "a camera dome", "camera", "dome", "cameras").With(
		engine.Description{Text: "(Placeholder) A smoked-glass dome on the " +
			"ceiling. It watches the corridor at human height and human " +
			"speed. Filed away."},
	)

	office = engine.NewEntity("okuda_office", "Okuda HQ — Records Office").With(
		engine.Description{Text: "(Placeholder) A records office nobody " +
			"visits: shelves of dead paper, a desk, and a maintenance " +
			"console still humming — the building's grid node, forgotten " +
			"but powered. Service stairs lead down toward the lobby."},
		engine.Exits{
			Dirs:    map[string]string{"down": "okuda_lobby"},
			Blocked: "(Placeholder) Paper, dust, and one way out: the stairs down.",
		},
	)

	console := engine.NewEntity("console", "the maintenance console", "console", "computer", "terminal", "node").With(
		engine.Description{Fn: func(w *engine.World) string {
			if w.Flags[flagOkudaPortOpen] {
				return "(Placeholder) The console's port light burns " +
					"steady green now. okuda.grid is listening for the deck."
			}
			return "(Placeholder) An old maintenance console, fans " +
				"whispering. Its network panel shows one port physically " +
				"switched off — someone wanted this node forgotten, not " +
				"dead."
		}},
		engine.On{Verb: "use", Do: func(w *engine.World) string {
			if w.Flags[flagOkudaPortOpen] {
				return "(Placeholder) The port is already open. The rest " +
					"of this lock lives on the deck."
			}
			w.Flags[flagOkudaPortOpen] = true
			return "(Placeholder) You hook a claw under the port switch " +
				"and lever it over. Somewhere in the walls, a dead line " +
				"warms up. okuda.grid just rejoined the net — and only " +
				"you know." + awardOnce(w, flagXPOkudaConsole, 5)
		}},
	)

	street.Add(facade)
	lobby.Add(receptionist, flyer)
	alley.Add(fireEscape)
	corridor.Add(guard, breaker, camera)
	office.Add(console)
	return street, lobby, alley, corridor, office
}

// okudaEvents narrates the blackboard forks (docs/systems/events.md).
func okudaEvents() []engine.When {
	return []engine.When{
		{
			Flags: []string{flagReceptionistSuspicious},
			Once:  flagSeenReceptionistSuspicious,
			Do: func(w *engine.World) string {
				return "(Placeholder) Behind you, the receptionist's " +
					"chair creaks — sitting up straighter, watching. " +
					"Cute has stopped being a plan. You'll need an angle."
			},
		},
		{
			Flags: []string{flagOkudaGuardAlerted},
			Once:  flagSeenOkudaGuardAlerted,
			Do: func(w *engine.World) string {
				return "(Placeholder) Somewhere in the building a radio " +
					"crackles, and boots change rhythm. Okuda knows " +
					"something small and quick got inside its skin."
			},
		},
	}
}

// okudaJournal: derived entries, visible once their flag is true.
func okudaJournal() []engine.Entry {
	return []engine.Entry{
		{Flag: flagReceptionistSuspicious, Text: "(Placeholder) The Okuda " +
			"receptionist has my number. Straight charm is burned — " +
			"there was a missing-cat flyer on that desk, though."},
		{Flag: flagOkudaGuardAlerted, Text: "(Placeholder) The guard on " +
			"Okuda's second floor is spooked. Darkness would help; " +
			"there was a breaker panel in that corridor."},
		{Flag: flagOkudaPortOpen, Text: "Threw the port switch on Okuda's " +
			"forgotten grid node. okuda.grid should answer the deck now."},
	}
}
