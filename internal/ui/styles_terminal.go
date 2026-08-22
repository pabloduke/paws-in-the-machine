package ui

import "github.com/charmbracelet/lipgloss"

// terminalTheme is one semantic terminal palette. A presentation theme owns
// a local, remote, and side-panel palette so connection state remains
// meaningful while the overall appearance is replaceable.
type terminalTheme struct {
	bright      lipgloss.Color
	dim         lipgloss.Color
	dark        lipgloss.Color
	titleEffect inlineEffect
}

type terminalStyles struct {
	local  terminalTheme
	remote terminalTheme
	panel  terminalTheme
}

func defaultTerminalStyles() terminalStyles {
	green := terminalTheme{
		bright: lipgloss.Color("#7CFF7C"),
		dim:    lipgloss.Color("#2E7D32"),
		dark:   lipgloss.Color("#020802"),
	}
	return terminalStyles{
		local: green,
		remote: terminalTheme{
			bright: lipgloss.Color("#FFB000"),
			dim:    lipgloss.Color("#8A5A00"),
			dark:   lipgloss.Color("#0B0700"),
		},
		panel: green,
	}
}

func chromeTerminalStyles() terminalStyles {
	cyan := terminalTheme{
		bright: lipgloss.Color("#73f7ff"),
		dim:    lipgloss.Color("#247c86"),
		dark:   lipgloss.Color("#02080b"),
		// Provisional polished-chrome defaults, isolated here for tuning.
		titleEffect: inlineEffect{
			enabled:        true,
			start:          rgbColor{r: 38, g: 151, b: 166},
			end:            rgbColor{r: 115, g: 247, b: 255},
			highlight:      rgbColor{r: 255, g: 255, b: 255},
			highlightWidth: 2,
			period:         30,
			direction:      1,
		},
	}
	return terminalStyles{
		local: cyan,
		remote: terminalTheme{
			bright: lipgloss.Color("#ff4fd8"),
			dim:    lipgloss.Color("#8a285f"),
			dark:   lipgloss.Color("#0b0208"),
			titleEffect: inlineEffect{
				enabled:        true,
				start:          rgbColor{r: 139, g: 44, b: 104},
				end:            rgbColor{r: 255, g: 79, b: 216},
				highlight:      rgbColor{r: 255, g: 255, b: 255},
				highlightWidth: 2,
				period:         30,
				direction:      1,
			},
		},
		panel: cyan,
	}
}

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
