package game

import (
	"github.com/pabloduke/paws-in-the-machine/internal/engine"
	"github.com/pabloduke/paws-in-the-machine/internal/systems/hubs"
)

// NewWorld constructs the world by stitching the per-area builders
// together. This file is the only place hubs are declared and areas
// reference each other (docs/BOUNDARIES.md); each area's contents
// live in its own file. All prose is placeholder.
func NewWorld() *engine.World {
	w := engine.NewWorld()

	lair, deck := buildLair()
	coffeeshop := buildCoffeeshop()
	backroom := buildBackroom()
	plazaSquare, arcade := buildPlaza()

	neighborhood := engine.NewEntity("neighborhood", "The Neighborhood").With(
		hubs.Hub{Entry: "lair"},
	)
	plaza := engine.NewEntity("plaza", "The Plaza").With(
		hubs.Hub{Entry: "plaza_square"},
	)

	w.Root.Add(neighborhood, plaza)
	neighborhood.Add(lair, coffeeshop, backroom)
	plaza.Add(plazaSquare, arcade)
	lair.Add(w.Player)
	w.Player.Add(deck)

	// Starting numbers. With the pinned seed below, the hound demos the
	// full loop: sneak fails at Stealth 10 (and would pass at 12 —
	// growth flips it), parkour clears, charm is refused outright.
	w.Stats = engine.Stats{Stealth: 10, Agility: 12, Charm: 8}
	w.XP = 0
	// Pinned while tuning the feel; remove to randomize per new game.
	w.Seed = 3

	// Event rules and journal content, per area (docs/systems/events.md).
	w.Rules = append(w.Rules, lairEvents()...)
	w.Journal = append(w.Journal, lairJournal()...)
	w.Journal = append(w.Journal, netJournal()...)

	// Idioms a player will reach for that the generic parser can't guess.
	w.Rewrites["log in"] = "use deck"

	return w
}
