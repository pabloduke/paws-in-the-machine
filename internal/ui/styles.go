package ui

import "github.com/charmbracelet/lipgloss"

const (
	leftPanelWidth  = 24
	rightPanelWidth = 24
	logHeight       = 8 // LOG panel total height, border included
	promptHeight    = 1
)

// overworldStyles is the rendered vocabulary for the room UI. Callers ask
// for a semantic style (title, echo, dim text); they do not own colors.
// Keeping the complete set on a theme lets the UI swap appearances without
// changing game content or storing ANSI in the transcript.
type overworldStyles struct {
	body       lipgloss.Style
	logBody    lipgloss.Style
	echo       lipgloss.Style
	prompt     lipgloss.Style
	dim        lipgloss.Style
	rain       lipgloss.Style
	panel      lipgloss.Style
	panelFocus lipgloss.Style
	logPanel   lipgloss.Style
	panelTitle lipgloss.Style
	roomTitle  lipgloss.Style
	selection  lipgloss.Style
	deckTitle  lipgloss.Style
	deckRow    lipgloss.Style
	signGlow   lipgloss.Style
	signFade   lipgloss.Style
	rainShades [5][3]lipgloss.Style
}

// defaultOverworldStyles reproduces the shipped wet-neon appearance. It is
// deliberately a constructor rather than package-global style state: every
// presentation theme owns its complete rendering vocabulary.
func defaultOverworldStyles() overworldStyles {
	neonMagenta := lipgloss.Color("#ff2a6d")
	neonCyan := lipgloss.Color("#05d9e8")
	sodiumAmber := lipgloss.Color("#f7b32b")
	rainGrey := lipgloss.Color("#5c6773")
	bone := lipgloss.Color("#c9d1d9")
	phosphor := lipgloss.Color("2")
	fadedAmber := lipgloss.Color("#7d5a16")

	// A normal square border whose top edge streaks like rain on glass.
	drizzle := lipgloss.NormalBorder()
	drizzle.Top = "─·── ─·─ ──·─ ── ·─ "

	panel := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(rainGrey).
		Padding(0, 1)

	return overworldStyles{
		body:       lipgloss.NewStyle().Padding(0, 1).Foreground(bone),
		logBody:    lipgloss.NewStyle(),
		echo:       lipgloss.NewStyle().Foreground(neonCyan).Bold(true),
		prompt:     lipgloss.NewStyle().Foreground(neonMagenta),
		dim:        lipgloss.NewStyle().Foreground(rainGrey),
		rain:       lipgloss.NewStyle().Foreground(rainGrey).Faint(true),
		panel:      panel,
		panelFocus: panel.BorderForeground(neonMagenta),
		logPanel: lipgloss.NewStyle().
			Border(drizzle).
			BorderForeground(rainGrey).
			Padding(0, 1),
		panelTitle: lipgloss.NewStyle().Bold(true).Foreground(neonMagenta),
		roomTitle:  lipgloss.NewStyle().Bold(true).Foreground(sodiumAmber),
		selection:  lipgloss.NewStyle().Reverse(true),
		deckTitle:  lipgloss.NewStyle().Bold(true).Foreground(phosphor),
		deckRow:    lipgloss.NewStyle().Foreground(phosphor),
		signGlow:   lipgloss.NewStyle().Foreground(fadedAmber),
		signFade:   lipgloss.NewStyle().Bold(true).Foreground(fadedAmber),
		rainShades: [5][3]lipgloss.Style{
			{},
			shadeRow("#5a5e62", "#292e34", "#1b1f23"),
			shadeRow("#8d9298", "#404850", "#2b3037"),
			shadeRow("#c9d1d9", "#5c6773", "#3d454e"),
			shadeRow("#f1faff", "#6e7c8a", "#49535e"),
		},
	}
}

const (
	defaultRainLevel = 3
	maxRainLevel     = 4
)

func shadeRow(head, mid, tail string) [3]lipgloss.Style {
	return [3]lipgloss.Style{
		lipgloss.NewStyle().Foreground(lipgloss.Color(head)),
		lipgloss.NewStyle().Foreground(lipgloss.Color(mid)),
		lipgloss.NewStyle().Foreground(lipgloss.Color(tail)),
	}
}
