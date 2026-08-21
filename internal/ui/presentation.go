package ui

import (
	"sort"
	"strings"
)

// Option configures presentation when a UI model is constructed.
type Option func(*modelOptions)

type modelOptions struct {
	themeID string
}

// WithTheme injects a registered presentation theme by ID. Unknown IDs fall
// back to the shipped default, keeping construction total for CLI callers.
func WithTheme(id string) Option {
	return func(opts *modelOptions) { opts.themeID = id }
}

// textRole records why text exists, not how it looks. ANSI is applied only
// when a frame is rendered, so changing themes can restyle existing history.
type textRole uint8

const (
	textBody textRole = iota
	textEcho
	textDim
)

type textElement struct {
	Role textRole
	Text string
}

// presentationTheme is a complete immutable UI appearance. It owns both
// surfaces because a player selects one theme for the application, while the
// separate style sets preserve the overworld/terminal visual boundary.
type presentationTheme struct {
	ID        string
	Overworld overworldStyles
	Terminal  terminalStyles
}

const defaultThemeID = "wet-neon"

var themeRegistry = map[string]presentationTheme{
	defaultThemeID: {
		ID:        defaultThemeID,
		Overworld: defaultOverworldStyles(),
		Terminal:  defaultTerminalStyles(),
	},
}

// ThemeIDs returns registered theme IDs in stable display order. The initial
// layer ships one baseline theme; future visual themes register alongside it.
func ThemeIDs() []string {
	ids := make([]string, 0, len(themeRegistry))
	for id := range themeRegistry {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func themeByID(id string) (presentationTheme, bool) {
	t, ok := themeRegistry[id]
	return t, ok
}

func (m Model) presentation() presentationTheme {
	if t, ok := themeByID(m.themeID); ok {
		return t
	}
	t, _ := themeByID(defaultThemeID)
	return t
}

// setTheme changes presentation only, then rebuilds cached styled views from
// their raw semantic data. It is the seam a future theme command will call.
func (m *Model) setTheme(id string) bool {
	if _, ok := themeByID(id); !ok {
		return false
	}
	m.themeID = id
	styles := m.presentation()
	m.input.Prompt = styles.Overworld.prompt.Render("> ")
	m.refreshLog()
	if m.shell != nil {
		m.syncShellInput()
		if m.shellEditor.Width() != 0 {
			m.styleShellEditor()
		}
		m.refreshShell()
		m.refreshShellReader()
		m.refreshMessenger()
	}
	return true
}

func (m *Model) appendLog(role textRole, text string) {
	if text == "" {
		return
	}
	m.entries = append(m.entries, textElement{Role: role, Text: text})
}

func renderLog(entries []textElement, styles overworldStyles) string {
	lines := make([]string, 0, len(entries))
	for _, entry := range entries {
		style := styles.logBody
		switch entry.Role {
		case textEcho:
			style = styles.echo
		case textDim:
			style = styles.dim
		}
		lines = append(lines, style.Render(entry.Text))
	}
	return strings.Join(lines, "\n")
}

func rawLog(entries []textElement) string {
	lines := make([]string, 0, len(entries))
	for _, entry := range entries {
		lines = append(lines, entry.Text)
	}
	return strings.Join(lines, "\n")
}
