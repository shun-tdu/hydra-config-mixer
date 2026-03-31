// Pythonクラスの構造解析とYAMLファイルの生成を担う

package inspector

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// Pythonの関数/クラスの引数情報を表す
type Param struct {
	Name       string `json:"name"`
	HasDefault bool   `json:"has_default"`
	Default    any    `json:"default"`
}

// inspect_helper.pyが返すJSONの構造体
type InspectResult struct {
	Target string  `json:"target"`
	Params []Param `json:"params"`
	Error  string  `json:"error"`
}

func GenerateYamlFromTarget(target string) (string, error) {
	cmd := exec.Command("python", "inspect_helper.py", target)
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
		if p.HasDefault {
			if p.Default == nil {
				sb.WriteString(fmt.Sprintf("%s: null\n", p.Name))
			} else {
				sb.WriteString(fmt.Sprintf("%s: %v\n", p.Name, p.Default))
			}
		} else {
			sb.WriteString(fmt.Sprintf("%s: ??? # 必須項目\n", p.Name))
		}
	}
	return sb.String(), nil
}
