package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"log"
	tui "sentinel/internal/ui"
)

func main() {
	p := tea.NewProgram(tui.InitialModel(), tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		log.Fatalf("Error starting Sentinel: %v", err)
	}
}
