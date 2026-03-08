package main

import (
	"log"
	"sentinel/internal/config"
	tui "sentinel/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	y := &config.YamlConfig{}
	d := &config.ServiceDef{}
	services := y.ReadFromConfigFile()

	p := tea.NewProgram(tui.InitialModel(y, d, services), tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		log.Fatalf("Error starting Sentinel: %v", err)
	}
}
