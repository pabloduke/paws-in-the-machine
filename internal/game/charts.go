package game

import (
	_ "embed"

	"github.com/pabloduke/paws-in-the-machine/internal/systems/charts"
)

// Geometry lives in a content file, not in Go (docs/systems/charts.md).
// It is embedded so the shipped binary stays self-contained, while the
// editor reads and writes the same file on disk — one format, one
// source of truth, no generated Go to clobber.
//
//go:embed content/charts.json
var chartsJSON []byte

// gameCharts loads the charted nodes. Malformed geometry is reported
// rather than fatal: a content bug should surface as a visible
// complaint, not a panic on startup.
//
// Okuda is deliberately absent from the file. Its six rooms are not
// realizable on a lattice as currently wired — the guard's passage
// joins the corridor and the office, which the other exits force
// diagonal. Charting it means adding a room, moving one, or re-routing
// the guard, all content decisions. Until one is taken, Okuda keeps its
// hand-declared exits; charts and hand-wiring coexist by design. See
// docs/systems/charts.md "Okuda is not lattice-realizable".
func gameCharts() (*charts.Weave, []string) {
	w, bugs, err := charts.Unmarshal(chartsJSON)
	if err != nil {
		return charts.NewWeave(), []string{"(bug) charts: " + err.Error()}
	}
	return w, bugs
}
