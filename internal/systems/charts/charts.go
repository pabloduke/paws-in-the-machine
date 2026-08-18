// Package charts is the lattice under a node (docs/systems/charts.md).
//
// Rooms sit at integer coordinates and walkable exits derive from
// adjacency, so reciprocity and geometric consistency hold by
// construction — the compass cannot lie, and a scrambled edge is not
// representable. Folds and level changes are authored gluings on the
// fourth axis, never scrambled edges.
//
// Charts produce engine.Exits and nothing else; gates, obstacles, and
// prose layer on top exactly as they do for hand-declared exits.
package charts

import (
	"sort"

	"github.com/pabloduke/paws-in-the-machine/internal/engine"
)

// Coord is a cell's position within its chart. Four slots in practice
// (hubs.md); w is the fourth axis, dev-facing only.
type Coord struct {
	X, Y, Z, W int
}

// Direction deltas. Only the six compass directions derive exits — the
// player's whole vocabulary (hubs.md ruling). ana/kata (w±1) are
// authoring words: they move coordinates but never become exits, which
// is what keeps folding invisible in the interface.
var deltas = map[string]Coord{
	"north": {Y: 1},
	"south": {Y: -1},
	"east":  {X: 1},
	"west":  {X: -1},
	"up":    {Z: 1},
	"down":  {Z: -1},
}

var opposites = map[string]string{
	"north": "south",
	"south": "north",
	"east":  "west",
	"west":  "east",
	"up":    "down",
	"down":  "up",
}

// Directions returns the six player-facing directions, sorted, for
// callers that need to enumerate them (the editor, validators).
func Directions() []string {
	out := make([]string, 0, len(deltas))
	for d := range deltas {
		out = append(out, d)
	}
	sort.Strings(out)
	return out
}

// Ana and Kata step the fourth axis. Authoring helpers: a fold is
// declared by gluing a compass direction to a cell one step ana.
func Ana(c Coord) Coord  { c.W++; return c }
func Kata(c Coord) Coord { c.W--; return c }

func (c Coord) add(d Coord) Coord {
	return Coord{c.X + d.X, c.Y + d.Y, c.Z + d.Z, c.W + d.W}
}

// Chart is a sparse grid of cells. Cells hold one entity ID each;
// unoccupied coordinates are simply nothing.
//
// Chart holds configuration, never mutable state: nothing about a
// chart changes during play (docs/systems/charts.md).
type Chart struct {
	ID    string
	Cells map[Coord]string
}

// Gluing is a declared identification between cells: leaving From in
// direction Dir arrives at To. Chart is To's chart; empty means this
// same chart. One mechanism serves folds, level changes, wraps, and
// twists.
//
// Gluings are bidirectional by construction (see Weave) — a one-way
// passage would be a scrambled edge, which lawful geometry bans.
type Gluing struct {
	From  Coord
	Dir   string
	To    Coord
	Chart string
}

// New builds a chart from a cell map. The map is copied, so later
// edits to the caller's map cannot mutate the chart.
func New(id string, cells map[Coord]string) *Chart {
	c := &Chart{ID: id, Cells: make(map[Coord]string, len(cells))}
	for at, entity := range cells {
		c.Cells[at] = entity
	}
	return c
}

// At returns the entity ID at a coordinate, and whether one is there.
func (c *Chart) At(at Coord) (string, bool) {
	id, ok := c.Cells[at]
	return id, ok
}

// Find returns the coordinate holding an entity ID.
func (c *Chart) Find(entity string) (Coord, bool) {
	for at, id := range c.Cells {
		if id == entity {
			return at, true
		}
	}
	return Coord{}, false
}

// Weave is a set of charts plus the gluings between them: everything
// needed to resolve exits. A single-chart node still uses a Weave —
// it just has one chart and no gluings.
type Weave struct {
	charts  map[string]*Chart
	gluings map[gkey]target
	// declared keeps gluings in authored order so a weave round-trips
	// through a file unchanged. The resolved map above is derived from
	// it (each declaration installs its reverse).
	declared []declaration
}

type declaration struct {
	chart string
	g     Gluing
}

// gkey identifies one face: a cell in a chart, and the direction out of
// it. Coord is comparable, so this works as a map key directly.
type gkey struct {
	chart string
	at    Coord
	dir   string
}

type target struct {
	chart string
	at    Coord
}

// NewWeave assembles charts into a resolvable set.
func NewWeave(cs ...*Chart) *Weave {
	w := &Weave{
		charts:  make(map[string]*Chart, len(cs)),
		gluings: make(map[gkey]target),
	}
	for _, c := range cs {
		w.charts[c.ID] = c
	}
	return w
}

// Chart returns a member chart by ID.
func (w *Weave) Chart(id string) (*Chart, bool) {
	c, ok := w.charts[id]
	return c, ok
}

// Glue installs a gluing and its reverse. Reciprocity is not optional:
// declaring a passage always declares the way back, so no gluing can
// produce a scrambled edge. An unknown direction, or a gluing naming a
// chart that isn't in the weave, is a content bug and is reported.
func (w *Weave) Glue(fromChart string, g Gluing) []string {
	var bugs []string
	toChart := g.Chart
	if toChart == "" {
		toChart = fromChart
	}
	if _, ok := opposites[g.Dir]; !ok {
		return append(bugs, "(bug) gluing from "+fromChart+
			": not a player-facing direction: "+g.Dir)
	}
	if _, ok := w.charts[fromChart]; !ok {
		bugs = append(bugs, "(bug) gluing names unknown chart "+fromChart)
	}
	if _, ok := w.charts[toChart]; !ok {
		bugs = append(bugs, "(bug) gluing names unknown chart "+toChart)
	}
	if len(bugs) > 0 {
		return bugs
	}
	w.set(fromChart, g.From, g.Dir, target{toChart, g.To})
	w.set(toChart, g.To, opposites[g.Dir], target{fromChart, g.From})
	w.declared = append(w.declared, declaration{fromChart, g})
	return nil
}

// Charts returns every chart in the weave, ordered by ID — for the
// editor, validators, and serialization.
func (w *Weave) Charts() []*Chart {
	out := make([]*Chart, 0, len(w.charts))
	for _, c := range w.charts {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Gluings returns the declared gluings in authored order, each paired
// with the chart it leaves from. Reverses are not included: they are
// installed by Glue, not authored.
func (w *Weave) Gluings() ([]string, []Gluing) {
	from := make([]string, len(w.declared))
	gs := make([]Gluing, len(w.declared))
	for i, d := range w.declared {
		from[i], gs[i] = d.chart, d.g
	}
	return from, gs
}

func (w *Weave) set(chart string, at Coord, dir string, to target) {
	w.gluings[gkey{chart, at, dir}] = to
}

// Exits resolves the walkable exits for one cell: derived adjacency,
// with any gluing on that direction taking precedence. The result maps
// a direction to a destination entity ID, ready for engine.Exits.
func (w *Weave) Exits(chartID string, at Coord) map[string]string {
	c, ok := w.charts[chartID]
	if !ok {
		return nil
	}
	out := make(map[string]string)
	for dir, delta := range deltas {
		if to, glued := w.gluings[gkey{chartID, at, dir}]; glued {
			dest, ok := w.charts[to.chart]
			if !ok {
				continue
			}
			if id, occupied := dest.At(to.at); occupied {
				out[dir] = id
			}
			continue
		}
		if id, occupied := c.At(at.add(delta)); occupied {
			out[dir] = id
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// Apply writes derived exits onto every room entity in the weave.
// Authored Blocked prose and Gated locks survive: geometry decides
// which directions exist, content decides what happens on the way
// through. Returns content bugs (a cell naming an entity that isn't in
// the world) rather than panicking.
func (w *Weave) Apply(world *engine.World) []string {
	var bugs []string
	for chartID, c := range w.charts {
		for at, entityID := range c.Cells {
			e := world.FindID(entityID)
			if e == nil {
				bugs = append(bugs, "(bug) chart "+chartID+
					" places unknown entity "+entityID)
				continue
			}
			dirs := w.Exits(chartID, at)
			existing, had := engine.Part[engine.Exits](e)
			if !had {
				e.Parts = append(e.Parts, engine.Exits{Dirs: dirs})
				continue
			}
			replaced := false
			for i, part := range e.Parts {
				if _, ok := part.(engine.Exits); !ok {
					continue
				}
				e.Parts[i] = engine.Exits{
					Dirs:    dirs,
					Blocked: existing.Blocked,
					Gated:   existing.Gated,
				}
				replaced = true
				break
			}
			if !replaced {
				e.Parts = append(e.Parts, engine.Exits{Dirs: dirs})
			}
		}
	}
	sort.Strings(bugs)
	return bugs
}
