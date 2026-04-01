package main

import (
	"flag"
	"hydra-config-mixer/internal/config"
	"hydra-config-mixer/internal/ui"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	confDir := flag.String("conf", "conf", "Hydra configディレクトリ")
	srcDir := flag.String("src", "models", "Pythonソースディレクトリ")
	flag.Parse()

	ui.ConfDir = *confDir
	ui.SrcDir = *srcDir

	files, err := config.LoadYamlFiles(*confDir)
	if err != nil {
		os.Exit(1)
	}

	p := tea.NewProgram(ui.New(files, *confDir), tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		os.Exit(1)
	}
}
