// Command editor is the navigation prototype for the full game editor.
//
// Run it from any directory with:
//
//	go run github.com/pabloduke/paws-in-the-machine/cmd/editor
//
// Or, from the repository root:
//
//	go run ./cmd/editor
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	if _, err := tea.NewProgram(newModel(), tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "editor:", err)
		os.Exit(1)
	}
}
