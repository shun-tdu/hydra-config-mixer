package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/chroma/v2/quick"
)

// 指定したディレクトリ以下の .yaml ファイルを再帰的に検索する関数
func LoadYamlFiles(rootDir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// ディレクトリではなく、拡張子が .yaml のものだけを追加
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".yaml") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// YAMLの文字列を受け取り、色付きの文字に変換する
func HighlightYAML(content string) string {
	buf := new(bytes.Buffer)

	err := quick.Highlight(buf, content, "yaml", "terminal256", "dracula")

	if err != nil {
		return content
	}
	return buf.String()
}

func EmbedConfigToYaml(targetFile string, embedPath string) error {
	// 🌟 ここを embedPath に修正！
	parts := strings.Split(embedPath, "/")

	var group, name, searchPrefix, newLine string
	if len(parts) >= 2 {
		group = parts[0]
		name = parts[len(parts)-1]
		searchPrefix = "- " + group + ":"
		newLine = fmt.Sprintf("  - %s: %s", group, name)
	} else {
		name = parts[0]
		searchPrefix = "- " + name
		newLine = fmt.Sprintf("  - %s", name)
	}

	contentBytes, err := os.ReadFile(targetFile)
	if err != nil {
		return err
	}
	lines := strings.Split(string(contentBytes), "\n")

	inDefaults := false
	replaced := false
	defaultsIdx := -1

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// defaultsブロックの開始を検知
		if strings.HasPrefix(trimmed, "defaults:") {
			inDefaults = true
			defaultsIdx = i
			continue
		}

		// defaultsブロック内を走査して上書き対象を探す
		if inDefaults {
			// インデントがなくなったらdefaultsブロック終了とみなす
			if len(trimmed) > 0 && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "-") {
				inDefaults = false
			} else if group != "" && strings.HasPrefix(trimmed, searchPrefix) {
				lines[i] = newLine
				replaced = true
				break
			}
		}
	}

	// 既存のグループが見つからない場合は同じブロック内に追加
	if defaultsIdx != -1 && !replaced {
		// Goのsliceトリック（安全な挿入）
		var newLines []string
		newLines = append(newLines, lines[:defaultsIdx+1]...)
		newLines = append(newLines, newLine)
		newLines = append(newLines, lines[defaultsIdx+1:]...)
		lines = newLines
	} else if defaultsIdx == -1 {
		// defaults: ブロックが存在しない場合は先頭に追加
		header := []string{"defaults:", newLine}
		lines = append(header, lines...)
	}

	newContent := strings.Join(lines, "\n")
	return os.WriteFile(targetFile, []byte(newContent), 0644)
}
