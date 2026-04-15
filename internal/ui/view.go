package ui

import (
	"fmt"
	"hydra-config-mixer/internal/config"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	if !m.ready {
		return "読み込み中..."
	}

	paneWidth := (m.width - 6) / 2
	paneHeight := m.height - 4

	// 左ペイン：リストまたは入力欄
	listStr := titleStyle.Render("📂 Config Files") + "\n"
	for i, file := range m.files {
		cursor := "  "
		if m.cursor == i {
			cursor = "> "
			listStr += selectedItemStyle.Render(cursor+strings.ReplaceAll(file, "\\", "/")) + "\n"
		} else {
			listStr += cursor + strings.ReplaceAll(file, "\\", "/") + "\n"
		}
	}

	// --- PythonファイルからYAML生成モードのUIの描画 ---
	if m.state == statePyFileSearch {
		listStr += "\n" + titleStyle.Render("🐍 Select Python File") + "\n"
		listStr += m.textInput.View() + "\n\n"
		for i, f := range m.filteredFiles {
			if i >= 10 {
				listStr += "  ...and more\n"
				break
			}
			if m.pyFileCursor == i {
				listStr += selectedItemStyle.Render("> "+f) + "\n"
			} else {
				listStr += "  " + f + "\n"
			}
		}
		listStr += "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("(↑/↓: 選択, Enter: 確定, Esc: キャンセル)")
	}

	// --- Pythonクラス選択モードのUIの描画 ---
	if m.state == statePyClassSelect {
		listStr += "\n" + titleStyle.Render("🐍 Select Class: "+m.selectedPyModule) + "\n\n"
		for i, c := range m.pyClasses {
			if m.pyClassCursor == i {
				listStr += selectedItemStyle.Render("> "+c) + "\n"
			} else {
				listStr += "  " + c + "\n"
			}
		}
		listStr += "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("(↑/↓: 選択, Enter: 確定, Esc: 戻る)")
	}

	// --- 入力モードのUIを追加表示 ---
	if m.state == stateInput {
		listStr += "\n" + titleStyle.Render("✨ Generate Config") + "\n"
		listStr += "Target Class Path:\n"
		listStr += m.textInput.View() + "\n\n"
		listStr += lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("(Enterで生成 / Escでキャンセル)")
	}

	// --- 保存先確認・編集モードのUIの描画 ---
	if m.state == stateSavePath {
		listStr += "\n" + titleStyle.Render("💾 Save Path") + "\n"
		listStr += m.textInput.View() + "\n\n"
		listStr += lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("(Enterで保存 / Escでクラス選択に戻る)")
	}

	// --- オートコンプリートUIの描画 ---
	if m.state == stateAutoComplete {
		listStr += "\n" + titleStyle.Render("🔍 Embed Config (Auto-complete)") + "\n"
		listStr += m.textInput.View() + "\n\n"

		// 絞り込まれた候補をリスト表示（最大10件程度にしておく）
		for i, f := range m.filteredFiles {
			if i >= 10 {
				listStr += "  ...and more\n"
				break
			}
			cursor := "  "
			if m.autoCompleteCursor == i {
				cursor = "> "
				listStr += selectedItemStyle.Render(cursor+f) + "\n"
			} else {
				listStr += cursor + f + "\n"
			}
		}
		listStr += "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("(↑/↓: 選択, Enter: 確定, Esc: キャンセル)")
	}

	// --- 複製モードのUIの描画 ---
	if m.state == stateClone {
		listStr += "\n" + titleStyle.Render("📋 Clone Config") + "\n"
		listStr += "New File Path (under conf/):\n"
		listStr += m.textInput.View() + "\n\n"
		listStr += lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("(Enterで複製 / Escでキャンセル)")
	}

	// --- 検索モードのUIの描画 ---
	if m.state == stateSearchFiles {
		listStr += "🔍 Search:\n"
		listStr += m.textInput.View() + "\n\n"
	} else if m.state == stateDeleteConfirm {
		if len(m.files) > 0 {
			target := strings.ReplaceAll(m.files[m.cursor], "\\", "/")
			listStr += "\n" + warningStyle.Render("🗑 削除しますか?") + "\n"
			listStr += target + "\n\n"
			listStr += lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("(y: 削除 / n, Esc: キャンセル)")
		}
	} else if m.state == stateList {
		listStr += lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("('/'で検索, 'n'で新規生成, 'a'で埋め込み, 'c'で複製, 'e'で編集, 'd'で削除)") + "\n\n"
	}

	if m.errMsg != "" {
		listStr += "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(m.errMsg)
	}

	// 右ペイン：プレビュー
	var rightPaneContent string

	if m.state == stateEditLineList || m.state == stateEditLineValue {
		var sb strings.Builder
		sb.WriteString(titleStyle.Render("✏️ Inline Edit: "+filepath.Base(m.editFile)) + "\n\n")

		// タイトル(2行) + 空白(2行) + フッター(2行) を除いた行数がコンテンツ領域
		contentHeight := paneHeight - 6
		if contentHeight < 1 {
			contentHeight = 1
		}

		// editCursor が常に中央付近に来るようにウィンドウをスライドさせる
		start := m.editCursor - contentHeight/2
		if start < 0 {
			start = 0
		}
		end := start + contentHeight
		if end > len(m.editLines) {
			end = len(m.editLines)
			start = end - contentHeight
			if start < 0 {
				start = 0
			}
		}

		for i := start; i < end; i++ {
			line := m.editLines[i]
			if i == m.editCursor {
				if m.state == stateEditLineValue {
					sb.WriteString(selectedItemStyle.Render(">> "+m.editLinePrefix+" ") + m.textInput.View() + "\n")
				} else {
					sb.WriteString(selectedItemStyle.Render(">  "+line) + "\n")
				}
			} else {
				sb.WriteString("   " + line + "\n")
			}
		}
		hint := "(↑/↓: 行選択  Enter: 編集/保存  Esc: 終了)"
		if isInDefaultsBlock(m.editLines, m.editCursor) {
			hint = "(↑/↓: 移動  K/J: 並び替え  x: 削除  Esc: 終了)"
		}
		sb.WriteString("\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(hint))
		rightPaneContent = sb.String()
	} else {
		var selectedFile string
		if len(m.files) > 0 {
			selectedFile = m.files[m.cursor]
		}
		previewTitle := titleStyle.Render("📄 Preview: " + filepath.Base(selectedFile))

		// 警告があればタイトルとYamlの間に赤文字で挿入
		warningBlock := ""
		if m.warningMsg != "" {
			warningBlock = warningStyle.Render(m.warningMsg)
		}

		if m.warningMsg != "" {
			rightPaneContent = lipgloss.JoinVertical(lipgloss.Left, previewTitle, warningBlock, m.viewport.View())
		} else {
			rightPaneContent = lipgloss.JoinVertical(lipgloss.Left, previewTitle, m.viewport.View())
		}
	}

	// paneHeight を超えないように行数をクランプする（上端が見切れるのを防ぐ）
	clamp := func(s string, max int) string {
		lines := strings.Split(s, "\n")
		if len(lines) > max {
			lines = lines[:max]
		}
		return strings.Join(lines, "\n")
	}

	// 左ペインを1つのペイン内でセパレーターで区切る
	// ファイルリスト: 上60% / 依存ツリー: 下40%
	depSectionHeight := paneHeight * 2 / 5
	if depSectionHeight < 6 {
		depSectionHeight = 6
	}
	filesSectionHeight := paneHeight - depSectionHeight - 1 // -1はセパレーター行

	separator := lipgloss.NewStyle().
		Foreground(lipgloss.Color("62")).
		Render(strings.Repeat("─", paneWidth-4))

	leftContent := clamp(listStr, filesSectionHeight) + "\n" +
		separator + "\n" +
		clamp(m.buildDepPaneContent(), depSectionHeight)

	// .Height() に頼らず両ペインを明示的に paneHeight 行にそろえる
	pad := func(s string, h int) string {
		lines := strings.Split(s, "\n")
		if len(lines) > h {
			lines = lines[:h]
		}
		for len(lines) < h {
			lines = append(lines, "")
		}
		return strings.Join(lines, "\n")
	}

	leftPane := paneStyle.Width(paneWidth).Render(pad(leftContent, paneHeight))
	rightPane := paneStyle.Width(paneWidth).Render(pad(clamp(rightPaneContent, paneHeight), paneHeight))

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
}

// buildDepPaneContent は左下ペインに常時表示する依存情報を組み立てる
func (m *Model) buildDepPaneContent() string {
	if len(m.files) == 0 {
		return ""
	}
	file := m.files[m.cursor]
	var sb strings.Builder

	sb.WriteString(titleStyle.Render("🌲 Dependencies") + "\n")

	deps := m.depForward[file]
	sb.WriteString(lipgloss.NewStyle().Bold(true).Render("Uses:") + "\n")
	if len(deps) == 0 {
		sb.WriteString("  (none)\n")
	} else {
		for i, dep := range deps {
			visited := map[string]bool{file: true}
			sb.WriteString(renderDepTree(dep, m.depForward, visited, "  ", i == len(deps)-1))
		}
	}

	sb.WriteString("\n")

	usedBy := m.depReverse[file]
	sb.WriteString(lipgloss.NewStyle().Bold(true).Render("Used by:") + "\n")
	if len(usedBy) == 0 {
		sb.WriteString("  (none)\n")
	} else {
		for _, f := range usedBy {
			sb.WriteString("  • " + strings.ReplaceAll(f, "\\", "/") + "\n")
		}
	}

	// 競合キーの表示
	conflicts := config.FindConflicts(file, m.depForward)
	if len(conflicts) > 0 {
		sb.WriteString("\n")
		conflictStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
		winnerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
		loserStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Strikethrough(true)
		sb.WriteString(conflictStyle.Render("⚠ Key Conflicts:") + "\n")
		for _, c := range conflicts {
			sb.WriteString(fmt.Sprintf("  %s\n", lipgloss.NewStyle().Bold(true).Render(c.Key)))
			for i, src := range c.Sources {
				base := filepath.Base(src.File)
				val := src.Value
				if val == "" {
					val = "(empty)"
				}
				entry := fmt.Sprintf("    %s: %s = %s", func() string {
					if i == len(c.Sources)-1 {
						return "✓"
					}
					return "✗"
				}(), base, val)
				if i == len(c.Sources)-1 {
					sb.WriteString(winnerStyle.Render(entry) + "\n")
				} else {
					sb.WriteString(loserStyle.Render(entry) + "\n")
				}
			}
		}
	}

	return sb.String()
}

// renderDepTree はファイルを起点とした依存ツリーをASCIIアートで返す（循環参照をvisitedで防止）
func renderDepTree(file string, forward map[string][]string, visited map[string]bool, prefix string, isLast bool) string {
	connector := "├── "
	childPrefix := prefix + "│   "
	if isLast {
		connector = "└── "
		childPrefix = prefix + "    "
	}
	line := prefix + connector + filepath.Base(file) + "\n"

	if visited[file] {
		return line
	}
	visited[file] = true
	deps := forward[file]
	for i, dep := range deps {
		line += renderDepTree(dep, forward, visited, childPrefix, i == len(deps)-1)
	}
	return line
}
