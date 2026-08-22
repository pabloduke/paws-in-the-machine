package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pabloduke/paws-in-the-machine/internal/systems/charts"
)

type mode int

const (
	browsing mode = iota
	naming        // typing an entity ID into the cell under the cursor
)

type model struct {
	path     string
	weave    *charts.Weave
	entities entityCatalog
	ids      []string // chart IDs, sorted — tab cycles them
	ci       int      // index into ids
	cur      charts.Coord
	mode     mode
	input    textinput.Model
	status   string
	dirty    bool
	w, h     int
}

func newModel(path string, entities entityCatalog) (*model, error) {
	if entities == nil {
		return nil, fmt.Errorf("editor requires an entity catalog")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	weave, bugs, err := charts.Unmarshal(data)
	if err != nil {
		return nil, err
	}

	ids := []string{}
	for _, c := range weave.Charts() {
		ids = append(ids, c.ID)
	}
	sort.Strings(ids)
	if len(ids) == 0 {
		return nil, fmt.Errorf("%s declares no charts", path)
	}

	in := textinput.New()
	in.Prompt = "room id: "
	in.CharLimit = 64

	status := fmt.Sprintf("loaded %s", path)
	problems := append([]string{}, bugs...)
	problems = append(problems, validateWeave(weave, entities)...)
	if len(problems) > 0 {
		status = "validation failed: " + strings.Join(problems, "; ")
	}
	return &model{
		path: path, weave: weave, entities: entities, ids: ids,
		input: in, status: status,
	}, nil
}

func (m *model) Init() tea.Cmd { return nil }

func (m *model) chart() *charts.Chart {
	c, _ := m.weave.Chart(m.ids[m.ci])
	return c
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		return m, nil
	case tea.KeyMsg:
		if m.mode == naming {
			return m.updateNaming(msg)
		}
		return m.updateBrowsing(msg)
	}
	return m, nil
}

func (m *model) updateNaming(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.mode = browsing
		m.status = "cancelled"
		return m, nil
	case tea.KeyEnter:
		id := strings.TrimSpace(m.input.Value())
		m.mode = browsing
		if id == "" {
			m.status = "cancelled"
			return m, nil
		}
		if problem := validatePlacement(m.weave, m.entities, m.ids[m.ci], m.cur, id); problem != "" {
			m.status = "placement rejected: " + problem
			return m, nil
		}
		m.chart().Cells[m.cur] = id
		m.dirty = true
		m.status = fmt.Sprintf("placed %s at %s", id, coordText(m.cur))
		return m, nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *model) updateBrowsing(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		if m.dirty {
			m.status = "unsaved changes — press s to save, or Q to discard"
			return m, nil
		}
		return m, tea.Quit
	case "Q":
		return m, tea.Quit

	// Movement uses the game's own compass, so authoring and playing
	// speak the same language: up is north (y+1), down is south.
	case "up", "k":
		m.cur.Y++
	case "down", "j":
		m.cur.Y--
	case "left", "h":
		m.cur.X--
	case "right", "l":
		m.cur.X++

	// Vertical slices: a building's floors.
	case "<":
		m.cur.Z--
	case ">":
		m.cur.Z++

	// The fourth axis. Dev-facing vocabulary only (hubs.md): the player
	// never sees ana/kata, but the author has to stand there to build a
	// fold.
	case "[":
		m.cur.W--
	case "]":
		m.cur.W++

	case "tab":
		m.ci = (m.ci + 1) % len(m.ids)
		m.status = "chart: " + m.ids[m.ci]
	case "n", "enter":
		m.mode = naming
		m.input.SetValue(m.chart().Cells[m.cur])
		m.input.CursorEnd()
		m.input.Focus()
		return m, textinput.Blink
	case "d":
		if _, ok := m.chart().At(m.cur); !ok {
			m.status = "nothing here"
			return m, nil
		}
		delete(m.chart().Cells, m.cur)
		m.dirty = true
		m.status = "deleted " + coordText(m.cur)
	case "s":
		m.save()
	}
	return m, nil
}

func (m *model) save() {
	if problems := validateWeave(m.weave, m.entities); len(problems) > 0 {
		m.status = "save blocked: " + strings.Join(problems, "; ")
		return
	}
	out, err := charts.Marshal(m.weave)
	if err != nil {
		m.status = "save failed: " + err.Error()
		return
	}
	if err := os.WriteFile(m.path, out, 0o644); err != nil {
		m.status = "save failed: " + err.Error()
		return
	}
	m.dirty = false
	m.status = "saved " + m.path
}

func coordText(c charts.Coord) string {
	s := fmt.Sprintf("(%d, %d", c.X, c.Y)
	if c.Z != 0 || c.W != 0 {
		s += fmt.Sprintf(", %d", c.Z)
	}
	if c.W != 0 {
		s += fmt.Sprintf(", %d", c.W)
	}
	return s + ")"
}

var (
	dim     = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	bright  = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	accent  = lipgloss.NewStyle().Foreground(lipgloss.Color("81"))
	cursorS = lipgloss.NewStyle().Foreground(lipgloss.Color("232")).
		Background(lipgloss.Color("81"))
	warn = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
)

const cellW = 14

// bounds returns the rectangle to draw: everything occupied on this
// slice, plus the cursor, padded by one so there is always somewhere to
// grow into.
func (m *model) bounds() (minX, maxX, minY, maxY int) {
	minX, maxX, minY, maxY = m.cur.X, m.cur.X, m.cur.Y, m.cur.Y
	for at := range m.chart().Cells {
		if at.Z != m.cur.Z || at.W != m.cur.W {
			continue
		}
		minX, maxX = min(minX, at.X), max(maxX, at.X)
		minY, maxY = min(minY, at.Y), max(maxY, at.Y)
	}
	return minX - 1, maxX + 1, minY - 1, maxY + 1
}

func (m *model) View() string {
	var b strings.Builder

	title := fmt.Sprintf("CHART EDITOR // %s", strings.ToUpper(m.ids[m.ci]))
	if m.dirty {
		title += " *"
	}
	b.WriteString(accent.Render(title) + "\n")
	slice := fmt.Sprintf("slice z=%d w=%d", m.cur.Z, m.cur.W)
	if m.cur.W != 0 {
		slice += dim.Render(fmt.Sprintf("  (%d step%s ana)", m.cur.W,
			plural(m.cur.W)))
	}
	b.WriteString(dim.Render(slice) + "\n\n")

	b.WriteString(m.grid())
	b.WriteString("\n")
	b.WriteString(m.inspector())
	b.WriteString("\n")

	if m.mode == naming {
		b.WriteString(m.input.View() + "\n")
	} else {
		b.WriteString(dim.Render(
			"hjkl/arrows move · n name · d delete · tab chart · "+
				"< > floor · [ ] ana/kata · s save · q quit") + "\n")
	}
	if m.status != "" {
		b.WriteString(warn.Render(m.status))
	}
	return b.String()
}

// grid draws one z/w slice, north up — the way a player would sketch it
// in a notebook, which is the whole point of lawful geometry.
func (m *model) grid() string {
	minX, maxX, minY, maxY := m.bounds()
	var b strings.Builder

	for y := maxY; y >= minY; y-- {
		b.WriteString(dim.Render(fmt.Sprintf("%3d ", y)))
		for x := minX; x <= maxX; x++ {
			at := charts.Coord{X: x, Y: y, Z: m.cur.Z, W: m.cur.W}
			id, occupied := m.chart().At(at)
			text := "."
			if occupied {
				text = id
			}
			text = fit(text, cellW-1)
			switch {
			case at == m.cur:
				b.WriteString(cursorS.Render(pad(text, cellW)))
			case occupied:
				b.WriteString(bright.Render(pad(text, cellW)))
			default:
				b.WriteString(dim.Render(pad(text, cellW)))
			}
		}
		b.WriteString("\n")
	}

	b.WriteString("    ")
	for x := minX; x <= maxX; x++ {
		b.WriteString(dim.Render(pad(fmt.Sprintf("%d", x), cellW)))
	}
	return b.String() + "\n"
}

// inspector shows what the game will actually do here: the exits that
// derive from this cell's neighbours, gluings included. Authoring and
// truth stay in the same window.
func (m *model) inspector() string {
	id, occupied := m.chart().At(m.cur)
	head := coordText(m.cur) + "  "
	if !occupied {
		return dim.Render(head + "empty")
	}

	exits := m.weave.Exits(m.ids[m.ci], m.cur)
	if len(exits) == 0 {
		return bright.Render(head+id) + dim.Render("  no exits — isolated")
	}
	dirs := make([]string, 0, len(exits))
	for d := range exits {
		dirs = append(dirs, d)
	}
	sort.Strings(dirs)
	parts := make([]string, 0, len(dirs))
	for _, d := range dirs {
		parts = append(parts, fmt.Sprintf("%s→%s", d, exits[d]))
	}
	return bright.Render(head+id) + dim.Render("  "+strings.Join(parts, "  "))
}

func fit(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return s[:n]
	}
	return s[:n-1] + "…"
}

func pad(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(s))
}

func plural(n int) string {
	if n == 1 || n == -1 {
		return ""
	}
	return "s"
}
