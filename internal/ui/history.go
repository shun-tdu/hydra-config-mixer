package ui

const historySize = 32

// Snapshot はundo/redoで復元する状態を保持する
type Snapshot struct {
	files      []string
	allFiles   []string
	cursor     int
	undoAction func() error // この操作を取り消すファイル操作
	redoAction func() error // この操作をやり直すファイル操作
}

// History はリングバッファで操作履歴を管理する
type History struct {
	snapshots [historySize]Snapshot
	current   int
	undoSize  int
	redoSize  int
}

// Save は現在の状態とundo/redoのファイル操作を保存する
func (h *History) Save(m Model, undoAction, redoAction func() error) {
	next := (h.current + 1) % historySize
	h.snapshots[next] = Snapshot{
		files:      append([]string{}, m.files...),
		allFiles:   append([]string{}, m.allFiles...),
		cursor:     m.cursor,
		undoAction: undoAction,
		redoAction: redoAction,
	}
	h.current = next
	if h.undoSize < historySize {
		h.undoSize++
	}
	h.redoSize = 0
}

// Undo は現在のsnapshotのundoActionを実行し、1つ前の状態を返す
func (h *History) Undo() (Snapshot, bool, error) {
	if h.undoSize <= 1 {
		return Snapshot{}, false, nil
	}
	snap := h.snapshots[h.current]
	if snap.undoAction != nil {
		if err := snap.undoAction(); err != nil {
			return Snapshot{}, false, err
		}
	}
	h.current = (h.current - 1 + historySize) % historySize
	h.undoSize--
	h.redoSize++
	return h.snapshots[h.current], true, nil
}

// Redo は1つ先のsnapshotのredoActionを実行し、その状態を返す
func (h *History) Redo() (Snapshot, bool, error) {
	if h.redoSize == 0 {
		return Snapshot{}, false, nil
	}
	h.current = (h.current + 1) % historySize
	snap := h.snapshots[h.current]
	if snap.redoAction != nil {
		if err := snap.redoAction(); err != nil {
			h.current = (h.current - 1 + historySize) % historySize
			return Snapshot{}, false, err
		}
	}
	h.redoSize--
	h.undoSize++
	return snap, true, nil
}
