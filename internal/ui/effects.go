package ui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/rivo/uniseg"
)

// inlineEffect is immutable, theme-owned configuration for a one-line text
// effect. The values are deliberately data rather than render state so the
// existing presentation phase remains the only animation clock.
type inlineEffect struct {
	enabled        bool
	start          rgbColor
	end            rgbColor
	highlight      rgbColor
	highlightWidth int
	period         int
	direction      int
}

type rgbColor struct {
	r uint8
	g uint8
	b uint8
}

// renderInlineEffect applies a horizontal truecolor gradient and a narrow
// phase-driven specular highlight. It styles whole grapheme clusters and
// advances by terminal cells, preserving the input text and display width.
// Disabled effects take the original single-style rendering path.
func renderInlineEffect(text string, phase int, base lipgloss.Style, effect inlineEffect) string {
	if !effect.enabled || text == "" {
		return base.Render(text)
	}

	totalWidth := uniseg.StringWidth(text)
	if totalWidth < 1 {
		return base.Render(text)
	}

	period := effect.period
	if period < 1 {
		period = 1
	}
	step := phase % period
	if step < 0 {
		step += period
	}
	progress := 0.0
	if period > 1 {
		progress = float64(step) / float64(period-1)
	}
	if effect.direction < 0 {
		progress = 1 - progress
	}
	highlightCenter := 0.5
	if totalWidth > 1 {
		highlightCenter += progress * float64(totalWidth-1)
	}

	var rendered strings.Builder
	graphemes := uniseg.NewGraphemes(text)
	cell := 0
	for graphemes.Next() {
		cluster := graphemes.Str()
		clusterWidth := uniseg.StringWidth(cluster)
		center := float64(cell) + float64(clusterWidth)/2
		if clusterWidth == 0 {
			center = float64(cell)
		}

		gradientPosition := 0.0
		if totalWidth > 1 {
			gradientPosition = (center - 0.5) / float64(totalWidth-1)
		}
		color := mixRGB(effect.start, effect.end, gradientPosition)
		if effect.highlightWidth > 0 {
			distance := math.Abs(center - highlightCenter)
			strength := 1 - distance/float64(effect.highlightWidth)
			color = mixRGB(color, effect.highlight, strength)
		}

		rendered.WriteString(base.Foreground(lipgloss.Color(color.hex())).Render(cluster))
		cell += clusterWidth
	}
	return rendered.String()
}

func mixRGB(from, to rgbColor, amount float64) rgbColor {
	if amount < 0 {
		amount = 0
	}
	if amount > 1 {
		amount = 1
	}
	mix := func(a, b uint8) uint8 {
		return uint8(math.Round(float64(a) + (float64(b)-float64(a))*amount))
	}
	return rgbColor{r: mix(from.r, to.r), g: mix(from.g, to.g), b: mix(from.b, to.b)}
}

func (c rgbColor) hex() string {
	return fmt.Sprintf("#%02x%02x%02x", c.r, c.g, c.b)
}
