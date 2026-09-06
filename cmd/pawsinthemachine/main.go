package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	contentDir := flag.String("content-dir", "", "play one authored content directory")
	worldsDir := flag.String("worlds-dir", "", "directory holding editor-created worlds (defaults to <repo>/worlds)")
	flag.Parse()
	choices, err := discoverWorlds(*contentDir, *worldsDir)
	program := tea.NewProgram(newStartup(choices, err), tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
