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
	stateList         sessionState = iota // 通常のリスト選択モード
	stateInput                            // ターゲット入力モード
	stateAutoComplete                     // コンフィグ検索、補完モード
)

// --- モデルの定義 ---
type Model struct {
	files              []string // 実際のファイルパスのリスト
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
}

func New(files []string) Model {
	ti := textinput.New()
	ti.Placeholder = "例: src.models.YourModel"
	ti.CharLimit = 156
	ti.Width = 40

	return Model{
		files:     files,
		textInput: ti,
		state:     stateList,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m *Model) updateViewportContent() {
	if len(m.files) == 0 {
		return
	}

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
						m.updateViewportContent()
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
						m.files, _ = config.LoadYamlFiles("conf")
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
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.updateViewportContent()
			}
		case "down", "j":
			if m.cursor < len(m.files)-1 {
				m.cursor++
				m.updateViewportContent()
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		paneWidth := (m.width - 6) / 2
		vpHeight := m.height - 4 - 2

		if !m.ready {
			m.viewport = viewport.New(paneWidth, vpHeight)
			m.updateViewportContent()
			m.ready = true
		} else {
			m.viewport.Width = paneWidth
			m.viewport.Height = vpHeight
		}
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

	// 入力モードのUIを追加表示
	if m.state == stateInput {
		listStr += "\n" + titleStyle.Render("✨ Generate Config") + "\n"
		listStr += "Target Class Path:\n"
		listStr += m.textInput.View() + "\n\n"
		listStr += lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("(Enterで生成 / Escでキャンセル)")
	}

	// オートコンプリートUIの描画
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

	if m.errMsg != "" {
		listStr += "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(m.errMsg)
	}
	leftPane := paneStyle.Width(paneWidth).Height(paneHeight).Render(listStr)

	// 右ペイン：プレビュー
	var selectedFile string
	if len(m.files) > 0 {
		selectedFile = m.files[m.cursor]
	}
	previewTitle := titleStyle.Render("📄 Preview: "+filepath.Base(selectedFile)) + "\n"
	rightPane := paneStyle.Width(paneWidth).Height(paneHeight).Render(previewTitle + m.viewport.View())

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
}
