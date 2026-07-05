package ui

import "github.com/charmbracelet/lipgloss"

// Terminal styles (docs/systems/hacking.md) — the hacking session's
// own look, separate from the overworld's neon in styles.go so the
// two worlds can diverge without stepping on each other. Consumed
// only by hacking_ui.go.
var (
	termEchoStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true)
	termPromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
	termDimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	termPanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("8")).
			Padding(0, 1)
	termPanelFocusStyle = termPanelStyle.BorderForeground(lipgloss.Color("5"))
	termPanelTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("5"))
	termTitleStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("3"))
)
