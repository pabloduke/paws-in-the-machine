package game

import "github.com/pabloduke/paws-in-the-machine/internal/engine"

// buildBackroom is The Back Room behind the coffee shop: the server
// rack, the writing desk, the candlestick trick, and the ruby.
func buildBackroom() *engine.Entity {
	backroom := engine.NewEntity("backroom", "The Back Room").With(
		engine.Description{Fn: func(w *engine.World) string {
			base := "(Placeholder) Storage, a humming server rack that has " +
				"no business in a coffee shop, and the smell of secrets. " +
				"An old writing desk sits against the wall, a brass " +
				"candlestick bolted to its top — bolted, in a room where " +
				"nothing else is."
			if w.Flags[flagDrawerOpen] {
				return base + "\n\nThe desk drawer hangs open, its felt " +
					"lining glowing faintly red."
			}
			return base
		}},
		engine.Exits{
			Dirs:    map[string]string{"north": "coffeeshop"},
			Blocked: "(Placeholder) One way in, one way out: north.",
		},
	)

	candlestick := engine.NewEntity("candlestick", "a brass candlestick", "candlestick", "candle").With(
		engine.Description{Text: "(Placeholder) Brass, bolted down, and " +
			"polished by many hands. Or paws."},
		engine.On{Verb: "turn", Do: func(w *engine.World) string {
			if w.Flags[flagDrawerOpen] {
				return "(Placeholder) It spins freely now. The drawer is " +
					"already open."
			}
			w.Flags[flagDrawerOpen] = true
			return "(Placeholder) You brace and twist. Somewhere inside " +
				"the desk, a counterweight shifts — the drawer slides " +
				"open with a click."
		}},
	)

	rack := engine.NewEntity("rack", "the server rack", "rack", "server", "servers").With(
		engine.Description{Text: "(Placeholder) Enterprise hardware, humming " +
			"and warm, in the back of a noodle-adjacent coffee shop. I " +
			"wonder who it really belongs to."},
		engine.On{Verb: "use", Do: func(w *engine.World) string {
			if w.Flags[flagMicroslopRouteOpen] {
				return "(Placeholder) The deck is already patched into the rack's forgotten maintenance jack."
			}
			w.Flags[flagMicroslopRouteOpen] = true
			return "(Placeholder) You nose a loose cable down, hook the deck " +
				"into the rack's forgotten maintenance jack, and the Microslop " +
				"intranet ghost-lights on your screen."
		}},
	)

	desk := engine.NewEntity("writing_desk", "the writing desk").With(
		engine.Description{Text: "(Placeholder) An old writing desk, older " +
			"than everything else in the room combined. The candlestick " +
			"bolted to its top gleams from handling."},
	)

	drawer := engine.NewEntity("drawer", "a desk drawer", "drawer").With(
		engine.Aspect{Fn: func(w *engine.World) string {
			if w.Flags[flagDrawerOpen] {
				return "an open desk drawer"
			}
			return "a desk drawer"
		}},
		engine.Description{Fn: func(w *engine.World) string {
			if w.Flags[flagDrawerOpen] {
				return "(Placeholder) Felt-lined and open. The ruby catches " +
					"what little light there is."
			}
			return "(Placeholder) Locked tight, no keyhole. The desk is " +
				"cleverer than it looks."
		}},
		engine.Openable{Flag: flagDrawerOpen},
	)

	ruby := engine.NewEntity("ruby", "a ruby", "ruby", "gem").With(
		engine.Description{Text: "(Placeholder) Deep red, real, and worth " +
			"more than this whole block. Someone hid it well."},
		engine.Portable{},
	)

	backroom.Add(candlestick, drawer, rack, desk)
	drawer.Add(ruby)
	return backroom
}
