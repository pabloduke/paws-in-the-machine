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

	crtGreen = lipgloss.Color("#7CFF7C")
	crtDim   = lipgloss.Color("#2E7D32")
	crtDark  = lipgloss.Color("#020802")

	crtPanelStyle = lipgloss.NewStyle().
			Border(lipgloss.ThickBorder()).
			BorderForeground(crtDim).
			Background(crtDark).
			Foreground(crtGreen).
			Padding(0, 1)
	crtPanelFocusStyle = crtPanelStyle.BorderForeground(crtGreen)
	crtTitleStyle      = lipgloss.NewStyle().Bold(true).Foreground(crtGreen).Background(crtDark)
	crtPanelTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(crtGreen).Background(crtDark)
	crtPromptStyle     = lipgloss.NewStyle().Foreground(crtGreen).Background(crtDark)
	crtEchoStyle       = lipgloss.NewStyle().Foreground(crtGreen).Background(crtDark).Bold(true)
	crtDimStyle        = lipgloss.NewStyle().Foreground(crtDim).Background(crtDark)
	crtOutputStyle     = lipgloss.NewStyle().Foreground(crtGreen).Background(crtDark)
)
