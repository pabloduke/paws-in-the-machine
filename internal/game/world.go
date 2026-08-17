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

	lair, pda := buildLair()
	coffeeshop := buildCoffeeshop()
	plazaSquare, arcade := buildPlaza()
	okudaStreet, okudaLobby, okudaAlley, okudaCorridor, okudaOffice, okudaAnnex := buildOkuda()

	neighborhood := engine.NewEntity("neighborhood", "The Neighborhood").With(
		hubs.Hub{Entry: "lair"},
	)
	plaza := engine.NewEntity("plaza", "The Plaza").With(
		hubs.Hub{Entry: "plaza_square"},
	)
	okuda := engine.NewEntity("okuda", "Okuda HQ").With(
		hubs.Hub{Entry: "okuda_street"},
	)

	w.Root.Add(neighborhood, plaza, okuda)
	neighborhood.Add(lair, coffeeshop)
	plaza.Add(plazaSquare, arcade)
	okuda.Add(okudaStreet, okudaLobby, okudaAlley, okudaCorridor, okudaOffice, okudaAnnex)
	lair.Add(w.Player)
	w.Player.Add(pda)

	// Starting numbers for a new game. The seed is randomized by
	// engine.NewWorld (rand.Int64), so every playthrough rolls its own
	// table — the XCOM rule stays honest and save-scumming stays
	// useless (the seed persists through save/load). Demo and content
	// tests pin w.Seed = 3 to assert specific tuned outcomes; see
	// badge and Okuda demo tests.
	w.Stats = engine.Stats{Stealth: 10, Agility: 12, Charm: 8}
	w.XP = 0

	// Event rules and journal content, per area (docs/systems/events.md).
	w.Rules = append(w.Rules, lairEvents()...)
	w.Rules = append(w.Rules, okudaEvents()...)
	w.Rules = append(w.Rules, netEvents()...)
	w.Journal = append(w.Journal, lairJournal()...)
	w.Journal = append(w.Journal, netJournal()...)
	w.Journal = append(w.Journal, coffeeshopJournal()...)
	w.Journal = append(w.Journal, okudaJournal()...)

	// Geometry: charted nodes derive their exits from adjacency
	// (docs/systems/charts.md). Authored Blocked prose and Gated locks
	// survive; uncharted nodes keep their hand-declared exits.
	w.Pending = append(w.Pending, gameCharts().Apply(w)...)

	// Idioms a player will reach for that the generic parser can't guess.
	w.Rewrites["log in"] = "use deck"

	return w
}
