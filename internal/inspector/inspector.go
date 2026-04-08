// Pythonクラスの構造解析とYAMLファイルの生成を担う

package inspector

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// PythonPath はPythonインタープリターのパス
var Python = "python"

//go:embed inspect_helper.py
var inspectHelperScript []byte

// Pythonの関数/クラスの引数情報を表す
type Param struct {
	Name       string `json:"name"`
	HasDefault bool   `json:"has_default"`
	Default    any    `json:"default"`
	Type       string `json:"type"`
}

// inspect_helper.pyが返すJSONの構造体
type InspectResult struct {
	Target string  `json:"target"`
	Params []Param `json:"params"`
	Error  string  `json:"error"`
}

// Pythonのクラス実装チェックの結果を受け取る構造体
type TargetCheckMsg struct {
	Target string
	Exists bool
}

// 指定されたPathの.pyを読み込み、クラスのイニシャライザからyamlを生成する
func GenerateYamlFromTarget(target string) (string, error) {
	tmp, err := os.CreateTemp("", "inspect_helper_*.py")
	if err != nil {
		return "", fmt.Errorf("一時ファイルの作成に失敗: %v", err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(inspectHelperScript); err != nil {
		tmp.Close()
		return "", fmt.Errorf("一時ファイルへの書き込みに失敗: %v", err)
	}
	tmp.Close()

	// PythonPathの存在確認
	pythonPath, lookErr := getPythonPath()
	if lookErr != nil {
		return "", fmt.Errorf("Pythonの実行に失敗: %v", lookErr)
	}

	// Pythonコマンドの実行確認
	cmd := exec.Command(pythonPath, tmp.Name(), target)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("Pythonの実行に失敗: %v\n%s", err, string(out))
	}

	var res InspectResult
	if err := json.Unmarshal(out, &res); err != nil {
		return "", fmt.Errorf("JSONのパースに失敗: %v", err)
	}
	if res.Error != "" {
		return "", fmt.Errorf("Pythonエラー: %s", res.Error)
	}

	// YAML文字列を組み立てる
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("_target_: %s\n", res.Target))
	for _, p := range res.Params {
		typeStr := ""
		if p.Type != "" {
			typeStr = fmt.Sprintf(" # type: %s", p.Type)
		}

		if p.HasDefault {
			if p.Default == nil {
				sb.WriteString(fmt.Sprintf("%s: null%s\n", p.Name, typeStr))
			} else {
				sb.WriteString(fmt.Sprintf("%s: %v%s\n", p.Name, p.Default, typeStr))
			}
		} else {
			sb.WriteString(fmt.Sprintf("%s: ??? # 必須項目%s\n", p.Name, typeStr))
		}
	}
	return sb.String(), nil
}

func CheckTargetCmd(filePath string) tea.Cmd {
	return func() tea.Msg {
		content, err := os.ReadFile(filePath)
		if err != nil {
			// エラー時はスキップ
			return TargetCheckMsg{"", true}
		}

		// ファイルから _target_:の行を探す
		lines := strings.Split(string(content), "\n")
		var target string
		for _, line := range lines {
			if strings.Contains(line, "_target_:") {
				parts := strings.Split(line, ":")
				if len(parts) >= 2 {
					target = strings.TrimSpace(parts[1])
					target = strings.Split(target, "#")[0]
					target = strings.TrimSpace(target)
					break
				}
			}
		}

		// ターゲット無し or ???はチェックをスキップ
		if target == "" || target == "???" {
			return TargetCheckMsg{"", true}
		}

		// 🌟 Pythonの importlib を使って「モジュールとクラスが存在するか」だけを爆速で判定するスクリプト
		script := "import sys, importlib\n" +
			"try:\n" +
			"    parts = sys.argv[1].rsplit('.', 1)\n" +
			"    m = importlib.import_module(parts[0])\n" +
			"    sys.exit(0 if hasattr(m, parts[1]) else 1)\n" +
			"except Exception:\n" +
			"    sys.exit(1)\n"

		if pythonPath, lookErr := getPythonPath(); lookErr == nil {
			cmd := exec.Command(pythonPath, "-c", script, target)
			err = cmd.Run()
		} else {
			err = lookErr
		}

		return TargetCheckMsg{
			Target: target,
			Exists: err == nil,
		}
	}
}

// getPythonPathはPythonが存在すれば返す。
func getPythonPath() (string, error) {
	if Python != "" {
		return Python, nil
	}

	// pythonPath未指定の場合は、python3, pythonの順に探索してフォールバック
	candidates := []string{"python3", "python"}
	for _, file := range candidates {
		if path, err := exec.LookPath(file); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("unable to find python path")
}
