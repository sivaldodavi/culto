package main

import (
	"fmt"
	"os"

	"culto/internal/app"
	"culto/internal/culto"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	program := tea.NewProgram(app.New(culto.DefaultConfig()), tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "culto:", err)
		os.Exit(1)
	}
}
