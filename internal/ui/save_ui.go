package ui

import (
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pabloduke/paws-in-the-machine/internal/engine"
)

// Save/load (docs/systems/saveload.md). The engine snapshots and
// applies; this surface owns the one place file I/O happens. One
// slot at Model.SavePath. Intercept-only: it never takes the
// keyboard or draws.

type saveSurface struct{}

func (saveSurface) Active(*Model) bool                   { return false }
func (saveSurface) HandleKey(*Model, tea.KeyMsg) tea.Cmd { return nil }

// defaultSavePath is one slot in the user's config dir; "" (no
// resolvable home) disables save/load with a readable error later.
func defaultSavePath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "pawsinthemachine", "save.json")
}

// Intercept claims the "save" and "load" verbs.
func (saveSurface) Intercept(m *Model, cmd engine.Command) bool {
	switch cmd.Verb {
	case "save":
		m.appendLog(textBody, m.doSave())
		return true
	case "load":
		m.appendLog(textBody, m.doLoad())
		return true
	}
	return false
}

func (m *Model) doSave() string {
	if m.Playtest {
		return "Saving is disabled in authored playtests. Restart to load editor changes."
	}
	if m.SavePath == "" {
		return "No save location available on this system."
	}
	data, err := engine.Snapshot(m.eng.World).Marshal()
	if err != nil {
		return "Couldn't build the save: " + err.Error()
	}
	if err := os.MkdirAll(filepath.Dir(m.SavePath), 0o755); err != nil {
		return "Couldn't create the save folder: " + err.Error()
	}
	if err := os.WriteFile(m.SavePath, data, 0o644); err != nil {
		return "Couldn't write the save: " + err.Error()
	}
	return "Saved. The rain can wait."
}

func (m *Model) doLoad() string {
	if m.Playtest {
		return "Loading saves is disabled in authored playtests. Restart to load editor changes."
	}
	if m.SavePath == "" {
		return "No save location available on this system."
	}
	data, err := os.ReadFile(m.SavePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "No save to load — nothing kept yet."
		}
		return "Couldn't read the save: " + err.Error()
	}
	s, err := engine.UnmarshalSave(data)
	if err != nil {
		return err.Error()
	}
	if err := s.Apply(m.eng.World); err != nil {
		return err.Error()
	}
	return "Loaded. Right where you left it."
}
