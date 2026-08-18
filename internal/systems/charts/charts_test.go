package charts_test

import (
	"testing"

	"github.com/pabloduke/paws-in-the-machine/internal/engine"
	"github.com/pabloduke/paws-in-the-machine/internal/systems/charts"
)

// A small node chart, laid out the way an author would read it:
//
//	y
//	1   .      office   annex
//	0   .      lobby      .
//	    0        1        2   x
func nodeChart() *charts.Chart {
	return charts.New("okuda", map[charts.Coord]string{
		{X: 1, Y: 0}: "lobby",
		{X: 1, Y: 1}: "office",
		{X: 2, Y: 1}: "annex",
	})
}

func TestExitsDeriveFromAdjacency(t *testing.T) {
	w := charts.NewWeave(nodeChart())

	got := w.Exits("okuda", charts.Coord{X: 1, Y: 0})
	if len(got) != 1 || got["north"] != "office" {
		t.Fatalf("lobby should have exactly one exit, north to office: %v", got)
	}

	got = w.Exits("okuda", charts.Coord{X: 1, Y: 1})
	if got["south"] != "lobby" || got["east"] != "annex" || len(got) != 2 {
		t.Fatalf("office should reach lobby south and annex east: %v", got)
	}
}

// Reciprocity is the whole point of deriving exits: it holds by
// construction, so there is no way to author a Zork maze.
func TestReciprocityHoldsForEveryCell(t *testing.T) {
	c := nodeChart()
	w := charts.NewWeave(c)
	opposite := map[string]string{
		"north": "south", "south": "north",
		"east": "west", "west": "east",
		"up": "down", "down": "up",
	}

	for at, from := range c.Cells {
		for dir, destID := range w.Exits("okuda", at) {
			destAt, ok := c.Find(destID)
			if !ok {
				t.Fatalf("exit %s from %s names a cell not in the chart", dir, from)
			}
			back := w.Exits("okuda", destAt)[opposite[dir]]
			if back != from {
				t.Fatalf("%s leaves %s going %s but %s does not lead back (got %q)",
					from, dir, dir, opposite[dir], back)
			}
		}
	}
}

// An empty neighbour is no exit at all — the chart is sparse, and
// nothing fabricates a passage into a coordinate nobody authored.
func TestEmptyNeighbourIsNoExit(t *testing.T) {
	w := charts.NewWeave(nodeChart())
	if got := w.Exits("okuda", charts.Coord{X: 2, Y: 1})["north"]; got != "" {
		t.Fatalf("annex has nothing north of it, got %q", got)
	}
}

// A fold: the archive's missing aisle sits one step ana, reached by an
// ordinary compass direction. The player's vocabulary never changes.
func TestFoldIsReachedByACompassDirection(t *testing.T) {
	archive := charts.Coord{X: 2, Y: 1}
	aisle := charts.Ana(archive)

	c := charts.New("okuda", map[charts.Coord]string{
		archive: "annex",
		aisle:   "aisle410",
	})
	w := charts.NewWeave(c)

	// Without the gluing, w+1 is invisible: no exit derives onto it.
	if got := w.Exits("okuda", archive); len(got) != 0 {
		t.Fatalf("the fourth axis must not derive exits on its own: %v", got)
	}

	if bugs := w.Glue("okuda", charts.Gluing{From: archive, Dir: "east", To: aisle}); bugs != nil {
		t.Fatalf("gluing the fold: %v", bugs)
	}
	if got := w.Exits("okuda", archive)["east"]; got != "aisle410" {
		t.Fatalf("east from the annex should reach the aisle, got %q", got)
	}
	// And the way back, installed for free.
	if got := w.Exits("okuda", aisle)["west"]; got != "annex" {
		t.Fatalf("the fold must lead back west, got %q", got)
	}
}

// Level changes use the same mechanism as folds, across charts.
func TestGluingCrossesCharts(t *testing.T) {
	street := charts.New("plaza", map[charts.Coord]string{{}: "plaza_square"})
	inside := charts.New("shop", map[charts.Coord]string{{}: "shop_floor"})
	w := charts.NewWeave(street, inside)

	if bugs := w.Glue("plaza", charts.Gluing{
		Dir: "north", To: charts.Coord{}, Chart: "shop",
	}); bugs != nil {
		t.Fatalf("gluing across charts: %v", bugs)
	}
	if got := w.Exits("plaza", charts.Coord{})["north"]; got != "shop_floor" {
		t.Fatalf("north from the square should enter the shop, got %q", got)
	}
	if got := w.Exits("shop", charts.Coord{})["south"]; got != "plaza_square" {
		t.Fatalf("south from the shop should return to the square, got %q", got)
	}
}

// A gluing takes precedence over the plain neighbour, so a fold can sit
// where ordinary adjacency would otherwise put a room.
func TestGluingOverridesDerivedAdjacency(t *testing.T) {
	here := charts.Coord{}
	c := charts.New("hall", map[charts.Coord]string{
		here:             "hall",
		{Y: 1}:           "mundane",
		charts.Ana(here): "elsewhere",
	})
	w := charts.NewWeave(c)

	if got := w.Exits("hall", here)["north"]; got != "mundane" {
		t.Fatalf("before gluing, north is the plain neighbour, got %q", got)
	}
	w.Glue("hall", charts.Gluing{From: here, Dir: "north", To: charts.Ana(here)})
	if got := w.Exits("hall", here)["north"]; got != "elsewhere" {
		t.Fatalf("the gluing should win over adjacency, got %q", got)
	}
}

func TestGluingRejectsNonPlayerDirections(t *testing.T) {
	w := charts.NewWeave(charts.New("hall", map[charts.Coord]string{{}: "hall"}))
	bugs := w.Glue("hall", charts.Gluing{Dir: "ana", To: charts.Coord{W: 1}})
	if len(bugs) == 0 {
		t.Fatal("ana is dev vocabulary and must never be a player exit")
	}
}

// Apply writes geometry onto entities while leaving content alone.
func TestApplyDerivesExitsAndKeepsGatesAndProse(t *testing.T) {
	w := engine.NewWorld()
	office := engine.NewEntity("office", "Records Office").With(
		engine.Exits{
			Blocked: "Paper, dust, and the archive door east.",
			Gated: map[string]engine.Gate{
				"east": {Flag: "unlocked", Shut: "The archive door doesn't budge."},
			},
		},
	)
	annex := engine.NewEntity("annex", "Deep Archive")
	w.Root.Add(office, annex)

	weave := charts.NewWeave(charts.New("okuda", map[charts.Coord]string{
		{X: 1, Y: 1}: "office",
		{X: 2, Y: 1}: "annex",
	}))
	if bugs := weave.Apply(w); bugs != nil {
		t.Fatalf("applying charts: %v", bugs)
	}

	got, ok := engine.Part[engine.Exits](office)
	if !ok {
		t.Fatal("office should carry exits after Apply")
	}
	if got.Dirs["east"] != "annex" {
		t.Fatalf("east should derive to the annex: %v", got.Dirs)
	}
	if got.Blocked == "" {
		t.Fatal("authored Blocked prose must survive Apply")
	}
	if g, gated := got.Gated["east"]; !gated || g.Flag != "unlocked" {
		t.Fatalf("authored gate must survive Apply: %v", got.Gated)
	}
	// The gate still governs the derived exit end to end.
	eng := engine.New(w)
	office.Add(w.Player)
	if out := eng.Execute("east"); out != "The archive door doesn't budge." {
		t.Fatalf("the gate should hold the derived exit shut: %q", out)
	}
	w.Flags["unlocked"] = true
	eng.Execute("east")
	if w.Room() != annex {
		t.Fatalf("unlocking should let the derived exit through, in %q", w.Room().ID)
	}
}

func TestApplyReportsUnknownEntities(t *testing.T) {
	w := engine.NewWorld()
	weave := charts.NewWeave(charts.New("okuda", map[charts.Coord]string{
		{}: "nobody_here",
	}))
	if bugs := weave.Apply(w); len(bugs) == 0 {
		t.Fatal("a cell naming an entity that isn't in the world is a content bug")
	}
}

// Round-tripping is the editor's core contract: load, save, and the
// bytes must not move. Anything else manufactures diff noise in
// content review.
func TestFileRoundTripsByteIdentical(t *testing.T) {
	here := charts.Coord{X: 1, Y: 1}
	w := charts.NewWeave(
		charts.New("okuda", map[charts.Coord]string{
			{X: 1, Y: 0}: "lobby",
			here:         "office",
			{X: 2, Y: 1}: "annex",
		}),
		charts.New("aisle", map[charts.Coord]string{{}: "aisle410"}),
	)
	if bugs := w.Glue("okuda", charts.Gluing{
		From: here, Dir: "north", To: charts.Coord{}, Chart: "aisle",
	}); bugs != nil {
		t.Fatalf("gluing: %v", bugs)
	}

	first, err := charts.Marshal(w)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	back, bugs, err := charts.Unmarshal(first)
	if err != nil || bugs != nil {
		t.Fatalf("unmarshal: err=%v bugs=%v", err, bugs)
	}
	second, err := charts.Marshal(back)
	if err != nil {
		t.Fatalf("re-marshal: %v", err)
	}
	if string(first) != string(second) {
		t.Fatalf("round trip moved bytes:\n--- first ---\n%s\n--- second ---\n%s",
			first, second)
	}

	// And the geometry survives, gluing included.
	if got := back.Exits("okuda", here)["north"]; got != "aisle410" {
		t.Fatalf("glued exit lost in the round trip, got %q", got)
	}
	if got := back.Exits("aisle", charts.Coord{})["south"]; got != "office" {
		t.Fatalf("reverse gluing lost in the round trip, got %q", got)
	}
	if got := back.Exits("okuda", charts.Coord{X: 1, Y: 0})["north"]; got != "office" {
		t.Fatalf("derived adjacency lost in the round trip, got %q", got)
	}
}

func TestUnmarshalRejectsAnotherVersion(t *testing.T) {
	if _, _, err := charts.Unmarshal([]byte(`{"version":99,"charts":[]}`)); err == nil {
		t.Fatal("a file from another version must not load silently")
	}
}
