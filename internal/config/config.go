package config

import (
	"bytes"
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

// LoadPyModules は rootDir 以下の .py ファイルをPythonモジュールパスとして返す。
// __init__.py は除外する。
func LoadPyModules(rootDir string) []string {
	var modules []string
	filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.HasSuffix(d.Name(), ".py") && d.Name() != "__init__.py" {
			module := strings.TrimSuffix(path, ".py")
			module = strings.ReplaceAll(module, string(filepath.Separator), ".")
			module = strings.ReplaceAll(module, "/", ".")
			modules = append(modules, module)
		}
		return nil
	})
	return modules
}

// LoadPyClasses は .py ファイルを読んでクラス名の一覧を返す。
func LoadPyClasses(filePath string) []string {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}
	var classes []string
	for _, line := range strings.Split(string(content), "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "class ") {
			continue
		}
		// "class Foo:" や "class Foo(Bar):" からクラス名を取り出す
		name := strings.TrimPrefix(trimmed, "class ")
		name = strings.FieldsFunc(name, func(r rune) bool {
			return r == ':' || r == '('
		})[0]
		name = strings.TrimSpace(name)
		if name != "" {
			classes = append(classes, name)
		}
	}
	return classes
}

// ParseDefaults はYAMLファイルの defaults: ブロックを解析し、依存するファイルパスのリストを返す
func ParseDefaults(filePath string, confDir string) []string {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}

	lines := strings.Split(string(content), "\n")
	inDefaults := false
	var deps []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "defaults:") {
			inDefaults = true
			continue
		}

		if inDefaults {
			if len(trimmed) > 0 && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") && !strings.HasPrefix(line, "-") {
				break
			}
			if !strings.HasPrefix(trimmed, "-") {
				continue
			}

			entry := strings.TrimSpace(strings.TrimPrefix(trimmed, "-"))
			// "_self_" はHydra組み込みなのでスキップ
			if entry == "_self_" {
				continue
			}

			var depPath string
			if strings.Contains(entry, ":") {
				// "group: name" 形式
				parts := strings.SplitN(entry, ":", 2)
				group := strings.TrimSpace(parts[0])
				name := strings.TrimSpace(parts[1])
				depPath = filepath.Join(confDir, group, name+".yaml")
			} else {
				// "path/name" 形式
				depPath = filepath.Join(confDir, filepath.FromSlash(entry)+".yaml")
			}

			if _, err := os.Stat(depPath); err == nil {
				deps = append(deps, depPath)
			}
		}
	}

	return deps
}

// ReadDefaultsEntries はYAMLファイルの defaults: ブロックのエントリを生文字列スライスで返す
func ReadDefaultsEntries(filePath string) ([]string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(content), "\n")
	inDefaults := false
	var entries []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "defaults:") {
			inDefaults = true
			continue
		}
		if inDefaults {
			if len(trimmed) > 0 && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") && !strings.HasPrefix(line, "-") {
				break
			}
			if strings.HasPrefix(trimmed, "-") {
				entries = append(entries, strings.TrimSpace(strings.TrimPrefix(trimmed, "-")))
			}
		}
	}
	return entries, nil
}

// WriteDefaultsEntries はYAMLファイルの defaults: ブロックを entries で上書きする
func WriteDefaultsEntries(filePath string, entries []string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	lines := strings.Split(string(content), "\n")

	defaultsStart := -1
	defaultsEnd := -1
	indent := "  "

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "defaults:") {
			defaultsStart = i
			continue
		}
		if defaultsStart >= 0 && defaultsEnd < 0 {
			if len(trimmed) > 0 && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") && !strings.HasPrefix(line, "-") {
				defaultsEnd = i
				break
			}
			if strings.HasPrefix(trimmed, "-") {
				indent = line[:len(line)-len(strings.TrimLeft(line, " \t"))]
			}
		}
	}

	if defaultsStart < 0 {
		return nil
	}
	if defaultsEnd < 0 {
		defaultsEnd = len(lines)
	}

	var newLines []string
	newLines = append(newLines, lines[:defaultsStart+1]...)
	for _, entry := range entries {
		newLines = append(newLines, indent+"- "+entry)
	}
	newLines = append(newLines, lines[defaultsEnd:]...)

	return os.WriteFile(filePath, []byte(strings.Join(newLines, "\n")), 0644)
}

// KeySource はあるキーがどのファイルで定義されているかを表す
type KeySource struct {
	File  string
	Value string
}

// CollectKeys はYAMLファイルからトップレベルの key: value を収集する（defaults・_target_ は除外）
func CollectKeys(filePath string) map[string]string {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}
	result := make(map[string]string)
	inDefaults := false

	for _, line := range strings.Split(string(content), "\n") {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "defaults:") {
			inDefaults = true
			continue
		}
		if inDefaults {
			if len(trimmed) > 0 && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") && !strings.HasPrefix(line, "-") {
				inDefaults = false
			} else {
				continue
			}
		}

		// トップレベル（インデントなし）の key: value のみ対象
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(trimmed, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if key == "" || key == "_target_" {
			continue
		}
		result[key] = val
	}
	return result
}

// ConflictInfo は競合しているキーの情報を表す
type ConflictInfo struct {
	Key     string
	Sources []KeySource // マージ順（先頭が最初に読まれる＝優先度低）
}

// FindConflicts は filePath の defaults: を再帰的に展開し、競合しているキーを返す。
// マージ順を保持するため deps は順序付きスライスで渡す。
func FindConflicts(filePath string, forward map[string][]string) []ConflictInfo {
	// defaults を再帰的にフラット化（BFS、マージ順）
	var ordered []string
	visited := map[string]bool{}
	queue := []string{filePath}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if visited[cur] {
			continue
		}
		visited[cur] = true
		if cur != filePath {
			ordered = append(ordered, cur)
		}
		for _, dep := range forward[cur] {
			queue = append(queue, dep)
		}
	}
	// _self_ 相当として対象ファイル自身を末尾に追加
	ordered = append(ordered, filePath)

	// キーごとに定義元を収集
	keyMap := map[string][]KeySource{}
	for _, f := range ordered {
		keys := CollectKeys(f)
		for k, v := range keys {
			keyMap[k] = append(keyMap[k], KeySource{File: f, Value: v})
		}
	}

	// 2つ以上のファイルで定義されているキーを競合として返す
	var conflicts []ConflictInfo
	for key, sources := range keyMap {
		if len(sources) > 1 {
			conflicts = append(conflicts, ConflictInfo{Key: key, Sources: sources})
		}
	}
	return conflicts
}

// BuildDepGraph は全YAMLファイルの依存グラフを構築する。
// 戻り値は forward（ファイル→依存先）と reverse（ファイル→被依存元）の2つのマップ。
func BuildDepGraph(files []string, confDir string) (forward, reverse map[string][]string) {
	forward = make(map[string][]string)
	reverse = make(map[string][]string)
	for _, f := range files {
		forward[f] = nil
	}
	for _, f := range files {
		deps := ParseDefaults(f, confDir)
		forward[f] = deps
		for _, dep := range deps {
			reverse[dep] = append(reverse[dep], f)
		}
	}
	return forward, reverse
}

func EmbedConfigToYaml(targetFile string, embedPath string) error {
	parts := strings.Split(embedPath, "/")

	var group, name string
	if len(parts) >= 2 {
		group = parts[0]
		name = parts[len(parts)-1]
	} else {
		name = parts[0]
	}

	contentBytes, err := os.ReadFile(targetFile)
	if err != nil {
		return err
	}
	lines := strings.Split(string(contentBytes), "\n")

	inDefaults := false
	replaced := false
	defaultsIdx := -1
	detectedIndent := "  " // デフォルトは2スペース

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// defaultsブロックの開始を検知
		if strings.HasPrefix(trimmed, "defaults:") {
			inDefaults = true
			defaultsIdx = i
			continue
		}

		// defaultsブロック内を走査する
		if inDefaults {
			// インデントがなくなったらdefaultsブロック終了とみなす
			if len(trimmed) > 0 && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") && !strings.HasPrefix(line, "-") {
				inDefaults = false
				continue
			}

			if strings.HasPrefix(trimmed, "-") {
				// 既存行からインデントを検出
				indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
				detectedIndent = indent

				// 同じグループの行があれば上書き
				if group != "" && strings.HasPrefix(trimmed, "- "+group+":") {
					lines[i] = indent + "- " + group + ": " + name
					replaced = true
					break
				}
			}
		}
	}

	// 既存のグループが見つからない場合は追加
	if defaultsIdx != -1 && !replaced {
		var newLine string
		if group != "" {
			newLine = detectedIndent + "- " + group + ": " + name
		} else {
			newLine = detectedIndent + "- " + name
		}
		var newLines []string
		newLines = append(newLines, lines[:defaultsIdx+1]...)
		newLines = append(newLines, newLine)
		newLines = append(newLines, lines[defaultsIdx+1:]...)
		lines = newLines
	} else if defaultsIdx == -1 {
		// defaults: ブロックが存在しない場合は先頭に追加
		var newLine string
		if group != "" {
			newLine = "  - " + group + ": " + name
		} else {
			newLine = "  - " + name
		}
		header := []string{"defaults:", newLine}
		lines = append(header, lines...)
	}

	newContent := strings.Join(lines, "\n")
	return os.WriteFile(targetFile, []byte(newContent), 0644)
}
