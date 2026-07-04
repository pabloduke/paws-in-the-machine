package ui

import "github.com/charmbracelet/lipgloss"

const (
	leftPanelWidth  = 24
	rightPanelWidth = 24
	logHeight       = 8 // LOG panel total height, border included
	promptHeight    = 1
)

var (
	bodyStyle   = lipgloss.NewStyle().Padding(0, 1)
	echoStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true)
	promptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("8")).
			Padding(0, 1)
	panelFocusStyle = panelStyle.BorderForeground(lipgloss.Color("5"))
	panelTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("5"))
	roomTitleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("3"))
	hubSelStyle     = lipgloss.NewStyle().Reverse(true)
	// Deck rows in the city panel get terminal-green, set apart from
	// the hubs — a log-in target, not a place you walk to.
	deckTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2"))
	deckRowStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
)
