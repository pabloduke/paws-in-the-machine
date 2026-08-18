// Command editor is the visual chart editor (docs/systems/charts.md).
//
// It edits the same content file the game embeds, so there is no
// generated Go to clobber and no second source of truth. Place rooms on
// a node's grid, page through z and w slices, and watch exits derive
// from adjacency as you go — the geometry the game will actually use,
// shown while you author it.
//
//	go run ./cmd/editor [path/to/charts.json]
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

const defaultPath = "internal/game/content/charts.json"

func main() {
	path := defaultPath
	if len(os.Args) > 1 {
		path = os.Args[1]
	}

	m, err := newModel(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "editor:", err)
		os.Exit(1)
	}

	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "editor:", err)
		os.Exit(1)
	}
}
