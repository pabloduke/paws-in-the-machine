package ui

import "github.com/charmbracelet/lipgloss"

// Terminal styles (docs/systems/hacking.md) — the hacking session's
// own look, separate from the overworld's neon in styles.go so the
// two worlds can diverge without stepping on each other. Consumed
// only by hacking_ui.go.
var (
	termGreen = lipgloss.Color("#7CFF7C")
	termDim   = lipgloss.Color("#2E7D32")
	termDark  = lipgloss.Color("#020802")

	termEchoStyle   = lipgloss.NewStyle().Foreground(termGreen).Background(termDark).Bold(true)
	termPromptStyle = lipgloss.NewStyle().Foreground(termGreen).Background(termDark)
	termDimStyle    = lipgloss.NewStyle().Foreground(termDim).Background(termDark)
	termOutputStyle = lipgloss.NewStyle().Foreground(termGreen).Background(termDark)

	termPanelStyle = lipgloss.NewStyle().
			Border(lipgloss.ThickBorder()).
			BorderForeground(termDim).
			Background(termDark).
			Foreground(termGreen).
			Padding(0, 1)
	termPanelFocusStyle = termPanelStyle.BorderForeground(termGreen)
	termPanelTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(termGreen).Background(termDark)
	termTitleStyle      = lipgloss.NewStyle().Bold(true).Foreground(termGreen).Background(termDark)
)
