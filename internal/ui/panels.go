package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/pabloduke/paws-in-the-machine/internal/engine"
)

// flickerMod: roughly one frame in this many, the room sign dips to
// its faded shade — a failing neon tube. At a 100ms frame, 53 is a
// dip every ~5s.
const flickerMod = 53

// neonSign renders the room name as a neon sign: faded-glow halo at
// each end, and a rare phase-based flicker. Style only ever changes;
// the text itself is constant.
func (m Model) neonSign(name string) string {
	h := uint32(2166136261)
	for _, r := range m.eng.World.Room().ID {
		h = (h ^ uint32(r)) * 16777619
	}
	title := roomTitleStyle
	if (int(h%flickerMod)+m.phase)%flickerMod == 0 {
		title = signFadeStyle
	}
	halo := signGlowStyle.Render("▒")
	return halo + " " + title.Render("◈ "+name) + " " + halo
}

// roomPanel is the live room view: title from the room name, body from
// world state — always current, never a transcript. Dead space below
// the prose carries the rain (rain.go), driven by the render phase.
func (m Model) roomPanel() string {
	width := m.width - leftPanelWidth - rightPanelWidth
	name, body, _ := strings.Cut(engine.Look(m.eng.World), "\n\n")
	content := m.neonSign(strings.ToUpper(name))
	if body != "" {
		content += "\n\n" + body
	}
	innerH := m.mainRowHeight() - 2
	prose := bodyStyle.Width(width - 4).Render(content)
	if free := innerH - lipgloss.Height(prose); free >= 3 {
		// One blank gap line, then rain to the bottom of the panel.
		rows := rainField(m.eng.World.Room().ID, m.phase, width-6, free-1)
		prose += "\n" + bodyStyle.Width(width-4).
			Render("\n"+strings.Join(rows, "\n"))
	}
	return panelStyle.Width(width - 2).Height(innerH).Render(prose)
}

// rightPanel: BUDDY (level and XP — the full sheet and inventory live
// in their modals), YOU SEE (only obvious entities — scenery is
// discovered through prose), and EXITS. See docs/systems/visibility.md.
func (m Model) rightPanel() string {
	w := m.eng.World

	var b strings.Builder
	b.WriteString(panelTitleStyle.Render("▸ BUDDY"))
	b.WriteString(fmt.Sprintf("\n  Lv %d · XP %d/%d", w.Level, w.XP, w.NextLevelCost()))
	if w.StatPoints > 0 {
		b.WriteString("\n  " + dimStyle.Render(fmt.Sprintf("● %d to spend", w.StatPoints)))
	}

	b.WriteString("\n\n" + panelTitleStyle.Render("▸ YOU SEE"))
	for _, e := range w.Obvious() {
		b.WriteString("\n  " + engine.DisplayName(w, e))
	}
	if x, ok := engine.Part[engine.Exits](w.Room()); ok && len(x.Dirs) > 0 {
		b.WriteString("\n\n" + panelTitleStyle.Render("▸ EXITS"))
		dirs := make([]string, 0, len(x.Dirs))
		for dir := range x.Dirs {
			dirs = append(dirs, dir)
		}
		sort.Strings(dirs)
		for _, dir := range dirs {
			b.WriteString("\n  " + dir)
		}
	}
	return panelStyle.Width(rightPanelWidth - 2).Height(m.mainRowHeight() - 2).Render(b.String())
}

// logPanel renders the transcript viewport.
func (m Model) logPanel() string {
	content := panelTitleStyle.Render("▸ LOG") + "\n" + m.log.View()
	return logPanelStyle.Width(m.width - 2).Render(content)
}
