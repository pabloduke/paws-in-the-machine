package game

import "github.com/pabloduke/paws-in-the-machine/internal/systems/charts"

// Geometry for the existing nodes (docs/systems/charts.md): rooms sit
// at coordinates and their exits derive from adjacency, so the compass
// cannot lie and reciprocity is free.
//
// Okuda is deliberately absent. Its six rooms are not realizable on a
// lattice as currently wired — the guard's passage joins the corridor
// and the office, but those two cells are forced diagonal by the
// exits already authored (see gameCharts's note below and
// docs/systems/charts.md "Okuda"). Charting it would either invent a
// room or move one, both of which are content decisions. Until then
// Okuda keeps its hand-declared exits, which is why charts and
// hand-wiring have to coexist.
func gameCharts() *charts.Weave {
	// The Neighborhood: the lair, and the coffee shop one step north.
	//
	//	y
	//	1   coffeeshop
	//	0   lair
	//	    0            x
	neighborhood := charts.New("neighborhood", map[charts.Coord]string{
		{}:     "lair",
		{Y: 1}: "coffeeshop",
	})

	// The Plaza: the square, and the arcade one step east.
	//
	//	y
	//	0   plaza_square   arcade
	//	    0              1      x
	plaza := charts.New("plaza", map[charts.Coord]string{
		{}:     "plaza_square",
		{X: 1}: "arcade",
	})

	return charts.NewWeave(neighborhood, plaza)
}
