package main

import (
	"log"
	"sentinel/internal/backend/docker"
	"sentinel/internal/config"
	tui "sentinel/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	s := &config.YamlConfig{}
	r := &docker.ServiceRuntime{}
	d := &config.ServiceDef{}

	p := tea.NewProgram(tui.InitialModel(s, r, d), tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		log.Fatalf("Error starting Sentinel: %v", err)
	}
}
