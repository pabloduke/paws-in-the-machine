package ui

import "github.com/charmbracelet/lipgloss"

const (
	leftPanelWidth  = 24
	rightPanelWidth = 24
	logHeight       = 8 // LOG panel total height, border included
	promptHeight    = 1
)

// Overworld palette: wet neon city at night. The rule is *chrome
// glows, prose doesn't* — UI furniture (titles, borders, focus) takes
// the neon; story text stays bone-plain so the words carry the scene.
// The terminal has its own separate set (styles_terminal.go); deck
// rows below are that world leaking through. Truecolor values degrade
// gracefully on 256/16-color terminals via lipgloss.
var (
	neonMagenta = lipgloss.Color("#ff2a6d") // signage, titles, focus
	neonCyan    = lipgloss.Color("#05d9e8") // player's own light: echoes, selection
	sodiumAmber = lipgloss.Color("#f7b32b") // streetlight: room names, warm beats
	rainGrey    = lipgloss.Color("#5c6773") // drizzle, hints, system whispers
	bone        = lipgloss.Color("#c9d1d9") // body prose — no glow
	phosphor    = lipgloss.Color("2")       // the terminal's green, leaking through
)

// drizzleBorder is a normal square border whose top edge streaks —
// rain on the window above the transcript.
var drizzleBorder = func() lipgloss.Border {
	b := lipgloss.NormalBorder()
	b.Top = "─·── ─·─ ──·─ ── ·─ "
	return b
}()

var (
	bodyStyle   = lipgloss.NewStyle().Padding(0, 1).Foreground(bone)
	echoStyle   = lipgloss.NewStyle().Foreground(neonCyan).Bold(true)
	promptStyle = lipgloss.NewStyle().Foreground(neonMagenta)
	dimStyle    = lipgloss.NewStyle().Foreground(rainGrey)
	rainStyle   = lipgloss.NewStyle().Foreground(rainGrey).Faint(true)

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(rainGrey).
			Padding(0, 1)
	panelFocusStyle = panelStyle.BorderForeground(neonMagenta)
	logPanelStyle   = lipgloss.NewStyle().
			Border(drizzleBorder).
			BorderForeground(rainGrey).
			Padding(0, 1)
	panelTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(neonMagenta)
	roomTitleStyle  = lipgloss.NewStyle().Bold(true).Foreground(sodiumAmber)
	hubSelStyle     = lipgloss.NewStyle().Reverse(true)
	// Deck rows in the city panel get terminal-green, set apart from
	// the hubs — a log-in target, not a place you walk to.
	deckTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(phosphor)
	deckRowStyle   = lipgloss.NewStyle().Foreground(phosphor)

	// Faded neon (the room sign, rain.go): a washed-out amber for the
	// sign's glow halo and its flicker frames.
	fadedAmber    = lipgloss.Color("#7d5a16")
	signGlowStyle = lipgloss.NewStyle().Foreground(fadedAmber)
	signFadeStyle = lipgloss.NewStyle().Bold(true).Foreground(fadedAmber)

	// Rain opacity ladder ('[' dims, ']' brightens; level 0 renders
	// nothing). Each step is the head/mid/tail shades of a drop's
	// fade; defaultRainLevel is the shipped look. Index 0 is unused —
	// rainField returns early instead.
	rainShades = [...][3]lipgloss.Style{
		{},
		shadeRow("#5a5e62", "#292e34", "#1b1f23"),
		shadeRow("#8d9298", "#404850", "#2b3037"),
		shadeRow("#c9d1d9", "#5c6773", "#3d454e"),
		shadeRow("#f1faff", "#6e7c8a", "#49535e"),
	}
)

const (
	defaultRainLevel = 3
	maxRainLevel     = len(rainShades) - 1
)

// shadeRow builds one head/mid/tail step of the rain ladder.
func shadeRow(head, mid, tail string) [3]lipgloss.Style {
	return [3]lipgloss.Style{
		lipgloss.NewStyle().Foreground(lipgloss.Color(head)),
		lipgloss.NewStyle().Foreground(lipgloss.Color(mid)),
		lipgloss.NewStyle().Foreground(lipgloss.Color(tail)),
	}
}
