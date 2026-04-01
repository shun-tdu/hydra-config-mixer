package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"hydra-config-mixer/internal/config"
	"hydra-config-mixer/internal/inspector"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SrcDir はYAML生成時にPythonファイルを検索するルートディレクトリ（main.goから上書き可能）
var SrcDir = "models"

// ConfDir はHydraのconfigディレクトリ（main.goから上書き可能）
var ConfDir = "conf"

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
	statePyFileSearch                      // Pythonファイルを選んでYAMLを生成するモード
	statePyClassSelect                     // Pythonファイル内のクラスを選ぶモード
	stateSavePath                          // 生成YAMLの保存先を確認・編集するモード
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

	// Pythonファイル選択用のデータ
	pyFiles          []string
	pyFileCursor     int
	pyClasses        []string
	pyClassCursor    int
	selectedPyModule string

	// YAML生成・保存用のデータ
	generatedYaml string
	pendingTarget string

	// ディレクトリ設定
	confDir string

	// Modelの履歴
	history History
}

func New(files []string, confDir string) Model {
	ti := textinput.New()
	ti.Placeholder = "例: src.models.YourModel"
	ti.CharLimit = 156
	ti.Width = 40

	m := Model{
		files:     files,
		allFiles:  files,
		textInput: ti,
		state:     stateList,
		confDir:   confDir,
	}

	m.history.Save(m, nil, nil) // 初期状態を保存
	return m
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

// toSnakeCase はCamelCaseの文字列をsnake_caseに変換する。
// "TransformerEncoder" → "transformer_encoder"
// "MotionVAE" → "motion_vae"
func toSnakeCase(s string) string {
	runes := []rune(s)
	var result []rune
	for i, r := range runes {
		if i == 0 {
			result = append(result, unicode.ToLower(r))
			continue
		}
		prev := runes[i-1]
		isUpper := unicode.IsUpper(r)
		prevIsLower := unicode.IsLower(prev)
		nextIsLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
		if isUpper && (prevIsLower || nextIsLower) {
			result = append(result, '_')
		}
		result = append(result, unicode.ToLower(r))
	}
	return string(result)
}

// moduleToSavePath はPythonのモジュールパスからHydraのconfig保存先を計算する。
// "models.transformer.TransformerEncoder" → "<confDir>/models/transformer_encoder.yaml"
// "models.blocks.resnet.ResNetLayer"      → "<confDir>/models/blocks/resnet_layer.yaml"
func moduleToSavePath(target string, confDir string) string {
	parts := strings.Split(target, ".")
	if len(parts) < 2 {
		return filepath.Join(confDir, toSnakeCase(target)+".yaml")
	}
	className := toSnakeCase(parts[len(parts)-1])
	// モジュールパス（クラス名とモジュールファイル名を除いた部分）をディレクトリにする
	var dirParts []string
	if len(parts) >= 3 {
		dirParts = parts[:len(parts)-2]
	}
	dir := filepath.Join(append([]string{confDir}, dirParts...)...)
	return filepath.Join(dir, className+".yaml")
}

func (m *Model) updateViewportHeight() {
	baseHeight := m.height - 4 - 2
	if m.warningMsg != "" {
		paneWidth := (m.width - 6) / 2
		innerWidth := paneWidth - 4
		rendered := warningStyle.Width(innerWidth).Render(m.warningMsg)
		warningLines := strings.Count(rendered, "\n")
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
				oldContent, _ := os.ReadFile(m.editFile)
				newVal := m.textInput.Value()
				comment := strings.TrimSpace(strings.ReplaceAll(m.editLineComment, "# 必須項目", ""))
				m.editLines[m.editCursor] = strings.TrimSpace(m.editLinePrefix + " " + newVal + " " + comment)

				newContent := strings.Join(m.editLines, "\n")
				capturedFile := m.editFile
				capturedOld := oldContent
				capturedNew := []byte(newContent)
				m.history.Save(m,
					func() error { return os.WriteFile(capturedFile, capturedOld, 0644) },
					func() error { return os.WriteFile(capturedFile, capturedNew, 0644) },
				)
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

		// --- PythonファイルからYAMLを生成するモードの処理 ---
		if m.state == statePyFileSearch {
			switch msg.String() {
			case "esc":
				m.state = stateList
				m.textInput.Blur()
				return m, nil
			case "enter":
				if len(m.filteredFiles) > 0 {
					selected := m.filteredFiles[m.pyFileCursor]
					filePath := strings.ReplaceAll(selected, ".", string(filepath.Separator)) + ".py"
					classes := config.LoadPyClasses(filePath)

					switch len(classes) {
					case 0:
						// クラスが見つからなければモジュールパスのまま入力モードへ
						m.state = stateInput
						m.textInput.SetValue(selected)
						m.textInput.Placeholder = "クラスパスを確認・編集してEnterで生成"
						m.textInput.Focus()
					case 1:
						// クラスが1つならそのまま補完して入力モードへ
						m.state = stateInput
						m.textInput.SetValue(selected + "." + classes[0])
						m.textInput.Placeholder = "クラスパスを確認・編集してEnterで生成"
						m.textInput.Focus()
					default:
						// クラスが複数ならクラス選択モードへ
						m.selectedPyModule = selected
						m.pyClasses = classes
						m.pyClassCursor = 0
						m.state = statePyClassSelect
					}
				}
				return m, nil
			case "up", "k":
				if m.pyFileCursor > 0 {
					m.pyFileCursor--
				}
				return m, nil
			case "down", "j":
				if m.pyFileCursor < len(m.filteredFiles)-1 {
					m.pyFileCursor++
				}
				return m, nil
			}

			oldVal := m.textInput.Value()
			m.textInput, cmd = m.textInput.Update(msg)
			if m.textInput.Value() != oldVal {
				query := strings.ToLower(m.textInput.Value())
				m.filteredFiles = []string{}
				for _, f := range m.pyFiles {
					if strings.Contains(strings.ToLower(f), query) {
						m.filteredFiles = append(m.filteredFiles, f)
					}
				}
				m.pyFileCursor = 0
			}
			return m, cmd
		}

		// --- Pythonクラス選択モードの処理 ---
		if m.state == statePyClassSelect {
			switch msg.String() {
			case "esc":
				m.state = statePyFileSearch
				return m, nil
			case "enter":
				if len(m.pyClasses) > 0 {
					fullPath := m.selectedPyModule + "." + m.pyClasses[m.pyClassCursor]
					m.state = stateInput
					m.textInput.SetValue(fullPath)
					m.textInput.Placeholder = "クラスパスを確認・編集してEnterで生成"
					m.textInput.Focus()
				}
				return m, nil
			case "up", "k":
				if m.pyClassCursor > 0 {
					m.pyClassCursor--
				}
				return m, nil
			case "down", "j":
				if m.pyClassCursor < len(m.pyClasses)-1 {
					m.pyClassCursor++
				}
				return m, nil
			}
			return m, nil
		}

		// --- 入力モードの処理 ---
		if m.state == stateInput {
			switch msg.Type {
			case tea.KeyEnter:
				target := m.textInput.Value()
				if target != "" {
					yamlContent, err := inspector.GenerateYamlFromTarget(target)
					if err != nil {
						m.errMsg = err.Error()
						m.state = stateList
						m.textInput.Blur()
					} else {
						// 生成成功 → 保存先確認モードへ
						m.generatedYaml = yamlContent
						m.pendingTarget = target
						m.state = stateSavePath
						m.textInput.SetValue(moduleToSavePath(target, m.confDir))
						m.textInput.Placeholder = "保存先のパス"
					}
				} else {
					m.state = stateList
					m.textInput.Blur()
				}
				return m, nil
			case tea.KeyEsc:
				m.state = stateList
				m.textInput.Blur()
				return m, nil
			}
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}

		// --- 保存先確認・編集モードの処理 ---
		if m.state == stateSavePath {
			switch msg.Type {
			case tea.KeyEsc:
				m.state = stateInput
				m.textInput.SetValue(m.pendingTarget)
				m.textInput.Placeholder = "クラスパスを確認・編集してEnterで生成"
				return m, nil
			case tea.KeyEnter:
				savePath := m.textInput.Value()
				if savePath != "" {
					oldContent, readErr := os.ReadFile(savePath)
					capturedPath := savePath
					capturedNew := []byte(m.generatedYaml)
					var undoAction func() error
					if readErr == nil {
						capturedOld := oldContent
						undoAction = func() error { return os.WriteFile(capturedPath, capturedOld, 0644) }
					} else {
						undoAction = func() error { return os.Remove(capturedPath) }
					}
					m.history.Save(m,
						undoAction,
						func() error { return os.WriteFile(capturedPath, capturedNew, 0644) },
					)
					os.MkdirAll(filepath.Dir(savePath), 0755)
					os.WriteFile(savePath, []byte(m.generatedYaml), 0644)
					m.files, _ = config.LoadYamlFiles(m.confDir)
					m.allFiles = m.files
					m.errMsg = "保存しました: " + savePath
				}
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
					destFile := filepath.Join(m.confDir, filepath.FromSlash(newPath))

					// 新しいディレクトリが含まれていれば自動で作成
					err := os.MkdirAll(filepath.Dir(destFile), 0755)
					if err != nil {
						m.errMsg = "ディレクトリ作成エラー: " + err.Error()
					} else {
						// ファイルのコピー処理
						inputBytes, err := os.ReadFile(sourceFile)
						if err == nil {
							capturedDest := destFile
							capturedContent := inputBytes
							err = os.WriteFile(destFile, inputBytes, 0644)
							if err == nil {
								m.history.Save(m,
									func() error { return os.Remove(capturedDest) },
									func() error { return os.WriteFile(capturedDest, capturedContent, 0644) },
								)
								m.errMsg = "✨ 複製しました: " + destFile
								m.files, _ = config.LoadYamlFiles(m.confDir)
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
				backup, _ := os.ReadFile(targetFile)
				capturedFile := targetFile
				capturedBackup := backup
				m.history.Save(m,
					func() error { return os.WriteFile(capturedFile, capturedBackup, 0644) },
					func() error { return os.Remove(capturedFile) },
				)
				if err := os.Remove(targetFile); err != nil {
					m.errMsg = "削除エラー: " + err.Error()
				} else {
					m.errMsg = "🗑 削除しました: " + targetFile
					m.files, _ = config.LoadYamlFiles(m.confDir)
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
		case "u":
			if snap, ok, err := m.history.Undo(); ok {
				if err != nil {
					m.errMsg = "Undoエラー: " + err.Error()
				} else {
					m.files, _ = config.LoadYamlFiles(m.confDir)
					m.allFiles = m.files
					m.cursor = snap.cursor
					if m.cursor >= len(m.files) {
						m.cursor = max(0, len(m.files)-1)
					}
					cmd := m.updateViewportContent()
					cmds = append(cmds, cmd)
				}
			}
		case "r":
			if snap, ok, err := m.history.Redo(); ok {
				if err != nil {
					m.errMsg = "Redoエラー: " + err.Error()
				} else {
					m.files, _ = config.LoadYamlFiles(m.confDir)
					m.allFiles = m.files
					m.cursor = snap.cursor
					if m.cursor >= len(m.files) {
						m.cursor = max(0, len(m.files)-1)
					}
					cmd := m.updateViewportContent()
					cmds = append(cmds, cmd)
				}
			}
		case "n":
			m.state = statePyFileSearch
			m.textInput.SetValue("")
			m.textInput.Placeholder = "検索: 例) models.vae"
			m.textInput.Focus()
			m.pyFiles = config.LoadPyModules(SrcDir)
			m.filteredFiles = m.pyFiles
			m.pyFileCursor = 0
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
		previewTitle := titleStyle.Render("📄 Preview: " + filepath.Base(selectedFile))

		// 警告があればタイトルとYamlの間に赤文字で挿入
		warningBlock := ""
		if m.warningMsg != "" {
			warningBlock = warningStyle.Render(m.warningMsg)
		}

		rightPaneContent = previewTitle + warningBlock + m.viewport.View()
	}

	// paneHeight を超えないように行数をクランプする（上端が見切れるのを防ぐ）
	clamp := func(s string, max int) string {
		lines := strings.Split(s, "\n")
		if len(lines) > max {
			lines = lines[:max]
		}
		return strings.Join(lines, "\n")
	}

	leftPane := paneStyle.Width(paneWidth).Height(paneHeight).Render(clamp(listStr, paneHeight))
	rightPane := paneStyle.Width(paneWidth).Height(paneHeight).Render(clamp(rightPaneContent, paneHeight))

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
}
