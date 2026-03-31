package main

import (
	"hydra-config-mixer/internal/config"
	"hydra-config-mixer/internal/ui"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	files, err := config.LoadYamlFiles("conf")
	if err != nil {
		os.Exit(1)
	}

	p := tea.NewProgram(ui.New(files), tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		os.Exit(1)
	}
}
