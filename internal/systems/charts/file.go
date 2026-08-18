package charts

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Serialized form. Versioned JSON, stdlib only — the same shape and
// discipline as engine.SaveState, so content files and save files age
// the same way.
//
// The package stays file-blind: this converts between a Weave and
// bytes, and callers do the I/O. Charts are content, so in the shipped
// binary these bytes come from go:embed; the editor reads and writes
// them on disk.

// FileVersion is bumped whenever the on-disk shape changes
// incompatibly.
const FileVersion = 1

type file struct {
	Version int          `json:"version"`
	Charts  []chartData  `json:"charts"`
	Gluings []gluingData `json:"gluings,omitempty"`
}

type chartData struct {
	ID string `json:"id"`
	// Cells are keyed by coordinate, written "x,y", "x,y,z" or
	// "x,y,z,w" — trailing zeros dropped. One line per room keeps a
	// city-sized chart readable and makes a moved room a one-line diff,
	// which is the point: content review is done by a human.
	Cells map[string]string `json:"cells"`
}

// parseCoord reads "x,y", "x,y,z" or "x,y,z,w". Missing axes are zero.
func parseCoord(s string) (Coord, error) {
	var c Coord
	into := []*int{&c.X, &c.Y, &c.Z, &c.W}
	parts := strings.Split(s, ",")
	if len(parts) < 2 || len(parts) > 4 {
		return c, fmt.Errorf("charts: bad coordinate %q", s)
	}
	for i, p := range parts {
		n, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			return c, fmt.Errorf("charts: bad coordinate %q", s)
		}
		*into[i] = n
	}
	return c, nil
}

// formatCoord writes the shortest form that preserves every axis.
func formatCoord(c Coord) string {
	axes := []int{c.X, c.Y, c.Z, c.W}
	n := len(axes)
	for n > 2 && axes[n-1] == 0 {
		n--
	}
	out := make([]string, n)
	for i := 0; i < n; i++ {
		out[i] = strconv.Itoa(axes[i])
	}
	return strings.Join(out, ",")
}

type gluingData struct {
	Chart   string `json:"chart"`
	From    coord  `json:"from"`
	Dir     string `json:"dir"`
	To      coord  `json:"to"`
	ToChart string `json:"to_chart,omitempty"`
}

type coord struct {
	X int `json:"x"`
	Y int `json:"y"`
	Z int `json:"z,omitempty"`
	W int `json:"w,omitempty"`
}

func (c coord) real() Coord { return Coord{c.X, c.Y, c.Z, c.W} }
func wire(c Coord) coord    { return coord{c.X, c.Y, c.Z, c.W} }

// Marshal renders a weave as indented JSON. Output is deterministic —
// charts by ID, cells by coordinate, gluings in authored order — so
// saving an unchanged weave produces a byte-identical file and the
// editor never manufactures diff noise.
func Marshal(w *Weave) ([]byte, error) {
	f := file{Version: FileVersion}
	for _, c := range w.Charts() {
		cd := chartData{ID: c.ID, Cells: make(map[string]string, len(c.Cells))}
		for at, entity := range c.Cells {
			cd.Cells[formatCoord(at)] = entity
		}
		f.Charts = append(f.Charts, cd)
	}
	from, gs := w.Gluings()
	for i, g := range gs {
		f.Gluings = append(f.Gluings, gluingData{
			Chart: from[i], From: wire(g.From), Dir: g.Dir,
			To: wire(g.To), ToChart: g.Chart,
		})
	}
	out, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

// Unmarshal rebuilds a weave. Malformed geometry is reported rather
// than panicking: a bad gluing is a content bug, and the caller
// decides whether that is fatal (tests, validation) or merely noisy.
func Unmarshal(data []byte) (*Weave, []string, error) {
	var f file
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, nil, err
	}
	if f.Version != FileVersion {
		return nil, nil, fmt.Errorf("charts: file version %d, want %d",
			f.Version, FileVersion)
	}
	cs := make([]*Chart, 0, len(f.Charts))
	var bugs []string
	for _, cd := range f.Charts {
		cells := make(map[Coord]string, len(cd.Cells))
		for at, entity := range cd.Cells {
			c, err := parseCoord(at)
			if err != nil {
				bugs = append(bugs, "(bug) chart "+cd.ID+": "+err.Error())
				continue
			}
			cells[c] = entity
		}
		cs = append(cs, New(cd.ID, cells))
	}
	w := NewWeave(cs...)
	for _, g := range f.Gluings {
		bugs = append(bugs, w.Glue(g.Chart, Gluing{
			From: g.From.real(), Dir: g.Dir,
			To: g.To.real(), Chart: g.ToChart,
		})...)
	}
	return w, bugs, nil
}
