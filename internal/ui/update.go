package ui

import (
	"fmt"
	"hydra-config-mixer/internal/config"
	"hydra-config-mixer/internal/inspector"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case stateEditLineList:
			// --- 編集する行を上下キーで選ぶモードの処理 ---
			return updateStateEditLineList(m, msg)
		case stateEditLineValue:
			// --- 値をテキストボックスで書き換えて保存するモードの処理 ---
			return updateStateEditLineValue(m, msg)
		case stateAutoComplete:
			// --- オートコンプリートモードの処理 ---
			return updateStateAutoComplete(m, msg)
		case statePyFileSearch:
			// --- PythonファイルからYAMLを生成するモードの処理 ---
			return updateStatePyFileSearch(m, msg)
		case statePyClassSelect:
			// --- Pythonクラス選択モードの処理 ---
			return updateStatePyClassSelect(m, msg)
		case stateInput:
			// --- 入力モードの処理 ---
			return updateStateInput(m, msg)
		case stateSavePath:
			// --- 保存先確認・編集モードの処理 ---
			return updateStateSavePath(m, msg)
		case stateClone:
			// --- 複製モードの処理
			return updateStateClone(m, msg)
		case stateDeleteConfirm:
			// --- 削除確認モードの処理 ---
			return updateStateDeleteConfirm(m, msg)
		case stateSearchFiles:
			// --- 検索(Fuzzy Finder)モードの処理 ---
			return updateStateSearchFiles(m, msg)
		case stateList:
			// --- リスト選択モードの処理 ---
			return updateStateList(m, msg)
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

func updateStateEditLineList(m Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.state = stateList
		m.updateFilteredFiles()
		cmd := m.updateViewportContent()
		return m, cmd
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
	case "a":
		// 編集モードからもオートコンプリートで defaults に追加
		m.autoCompleteReturnState = stateEditLineList
		m.state = stateAutoComplete
		m.textInput.SetValue("")
		m.textInput.Placeholder = "検索: 例) models/vae"
		m.textInput.Focus()
		m.updateFilteredFiles()
		m.autoCompleteCursor = 0
		m.errMsg = ""
		return m, textinput.Blink
	case "K":
		// defaults エントリを上に移動
		i := m.editCursor
		if isInDefaultsBlock(m.editLines, i) {
			for prev := i - 1; prev >= 0; prev-- {
				if isInDefaultsBlock(m.editLines, prev) {
					oldContent, _ := os.ReadFile(m.editFile)
					m.editLines[prev], m.editLines[i] = m.editLines[i], m.editLines[prev]
					m.editCursor = prev
					newContent := strings.Join(m.editLines, "\n")
					capturedFile, capturedOld, capturedNew := m.editFile, oldContent, []byte(newContent)
					m.history.Save(m,
						func() error { return os.WriteFile(capturedFile, capturedOld, 0644) },
						func() error { return os.WriteFile(capturedFile, capturedNew, 0644) },
					)
					os.WriteFile(m.editFile, []byte(newContent), 0644)
					break
				}
			}
		}
		return m, nil
	case "J":
		// defaults エントリを下に移動
		i := m.editCursor
		if isInDefaultsBlock(m.editLines, i) {
			for next := i + 1; next < len(m.editLines); next++ {
				if isInDefaultsBlock(m.editLines, next) {
					oldContent, _ := os.ReadFile(m.editFile)
					m.editLines[next], m.editLines[i] = m.editLines[i], m.editLines[next]
					m.editCursor = next
					newContent := strings.Join(m.editLines, "\n")
					capturedFile, capturedOld, capturedNew := m.editFile, oldContent, []byte(newContent)
					m.history.Save(m,
						func() error { return os.WriteFile(capturedFile, capturedOld, 0644) },
						func() error { return os.WriteFile(capturedFile, capturedNew, 0644) },
					)
					os.WriteFile(m.editFile, []byte(newContent), 0644)
					break
				}
			}
		}
		return m, nil
	case "x":
		// defaults エントリを削除
		i := m.editCursor
		if isInDefaultsBlock(m.editLines, i) {
			oldContent, _ := os.ReadFile(m.editFile)
			m.editLines = append(m.editLines[:i], m.editLines[i+1:]...)
			if m.editCursor >= len(m.editLines) {
				m.editCursor = max(0, len(m.editLines)-1)
			}
			newContent := strings.Join(m.editLines, "\n")
			capturedFile, capturedOld, capturedNew := m.editFile, oldContent, []byte(newContent)
			m.history.Save(m,
				func() error { return os.WriteFile(capturedFile, capturedOld, 0644) },
				func() error { return os.WriteFile(capturedFile, capturedNew, 0644) },
			)
			os.WriteFile(m.editFile, []byte(newContent), 0644)
			m.reloadFiles()
		}
		return m, nil
	case "enter":
		// defaults エントリは値編集不可
		if isInDefaultsBlock(m.editLines, m.editCursor) {
			m.errMsg = "defaults エントリは K/J で並び替え、x で削除できます"
			return m, nil
		}
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

func updateStateEditLineValue(m Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func updateStateClone(m Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
						m.reloadFiles()
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

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func updateStateSearchFiles(m Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg.String() {
	case "esc":
		// 検索をキャンセルして全ファイル表示に戻す
		m.state = stateList
		m.files = m.allFiles
		m.cursor = 0
		m.textInput.Blur()
		cmds = append(cmds, m.updateViewportContent())
		return m, tea.Batch(cmds...)
	case "enter":
		// 現在の絞り込み結果を確定して通常モードに戻る
		m.state = stateList
		m.textInput.Blur()
		return m, nil
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			cmds = append(cmds, m.updateViewportContent())
		}
		return m, tea.Batch(cmds...)
	case "down", "j":
		if m.cursor < len(m.files)-1 {
			m.cursor++
			cmds = append(cmds, m.updateViewportContent())
		}
		return m, tea.Batch(cmds...)
	}

	var cmd tea.Cmd
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
		cmds = append(cmds, m.updateViewportContent())
	}
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func updateStateDeleteConfirm(m Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
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
			m.reloadFiles()
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

func updateStatePyFileSearch(m Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

	var cmd tea.Cmd
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

func updateStateAutoComplete(m Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg.String() {
	case "esc":
		m.state = m.autoCompleteReturnState
		m.autoCompleteReturnState = stateList
		m.textInput.Blur()
		return m, nil
	case "enter":
		if len(m.filteredFiles) > 0 {
			selected := m.filteredFiles[m.autoCompleteCursor]
			targetFile := m.files[m.cursor]
			if m.autoCompleteReturnState == stateEditLineList {
				targetFile = m.editFile
			}

			err := config.EmbedConfigToYaml(targetFile, selected)
			if err != nil {
				m.errMsg = "書き込みエラー: " + err.Error()
			} else {
				m.errMsg = fmt.Sprintf("✨ %s に %s を追加しました", filepath.Base(targetFile), selected)
				cmd := m.updateViewportContent()
				cmds = append(cmds, cmd)
				// 編集モードから来た場合は editLines を再読み込み
				if m.autoCompleteReturnState == stateEditLineList {
					contentBytes, _ := os.ReadFile(m.editFile)
					clean := strings.ReplaceAll(string(contentBytes), "\r\n", "\n")
					m.editLines = strings.Split(clean, "\n")
				}
			}
		}
		m.state = m.autoCompleteReturnState
		m.autoCompleteReturnState = stateList
		m.textInput.Blur()
		return m, tea.Batch(cmds...)
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

	var cmd tea.Cmd
	oldVal := m.textInput.Value()
	m.textInput, cmd = m.textInput.Update(msg)

	if m.textInput.Value() != oldVal {
		m.updateFilteredFiles()
	}
	return m, cmd
}

func updateStateSavePath(m Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
			m.reloadFiles()
			m.errMsg = "保存しました: " + savePath
		}
		m.state = stateList
		m.textInput.Blur()
		return m, nil
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func updateStatePyClassSelect(m Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

func updateStateList(m Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "u":
		if snap, ok, err := m.history.Undo(); ok {
			if err != nil {
				m.errMsg = "Undoエラー: " + err.Error()
			} else {
				m.reloadFiles()
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
				m.reloadFiles()
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
		m.autoCompleteReturnState = stateList
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

	return m, tea.Batch(cmds...)
}

func updateStateInput(m Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}
