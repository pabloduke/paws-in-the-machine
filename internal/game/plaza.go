package game

import "github.com/pabloduke/paws-in-the-machine/internal/engine"

// buildPlaza is The Plaza hub's rooms: Plaza Square under the
// ad-drones, and the shuttered arcade with its humming wall.
func buildPlaza() (square, arcade *engine.Entity) {
	square = engine.NewEntity("plaza_square", "Plaza Square").With(
		engine.Description{Text: "(Placeholder) The Plaza. Ad-drones wheeling " +
			"under the dome glow, crowds that part around you without " +
			"noticing you. Nothing here needs a cat. Yet."},
		engine.Exits{
			Dirs:    map[string]string{"east": "arcade"},
			Blocked: "(Placeholder) Crowds and chrome in every other direction.",
		},
	)

	arcade = engine.NewEntity("arcade", "The Shuttered Arcade").With(
		engine.Description{Text: "(Placeholder) A dead arcade, cabinets under " +
			"dust sheets. Something hums in the back wall that shouldn't."},
		engine.Exits{
			Dirs:    map[string]string{"west": "plaza_square"},
			Blocked: "(Placeholder) The square is back west.",
		},
	)

	drones := engine.NewEntity("drones", "the ad-drones", "drone", "drones", "ad-drones").With(
		engine.Description{Text: "(Placeholder) They wheel and flash overhead, " +
			"selling things to people who aren't looking up. They never " +
			"look down, either. Worth remembering."},
	)

	cabinets := engine.NewEntity("cabinets", "the cabinets", "cabinet", "dust sheets", "sheets").With(
		engine.Description{Text: "(Placeholder) Arcade cabinets under dust " +
			"sheets, dead a decade. Pretty boring — except the dust on " +
			"the nearest sheet is disturbed."},
	)

	hum := engine.NewEntity("hum", "the humming wall", "wall", "hum").With(
		engine.On{Verb: "examine", Do: func(w *engine.World) string {
			return "(Placeholder) You press an ear to the wall. Something " +
				"back there is alive in the electrical sense. Filed away." +
				awardOnce(w, flagXPHum, 3)
		}},
	)

	square.Add(drones)
	arcade.Add(hum, cabinets)
	return square, arcade
}
