package ui

import "github.com/charmbracelet/lipgloss"

// Terminal styles (docs/systems/hacking.md) — the hacking session's
// own look, separate from the overworld's neon in styles.go so the
// two worlds can diverge without stepping on each other. Consumed
// only by hacking_ui.go.
var (
	termGreen     = lipgloss.Color("#7CFF7C")
	termDim       = lipgloss.Color("#2E7D32")
	termDark      = lipgloss.Color("#020802")
	termAmber     = lipgloss.Color("#FFB000")
	termAmberDim  = lipgloss.Color("#8A5A00")
	termAmberDark = lipgloss.Color("#0B0700")

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

type terminalTheme struct {
	bright lipgloss.Color
	dim    lipgloss.Color
	dark   lipgloss.Color
}

var (
	localTerminalTheme  = terminalTheme{bright: termGreen, dim: termDim, dark: termDark}
	remoteTerminalTheme = terminalTheme{bright: termAmber, dim: termAmberDim, dark: termAmberDark}
)

func (t terminalTheme) echoStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.bright).Background(t.dark).Bold(true)
}

func (t terminalTheme) promptStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.bright).Background(t.dark)
}

func (t terminalTheme) dimStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.dim).Background(t.dark)
}

func (t terminalTheme) outputStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.bright).Background(t.dark)
}

func (t terminalTheme) panelStyle(focused bool) lipgloss.Style {
	border := t.dim
	if focused {
		border = t.bright
	}
	return lipgloss.NewStyle().
		Border(lipgloss.ThickBorder()).
		BorderForeground(border).
		Background(t.dark).
		Foreground(t.bright).
		Padding(0, 1)
}

func (t terminalTheme) titleStyle() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(t.bright).Background(t.dark)
}
