package main

import (
	"log"
	"sentinel/internal/config"
	tui "sentinel/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	s := &config.YamlConfig{}
	p := tea.NewProgram(tui.InitialModel(s), tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		log.Fatalf("Error starting Sentinel: %v", err)
	}
}
