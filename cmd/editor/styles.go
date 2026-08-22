package main

import "github.com/charmbracelet/lipgloss"

var (
	cyberBackground = lipgloss.Color("#02050a")
	cyberPanel      = lipgloss.Color("#07111a")
	cyberCyan       = lipgloss.Color("#39f6ff")
	cyberMagenta    = lipgloss.Color("#ff3cac")
	cyberWhite      = lipgloss.Color("#eaffff")
	cyberMuted      = lipgloss.Color("#7596a5")
	cyberYellow     = lipgloss.Color("#ffe66d")

	brandStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(cyberCyan)
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(cyberMagenta)
	menuStyle = lipgloss.NewStyle().
			Foreground(cyberWhite)
	selectedMenuStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(cyberBackground).
				Background(cyberCyan)
	selectorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(cyberMagenta)
	fieldLabelStyle = lipgloss.NewStyle().
			Foreground(cyberMuted)
	selectedFieldLabelStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(cyberCyan)
	choiceStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(cyberYellow)
	inputTextStyle = lipgloss.NewStyle().
			Foreground(cyberWhite)
	inputCursorStyle = lipgloss.NewStyle().
				Foreground(cyberBackground).
				Background(cyberMagenta)
)
