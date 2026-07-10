package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pabloduke/paws-in-the-machine/internal/engine"
	"github.com/pabloduke/paws-in-the-machine/internal/game"
	"github.com/pabloduke/paws-in-the-machine/internal/ui"
)

func main() {
	eng := engine.New(game.NewWorld())
	mod := ui.New(eng, game.Intro)
	mod.BootIntoDeck() // the game opens at the terminal (docs/draft.md)
	program := tea.NewProgram(mod, tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
