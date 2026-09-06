package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/pabloduke/paws-in-the-machine/internal/engine"
	"github.com/pabloduke/paws-in-the-machine/internal/game"
	"github.com/pabloduke/paws-in-the-machine/internal/ui"
)

type worldChoice struct {
	Name, Dir string
	BuiltIn   bool
}

func repositoryContentDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if info, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil && !info.IsDir() {
			return filepath.Join(dir, "internal", "game", "content"), nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("repository not found; use -content-dir or run from the repository")
		}
		dir = parent
	}
}

func discoverWorlds(contentDir, worldsDir string) ([]worldChoice, error) {
	if contentDir != "" {
		dir, err := filepath.Abs(contentDir)
		return []worldChoice{{Name: filepath.Base(dir), Dir: dir}}, err
	}
	choices := []worldChoice{{Name: "Built-in game", BuiltIn: true}}
	mainDir, err := repositoryContentDir()
	if err == nil {
		choices = append(choices, worldChoice{Name: "main (editor)", Dir: mainDir})
	}
	if worldsDir == "" {
		if err != nil {
			return choices, err
		}
		worldsDir = filepath.Join(filepath.Dir(filepath.Dir(filepath.Dir(mainDir))), "worlds")
	}
	entries, readErr := os.ReadDir(worldsDir)
	if os.IsNotExist(readErr) {
		return choices, nil
	}
	if readErr != nil {
		return choices, readErr
	}
	var named []worldChoice
	for _, entry := range entries {
		if !entry.IsDir() || !validWorldName(entry.Name()) {
			continue
		}
		dir, err := filepath.Abs(filepath.Join(worldsDir, entry.Name()))
		if err != nil {
			return choices, err
		}
		named = append(named, worldChoice{Name: entry.Name(), Dir: dir})
	}
	sort.Slice(named, func(i, j int) bool {
		a, b := strings.ToLower(named[i].Name), strings.ToLower(named[j].Name)
		if a == b {
			return named[i].Name < named[j].Name
		}
		return a < b
	})
	return append(choices, named...), nil
}
func validWorldName(name string) bool {
	if name == "" || len(name) > 64 || name == "main" {
		return false
	}
	for _, r := range name {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_') {
			return false
		}
	}
	return true
}

type startupModel struct {
	choices     []worldChoice
	selected    int
	notice      string
	reviewing   bool
	pending     game.AuthoredResult
	diagnostics viewport.Model
	session     tea.Model
	size        tea.WindowSizeMsg
}

func newStartup(choices []worldChoice, err error) startupModel {
	m := startupModel{choices: choices, diagnostics: viewport.New(76, 16), size: tea.WindowSizeMsg{Width: 80, Height: 24}}
	if err != nil {
		m.notice = err.Error()
	}
	return m
}
func (m startupModel) Init() tea.Cmd { return nil }
func (m startupModel) begin(w *engine.World, builtIn bool) (tea.Model, tea.Cmd) {
	intro := ""
	if builtIn {
		intro = game.Intro
	}
	session := ui.New(engine.New(w), intro)
	if builtIn {
		session.BootIntoDeck()
	} else {
		session.Playtest = true
		session.SavePath = ""
	}
	sized, resizeCmd := session.Update(m.size)
	m.session = sized
	return m, tea.Batch(sized.Init(), resizeCmd)
}
func (m startupModel) loadSelected() (tea.Model, tea.Cmd) {
	if len(m.choices) == 0 {
		return m, nil
	}
	choice := m.choices[m.selected]
	if choice.BuiltIn {
		return m.begin(game.NewWorld(), true)
	}
	m.pending = game.LoadAuthored(choice.Dir)
	m.reviewing = true
	text := m.pending.DiagnosticText()
	if text == "" {
		text = "Ready to play."
	}
	m.diagnostics.SetContent(text)
	m.diagnostics.GotoTop()
	return m, nil
}
func (m startupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		m.size = size
		m.diagnostics.Width = max(10, size.Width-4)
		m.diagnostics.Height = max(1, size.Height-9)
	}
	if m.session != nil {
		next, cmd := m.session.Update(msg)
		m.session = next
		return m, cmd
	}
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc":
			m.reviewing = false
			m.pending = game.AuthoredResult{}
			return m, nil
		case "r":
			if m.reviewing {
				return m.loadSelected()
			}
		case "enter":
			if m.reviewing {
				if m.pending.Ready() {
					return m.begin(m.pending.World, false)
				}
				return m.loadSelected()
			}
			return m.loadSelected()
		case "up", "k":
			if !m.reviewing && len(m.choices) > 0 {
				m.selected = (m.selected + len(m.choices) - 1) % len(m.choices)
				return m, nil
			}
		case "down", "j":
			if !m.reviewing && len(m.choices) > 0 {
				m.selected = (m.selected + 1) % len(m.choices)
				return m, nil
			}
		}
	}
	if m.reviewing {
		var cmd tea.Cmd
		m.diagnostics, cmd = m.diagnostics.Update(msg)
		return m, cmd
	}
	return m, nil
}
func (m startupModel) View() string {
	if m.session != nil {
		return m.session.View()
	}
	var b strings.Builder
	if m.reviewing {
		fmt.Fprintf(&b, "\nPlaytest: %s\n\n%s\n\n", m.choices[m.selected].Name, m.diagnostics.View())
		if m.pending.Ready() {
			b.WriteString("Enter: play fresh session")
		} else {
			b.WriteString("Not ready. Fix the errors in the editor, then press Enter to retry.")
		}
		b.WriteString("\nr: reload files · Esc: worlds · ↑/↓: scroll · q: quit\n")
	} else {
		b.WriteString("\nChoose a world\n\n")
		if m.notice != "" {
			b.WriteString(m.notice + "\n\n")
		}
		// Keep the selected row visible even with more worlds than terminal rows.
		count := max(1, m.size.Height-9)
		start := max(0, m.selected-count+1)
		end := min(len(m.choices), start+count)
		for i := start; i < end; i++ {
			marker := "  "
			if i == m.selected {
				marker = "> "
			}
			b.WriteString(marker + m.choices[i].Name + "\n")
		}
		b.WriteString("\n↑/↓: choose · Enter: load · q: quit\n")
	}
	return b.String()
}
