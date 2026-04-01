package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"hydra-config-mixer/internal/config"
	"hydra-config-mixer/internal/inspector"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// アプリケーションの状態管理
type sessionState int

const (
	stateList          sessionState = iota // 通常のリスト選択モード
	stateInput                             // ターゲット入力モード
	stateAutoComplete                      // コンフィグ検索、補完モード
	stateEditLineList                      // ファイル内の行を選ぶモード
	stateEditLineValue                     // 選んだ行の値を書き換えるモード
	stateClone                             // コンフィグを複製するモード
	stateSearchFiles                       // ファイルの絞り込み検索モード
	stateDeleteConfirm                     // 削除確認モード
)

// --- モデルの定義 ---
type Model struct {
	files              []string // 絞り込みこまれているリスト
	allFiles           []string // 全てのファイルのマスターリスト
	cursor             int
	width              int
	height             int
	viewport           viewport.Model
	textInput          textinput.Model
	state              sessionState
	ready              bool
	errMsg             string
	filteredFiles      []string
	autoCompleteCursor int
	warningMsg         string

	// インライン編集用のデータ
	editLines       []string
	editCursor      int
	editFile        string
	editLinePrefix  string
	editLineComment string
}

func New(files []string) Model {
	ti := textinput.New()
	ti.Placeholder = "例: src.models.YourModel"
	ti.CharLimit = 156
	ti.Width = 40

	return Model{
		files:     files,
		allFiles:  files,
		textInput: ti,
		state:     stateList,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m *Model) updateViewportContent() tea.Cmd {
	if len(m.files) == 0 {
		return nil
	}

	m.warningMsg = ""
	selectedFile := m.files[m.cursor]
	contentBytes, err := os.ReadFile(selectedFile)
	content := ""
	if err != nil {
		content = "エラー: " + err.Error()
	} else {
		content = config.HighlightYAML(string(contentBytes))
	}
	m.viewport.SetContent(content)
	m.viewport.GotoTop()

	return inspector.CheckTargetCmd(selectedFile)
}

func (m *Model) updateViewportHeight() {
	baseHeight := m.height - 4 - 2
	if m.warningMsg != "" {
		paneWidth := (m.width - 6) / 2
		innerWidth := paneWidth - 4
		rendered := warningStyle.Width(innerWidth).Render(m.warningMsg)
		warningLines := strings.Count(rendered, "\n") + 1
		m.viewport.Height = baseHeight - warningLines
	} else {
		m.viewport.Height = baseHeight
	}
}

func (m *Model) updateFilteredFiles() {
	query := strings.ToLower(m.textInput.Value())
	m.filteredFiles = []string{}

	for _, f := range m.files {
		cleanPath := strings.TrimPrefix(f, "conf\\")
		cleanPath = strings.TrimPrefix(cleanPath, "conf/")
		cleanPath = strings.TrimSuffix(cleanPath, ".yaml")
		cleanPath = strings.ReplaceAll(cleanPath, "\\", "/")

		if strings.Contains(strings.ToLower(cleanPath), query) {
			m.filteredFiles = append(m.filteredFiles, cleanPath)
		}
	}

	// 絞り込みの結果、カーソルが範囲外に行かないようにリセット
	if m.autoCompleteCursor >= len(m.filteredFiles) {
		m.autoCompleteCursor = 0
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// --- 編集する行を上下キーで選ぶモードの処理 ---
		if m.state == stateEditLineList {
			switch msg.String() {
			case "esc", "q":
				m.state = stateList
				m.updateFilteredFiles()
				return m, nil
			case "up", "k":
				if m.editCursor > 0 {
					m.editCursor--
				}
				return m, nil
			case "down", "j":
				if m.editCursor < len(m.editLines)-1 {
					m.editCursor++
				}
				return m, nil
			case "enter":
				// 選んだ行を":"で左右に分割
				line := m.editLines[m.editCursor]
				parts := strings.SplitN(line, ":", 2)

				if len(parts) == 2 {
					// 編集モードに入る前処理
					m.editLinePrefix = parts[0] + ":"
					rawValue := strings.TrimSpace(parts[1])

					// コメントが含まれているか確認し、値と分離する
					comment := ""
					valParts := strings.SplitN(rawValue, "#", 2)
					if len(valParts) == 2 {
						rawValue = strings.TrimSpace(valParts[0])
						comment = "#" + valParts[1]
					}

					m.textInput.SetValue(rawValue)
					m.textInput.Placeholder = "新しい値を入力..."
					m.textInput.Focus()
					m.state = stateEditLineValue
					m.editLineComment = comment
					m.errMsg = ""
				} else {
					m.errMsg = "この行は編集できません（'キー: 値' の形式ではありません）"
				}
				return m, nil
			}
			return m, nil
		}

		// --- 値をテキストボックスで書き換えて保存するモードの処理 ---
		if m.state == stateEditLineValue {
			switch msg.Type {
			case tea.KeyEsc:
				m.state = stateEditLineList
				m.textInput.Blur()
				return m, nil
			case tea.KeyEnter:
				newVal := m.textInput.Value()
				comment := strings.TrimSpace(strings.ReplaceAll(m.editLineComment, "# 必須項目", ""))
				m.editLines[m.editCursor] = strings.TrimSpace(m.editLinePrefix + " " + newVal + " " + comment)

				newContent := strings.Join(m.editLines, "\n")
				err := os.WriteFile(m.editFile, []byte(newContent), 0644)

				if err != nil {
					m.errMsg = "保存エラー: " + err.Error()
				} else {
					m.errMsg = "✨ 値を更新しました"
				}

				m.state = stateEditLineList
				m.textInput.Blur()
				return m, nil
			}

			// 文字入力の反映
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}

		// --- オートコンプリートモードの処理 ---
		if m.state == stateAutoComplete {
			switch msg.String() {
			case "esc":
				m.state = stateList
				m.textInput.Blur()
				return m, nil
			case "enter":
				if len(m.filteredFiles) > 0 {
					selected := m.filteredFiles[m.autoCompleteCursor]
					targetFile := m.files[m.cursor]

					err := config.EmbedConfigToYaml(targetFile, selected)
					if err != nil {
						m.errMsg = "書き込みエラー: " + err.Error()
					} else {
						m.errMsg = fmt.Sprintf("✨ %s に %s を追加しました", filepath.Base(targetFile), selected)
						cmd := m.updateViewportContent()
						cmds = append(cmds, cmd)
					}
				}
				m.state = stateList
				m.textInput.Blur()
				return m, nil
			case "up", "ctrl+k":
				if m.autoCompleteCursor > 0 {
					m.autoCompleteCursor--
				}
				return m, nil
			case "down", "ctrl+j":
				if m.autoCompleteCursor < len(m.filteredFiles)-1 {
					m.autoCompleteCursor++
				}
				return m, nil
			}

			oldVal := m.textInput.Value()
			m.textInput, cmd = m.textInput.Update(msg)

			if m.textInput.Value() != oldVal {
				m.updateFilteredFiles()
			}
			return m, cmd
		}

		// --- 入力モードの処理 ---
		if m.state == stateInput {
			switch msg.Type {
			case tea.KeyEnter:
				// 入力されたターゲットクラス名を取得
				target := m.textInput.Value()
				if target != "" {
					yamlContent, err := inspector.GenerateYamlFromTarget(target)
					if err != nil {
						m.errMsg = err.Error()
					} else {
						// 成功したらファイルを保存してリストを再読み込み
						savePath := filepath.Join("conf", "model", "generated.yaml")
						os.WriteFile(savePath, []byte(yamlContent), 0644)

						// ファイルをリストを最新の状態に更新、マスターリストにも同期
						m.files, _ = config.LoadYamlFiles("conf")
						m.allFiles = m.files
						m.errMsg = "保存しました: " + savePath
					}
				}
				m.state = stateList
				m.textInput.Blur()
				return m, nil
			case tea.KeyEsc:
				m.state = stateList
				m.textInput.Blur()
				return m, nil
			}
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}

		// --- 複製モードの処理
		if m.state == stateClone {
			switch msg.String() {
			case "esc":
				m.state = stateList
				m.textInput.Blur()
				return m, nil
			case "enter":
				newPath := m.textInput.Value()
				if newPath != "" {
					sourceFile := m.files[m.cursor]
					// 保存先のパスを組み立てる
					destFile := filepath.Join("conf", filepath.FromSlash(newPath))

					// 新しいディレクトリが含まれていれば自動で作成
					err := os.MkdirAll(filepath.Dir(destFile), 0755)
					if err != nil {
						m.errMsg = "ディレクトリ作成エラー: " + err.Error()
					} else {
						// ファイルのコピー処理
						inputBytes, err := os.ReadFile(sourceFile)
						if err == nil {
							err = os.WriteFile(destFile, inputBytes, 0644)
							if err == nil {
								m.errMsg = "✨ 複製しました: " + destFile
								m.files, _ = config.LoadYamlFiles("conf")
								m.allFiles = m.files
							} else {
								m.errMsg = "書き込みエラー: " + err.Error()
							}
						} else {
							m.errMsg = "読み込みエラー: " + err.Error()
						}
					}
				}
				m.state = stateList
				m.textInput.Blur()
				return m, nil
			}

			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}

		// --- 削除確認モードの処理 ---
		if m.state == stateDeleteConfirm {
			switch msg.String() {
			case "y":
				targetFile := m.files[m.cursor]
				if err := os.Remove(targetFile); err != nil {
					m.errMsg = "削除エラー: " + err.Error()
				} else {
					m.errMsg = "🗑 削除しました: " + targetFile
					m.files, _ = config.LoadYamlFiles("conf")
					m.allFiles = m.files
					if m.cursor >= len(m.files) {
						m.cursor = max(0, len(m.files)-1)
					}
					cmd := m.updateViewportContent()
					cmds = append(cmds, cmd)
				}
				m.state = stateList
				return m, tea.Batch(cmds...)
			case "n", "esc":
				m.state = stateList
				return m, nil
			}
			return m, nil
		}

		// --- 検索(Fuzzy Finder)モードの処理 ---
		if m.state == stateSearchFiles {
			switch msg.String() {
			case "esc":
				// 検索をキャンセルして全ファイル表示に戻す
				m.state = stateList
				m.files = m.allFiles
				m.cursor = 0
				m.textInput.Blur()
				cmd := m.updateViewportContent()
				cmds = append(cmds, cmd)
				return m, nil
			case "enter":
				// 現在の絞り込み結果を確定して通常モードに戻る
				m.state = stateList
				m.textInput.Blur()
				return m, nil
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
					cmd := m.updateViewportContent()
					cmds = append(cmds, cmd)
				}
				return m, nil
			case "down", "j":
				if m.cursor < len(m.files)-1 {
					m.cursor++
					cmd := m.updateViewportContent()
					cmds = append(cmds, cmd)
				}
				return m, nil
			}

			oldVal := m.textInput.Value()
			m.textInput, cmd = m.textInput.Update(msg)

			// 文字が入力、削除されて値が変わったらリアルタイムで絞り込みを実行
			if m.textInput.Value() != oldVal {
				query := strings.ToLower(m.textInput.Value())
				var filtered []string

				for _, f := range m.allFiles {
					if strings.Contains(strings.ToLower(f), query) {
						filtered = append(filtered, f)
					}
				}
				m.files = filtered
				m.cursor = 0
				cmd := m.updateViewportContent()
				cmds = append(cmds, cmd)
			}
			return m, cmd
		}

		// --- リスト選択モードの処理 ---
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "n": // 'n'キーで新規作成（入力モード）へ移行
			m.state = stateInput
			m.textInput.SetValue("src.models.vae.MotionVAE") // テスト用初期値
			m.textInput.Focus()
			m.errMsg = ""
			return m, textinput.Blink
		case "a":
			m.state = stateAutoComplete
			m.textInput.SetValue("")
			m.textInput.Placeholder = "検索: 例) models/vae"
			m.textInput.Focus()
			m.updateFilteredFiles() // 最初に全件表示
			m.autoCompleteCursor = 0
			m.errMsg = ""
			return m, textinput.Blink
		case "e":
			if len(m.files) > 0 {
				selectedFile := m.files[m.cursor]
				contentBytes, err := os.ReadFile(selectedFile)
				if err == nil {
					// ファイルを行ごとに分割してメモリに保持
					cleanContent := strings.ReplaceAll(string(contentBytes), "\r\n", "\n")
					m.editLines = strings.Split(cleanContent, "\n")
					m.editCursor = 0
					m.editFile = selectedFile
					m.state = stateEditLineList
					m.errMsg = ""
				} else {
					m.errMsg = "読み込みエラー: " + err.Error()
				}
			}
			return m, nil
		case "c":
			if len(m.files) > 0 {
				m.state = stateClone
				selectedFile := m.files[m.cursor]

				// プレフィックスを消して表示用のパスを作成
				cleanPath := strings.TrimPrefix(selectedFile, "conf\\")
				cleanPath = strings.TrimPrefix(cleanPath, "conf/")
				cleanPath = strings.ReplaceAll(cleanPath, "\\", "/")

				// デフォルトの提案として"_copy"を付けた名前をセット
				suggested := strings.TrimSuffix(cleanPath, ".yaml") + "_copy.yaml"

				m.textInput.SetValue(suggested)
				m.textInput.Placeholder = "新しいファイルパス (例: model/vae_large.yaml)"
				m.textInput.Focus()
				m.errMsg = ""
				return m, textinput.Blink
			}
		case "d":
			if len(m.files) > 0 {
				m.state = stateDeleteConfirm
				m.errMsg = ""
			}
			return m, nil
		case "/":
			m.state = stateSearchFiles
			m.textInput.SetValue("")
			m.textInput.Placeholder = "ファイル検索 (例: vae)"
			m.textInput.Focus()
			m.errMsg = ""
			return m, textinput.Blink

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				cmd := m.updateViewportContent()
				cmds = append(cmds, cmd)
			}
		case "down", "j":
			if m.cursor < len(m.files)-1 {
				m.cursor++
				cmd := m.updateViewportContent()
				cmds = append(cmds, cmd)
			}
		}

	case inspector.TargetCheckMsg:
		if !msg.Exists {
			m.warningMsg = "⚠ _target_: " + msg.Target + " が見つかりません"
		} else {
			m.warningMsg = ""
		}
		m.updateViewportHeight()

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		paneWidth := (m.width - 6) / 2

		if !m.ready {
			m.viewport = viewport.New(paneWidth, m.height-4-2)
			cmd := m.updateViewportContent()
			cmds = append(cmds, cmd)
			m.ready = true
		} else {
			m.viewport.Width = paneWidth
		}
		m.updateViewportHeight()
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

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

	// --- 入力モードのUIを追加表示 ---
	if m.state == stateInput {
		listStr += "\n" + titleStyle.Render("✨ Generate Config") + "\n"
		listStr += "Target Class Path:\n"
		listStr += m.textInput.View() + "\n\n"
		listStr += lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("(Enterで生成 / Escでキャンセル)")
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
	leftPane := paneStyle.Width(paneWidth).Height(paneHeight).Render(listStr)

	// 右ペイン：プレビュー
	var rightPaneContent string

	if m.state == stateEditLineList || m.state == stateEditLineValue {
		var sb strings.Builder
		sb.WriteString(titleStyle.Render("✏️ Inline Edit: "+filepath.Base(m.editFile)) + "\n\n")

		for i, line := range m.editLines {
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
		sb.WriteString("\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("(↑/↓: 行選択, Enter: 編集/保存, Esc: 終了)"))
		rightPaneContent = sb.String()
	} else {
		var selectedFile string
		if len(m.files) > 0 {
			selectedFile = m.files[m.cursor]
		}
		previewTitle := titleStyle.Render("📄 Preview: "+filepath.Base(selectedFile)) + "\n"

		// 警告があればタイトルとYamlの間に赤文字で挿入
		warningBlock := ""
		if m.warningMsg != "" {
			warningBlock = warningStyle.Render(m.warningMsg) + "\n"
		}

		rightPaneContent = previewTitle + warningBlock + m.viewport.View()
	}

	rightPane := paneStyle.Width(paneWidth).Height(paneHeight).Render(rightPaneContent)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
}
