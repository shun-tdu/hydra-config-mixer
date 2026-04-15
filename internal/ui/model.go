package ui

import (
	"hydra-config-mixer/internal/config"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
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
	files                   []string // 絞り込みこまれているリスト
	allFiles                []string // 全てのファイルのマスターリスト
	cursor                  int
	width                   int
	height                  int
	viewport                viewport.Model
	textInput               textinput.Model
	state                   sessionState
	ready                   bool
	errMsg                  string
	filteredFiles           []string
	autoCompleteCursor      int
	autoCompleteReturnState sessionState // オートコンプリート終了後の戻り先
	warningMsg              string

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

	// 依存グラフ
	depForward map[string][]string // ファイル → 依存先
	depReverse map[string][]string // ファイル → 被依存元

	// Modelの履歴
	history History
}

func New(files []string, confDir string) Model {
	ti := textinput.New()
	ti.Placeholder = "例: src.models.YourModel"
	ti.CharLimit = 156
	ti.Width = 40

	fwd, rev := config.BuildDepGraph(files, confDir)
	m := Model{
		files:      files,
		allFiles:   files,
		textInput:  ti,
		state:      stateList,
		confDir:    confDir,
		depForward: fwd,
		depReverse: rev,
	}

	m.history.Save(m, nil, nil) // 初期状態を保存
	return m
}

func (m *Model) reloadFiles() {
	m.files, _ = config.LoadYamlFiles(m.confDir)
	m.allFiles = m.files
	m.depForward, m.depReverse = config.BuildDepGraph(m.files, m.confDir)
}

func (m Model) Init() tea.Cmd {
	return nil
}

// isInDefaultsBlock はその行インデックスが defaults: ブロック内のリストエントリかどうかを返す
func isInDefaultsBlock(lines []string, idx int) bool {
	if idx < 0 || idx >= len(lines) {
		return false
	}
	trimmed := strings.TrimSpace(lines[idx])
	if !strings.HasPrefix(trimmed, "-") {
		return false
	}
	for i := idx - 1; i >= 0; i-- {
		t := strings.TrimSpace(lines[i])
		if strings.HasPrefix(t, "defaults:") {
			return true
		}
		if t == "" {
			continue
		}
		// インデントのない非defaults行に当たったら範囲外
		if !strings.HasPrefix(lines[i], " ") && !strings.HasPrefix(lines[i], "\t") {
			return false
		}
	}
	return false
}
