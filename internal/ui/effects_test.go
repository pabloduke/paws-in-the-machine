package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/pabloduke/paws-in-the-machine/internal/engine"
	"github.com/pabloduke/paws-in-the-machine/internal/game"
)

func testInlineEffect() inlineEffect {
	return inlineEffect{
		enabled:        true,
		start:          rgbColor{r: 0, g: 0, b: 0},
		end:            rgbColor{r: 0, g: 0, b: 255},
		highlight:      rgbColor{r: 255, g: 255, b: 255},
		highlightWidth: 1,
		period:         5,
		direction:      1,
	}
}

func TestInlineEffectIsDeterministicAndMovesSpecularHighlight(t *testing.T) {
	profile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(profile) })

	base := lipgloss.NewStyle()
	effect := testInlineEffect()
	atStart := renderInlineEffect("ABCDE", 0, base, effect)
	if again := renderInlineEffect("ABCDE", 0, base, effect); again != atStart {
		t.Fatal("the same text, phase, and effect produced different output")
	}
	atMiddle := renderInlineEffect("ABCDE", 2, base, effect)
	atEnd := renderInlineEffect("ABCDE", 4, base, effect)
	if atStart == atMiddle || atMiddle == atEnd || atStart == atEnd {
		t.Fatal("specular output did not change as its phase moved")
	}
	if !strings.Contains(atStart, "\x1b[38;2;255;255;255mA") {
		t.Fatalf("phase 0 did not highlight the first cell in truecolor: %q", atStart)
	}
	if !strings.Contains(atMiddle, "\x1b[38;2;255;255;255mC") {
		t.Fatalf("middle phase did not highlight the middle cell in truecolor: %q", atMiddle)
	}
	if !strings.Contains(atEnd, "\x1b[38;2;255;255;255mE") {
		t.Fatalf("final phase did not highlight the last cell in truecolor: %q", atEnd)
	}
	if !strings.Contains(atMiddle, "\x1b[38;2;0;0;0mA") ||
		!strings.Contains(atMiddle, "\x1b[38;2;0;0;255mE") {
		t.Fatalf("truecolor gradient endpoints were not rendered: %q", atMiddle)
	}
}

func TestInlineEffectPreservesGraphemesTextAndWidth(t *testing.T) {
	profile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(profile) })

	text := "A e\u0301 界 "
	for phase := 0; phase < 9; phase++ {
		rendered := renderInlineEffect(text, phase, lipgloss.NewStyle().Bold(true), testInlineEffect())
		if plain := stripANSI(rendered); plain != text {
			t.Fatalf("phase %d changed raw grapheme text: got %q want %q", phase, plain, text)
		}
		if got, want := lipgloss.Width(rendered), lipgloss.Width(text); got != want {
			t.Fatalf("phase %d changed display width: got %d want %d", phase, got, want)
		}
	}
}

func TestDisabledInlineEffectUsesBaselineStyle(t *testing.T) {
	base := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#123456"))
	text := "◈ TITLE"
	if got, want := renderInlineEffect(text, 99, base, inlineEffect{}), base.Render(text); got != want {
		t.Fatalf("disabled effect changed baseline rendering:\n got %q\nwant %q", got, want)
	}
}

func TestInlineEffectFallsBackUnderANSIProfile(t *testing.T) {
	profile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.ANSI)
	t.Cleanup(func() { lipgloss.SetColorProfile(profile) })

	text := "A e\u0301 界 "
	rendered := renderInlineEffect(text, 3, lipgloss.NewStyle(), testInlineEffect())
	if plain := stripANSI(rendered); plain != text {
		t.Fatalf("ANSI fallback changed raw text: got %q want %q", plain, text)
	}
	if got, want := lipgloss.Width(rendered), lipgloss.Width(text); got != want {
		t.Fatalf("ANSI fallback changed display width: got %d want %d", got, want)
	}
}

func TestChromeEffectsIntegrateWithBothTitlesAndRuntimeSwitching(t *testing.T) {
	profile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(profile) })

	m := New(engine.New(game.NewWorld()), "intro")
	if m.presentation().Overworld.roomTitleEffect.enabled ||
		m.presentation().Terminal.local.titleEffect.enabled {
		t.Fatal("wet-neon unexpectedly enabled chrome title effects")
	}
	if !m.setTheme("chrome") {
		t.Fatal("chrome theme was not registered")
	}
	if !m.presentation().Overworld.roomTitleEffect.enabled ||
		!m.presentation().Terminal.local.titleEffect.enabled ||
		!m.presentation().Terminal.remote.titleEffect.enabled {
		t.Fatal("chrome did not enable both overworld and terminal title effects")
	}

	// neonSign is the overworld integration. Find adjacent non-flicker frames;
	// their raw title and geometry must be stable while the shimmer changes.
	var signA, signB string
	for phase := 0; phase < flickerMod-1; phase++ {
		m.phase = phase
		a := m.neonSign("ROOM")
		m.phase = phase + 1
		b := m.neonSign("ROOM")
		if stripANSI(a) == stripANSI(b) && a != b {
			signA, signB = a, b
			break
		}
	}
	if signA == "" || signB == "" {
		t.Fatal("overworld room title did not produce adjacent shimmer frames")
	}
	if stripANSI(signA) != "▒ ◈ ROOM ▒" || stripANSI(signB) != "▒ ◈ ROOM ▒" {
		t.Fatalf("overworld effect changed title text: %q / %q", stripANSI(signA), stripANSI(signB))
	}
	if lipgloss.Width(signA) != lipgloss.Width(signB) {
		t.Fatal("overworld shimmer changed room-sign width")
	}

	// renderTerminalTitle is the hacking-terminal integration used by Screen.
	theme := m.presentation().Terminal.local
	terminalA := renderTerminalTitle("CYBERDECK // DECK", 30, 0, theme)
	terminalB := renderTerminalTitle("CYBERDECK // DECK", 30, 12, theme)
	if terminalA == terminalB {
		t.Fatal("terminal title did not shimmer across phases")
	}
	if stripANSI(terminalA) != stripANSI(terminalB) || lipgloss.Width(terminalA) != 30 || lipgloss.Width(terminalB) != 30 {
		t.Fatal("terminal shimmer changed title text, width, or alignment")
	}

	if !m.setTheme(defaultThemeID) {
		t.Fatal("could not switch back to wet-neon")
	}
	if m.presentation().Overworld.roomTitleEffect.enabled ||
		m.presentation().Terminal.local.titleEffect.enabled {
		t.Fatal("runtime switch back to wet-neon retained chrome effects")
	}
	wetTheme := m.presentation().Terminal.local
	if got, want := renderTerminalTitle("CYBERDECK // DECK", 30, 12, wetTheme),
		wetTheme.titleStyle().Width(30).MaxWidth(30).Render("CYBERDECK // DECK"); got != want {
		t.Fatal("wet-neon terminal title did not retain its baseline render path")
	}
}
