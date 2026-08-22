package ui

import (
	"fmt"
	"sort"
	"strings"
)

const themeUsage = "(Placeholder) usage: theme [list|next|previous|<id>]"

// handleThemeCommand owns the local display command before input reaches the
// fake shell. A theme is a machine-local UI preference even when the player
// is connected to a remote host; it never becomes host or world state.
func (m *Model) handleThemeCommand(line string) (string, bool) {
	args := strings.Fields(line)
	if len(args) == 0 || args[0] != "theme" {
		return "", false
	}
	if len(args) == 1 {
		return "(Placeholder) theme: " + m.themeID + "\n" + themeUsage, true
	}
	if len(args) != 2 {
		return themeUsage, true
	}

	switch args[1] {
	case "list":
		var lines []string
		for _, id := range ThemeIDs() {
			marker := "  "
			if id == m.themeID {
				marker = "* "
			}
			lines = append(lines, marker+id)
		}
		return "(Placeholder) themes:\n" + strings.Join(lines, "\n"), true
	case "next":
		return m.cycleTheme(1), true
	case "previous", "prev":
		return m.cycleTheme(-1), true
	default:
		if !m.setTheme(args[1]) {
			return fmt.Sprintf("(Placeholder) theme: unknown theme %q\navailable: %s",
				args[1], strings.Join(ThemeIDs(), ", ")), true
		}
		return "(Placeholder) theme: " + m.themeID, true
	}
}

func (m *Model) cycleTheme(delta int) string {
	ids := ThemeIDs()
	if len(ids) == 0 {
		return "(Placeholder) theme: no themes registered"
	}
	index := 0
	for i, id := range ids {
		if id == m.themeID {
			index = i
			break
		}
	}
	index = (index + delta + len(ids)) % len(ids)
	m.setTheme(ids[index])
	return "(Placeholder) theme: " + m.themeID
}

// completeShellLine merges the UI-local command vocabulary with the fake
// shell's completion results without adding presentation concerns to the
// hacking system.
func (m *Model) completeShellLine(line string) (string, []string) {
	if strings.HasPrefix(line, "theme ") {
		active := strings.TrimPrefix(line, "theme ")
		if strings.ContainsAny(active, " \t") {
			return line, nil
		}
		candidates := append([]string{"list", "next", "previous"}, ThemeIDs()...)
		return completeFrom("theme ", active, candidates)
	}

	if !strings.ContainsAny(line, " \t") {
		_, shellMatches := m.shell.Complete(line)
		matches := append([]string{}, shellMatches...)
		if strings.HasPrefix("theme", line) {
			matches = append(matches, "theme")
		}
		return completeFrom("", line, uniqueSorted(matches))
	}
	return m.shell.Complete(line)
}

func completeFrom(before, active string, candidates []string) (string, []string) {
	var matches []string
	for _, candidate := range candidates {
		if strings.HasPrefix(candidate, active) {
			matches = append(matches, candidate)
		}
	}
	matches = uniqueSorted(matches)
	if len(matches) == 0 {
		return before + active, nil
	}
	completed := commonTextPrefix(matches)
	if len(matches) == 1 {
		completed += " "
	}
	return before + completed, matches
}

func uniqueSorted(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		if value != "" && !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	sort.Strings(out)
	return out
}

func commonTextPrefix(values []string) string {
	prefix := values[0]
	for _, value := range values[1:] {
		for !strings.HasPrefix(value, prefix) {
			prefix = prefix[:len(prefix)-1]
		}
	}
	return prefix
}
