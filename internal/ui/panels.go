package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pabloduke/paws-in-the-machine/internal/engine"
)

// roomPanel is the live room view: title from the room name, body from
// world state — always current, never a transcript.
func (m Model) roomPanel() string {
	width := m.width - leftPanelWidth - rightPanelWidth
	name, body, _ := strings.Cut(engine.Look(m.eng.World), "\n\n")
	content := roomTitleStyle.Render(strings.ToUpper(name))
	if body != "" {
		content += "\n\n" + body
	}
	return panelStyle.Width(width - 2).Height(m.mainRowHeight() - 2).
		Render(bodyStyle.Width(width - 4).Render(content))
}

// rightPanel: BUDDY (level and XP — the full sheet and inventory live
// in their modals), YOU SEE (only obvious entities — scenery is
// discovered through prose), and EXITS. See docs/systems/visibility.md.
func (m Model) rightPanel() string {
	w := m.eng.World

	var b strings.Builder
	b.WriteString(panelTitleStyle.Render("BUDDY"))
	b.WriteString(fmt.Sprintf("\n  Lv %d · XP %d/%d", w.Level, w.XP, w.NextLevelCost()))
	if w.StatPoints > 0 {
		b.WriteString("\n  " + dimStyle.Render(fmt.Sprintf("● %d to spend", w.StatPoints)))
	}

	b.WriteString("\n\n" + panelTitleStyle.Render("YOU SEE"))
	for _, e := range w.Obvious() {
		b.WriteString("\n  " + engine.DisplayName(w, e))
	}
	if x, ok := engine.Part[engine.Exits](w.Room()); ok && len(x.Dirs) > 0 {
		b.WriteString("\n\n" + panelTitleStyle.Render("EXITS"))
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
	content := panelTitleStyle.Render("LOG") + "\n" + m.log.View()
	return panelStyle.Width(m.width - 2).Render(content)
}
