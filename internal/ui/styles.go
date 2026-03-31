package ui

import "github.com/charmbracelet/lipgloss"

// --- スタイルの定義 ---
var (
	paneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2)

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("212")).
				Bold(true)

	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("78")).
			Bold(true).
			MarginBottom(1)
)
