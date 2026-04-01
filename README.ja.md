# HydraConfigMixer

[Hydra](https://hydra.cc/) の設定ファイルをターミナル上で管理するための TUI ツールです。ML 実験のコンフィグをターミナルから離れることなく、閲覧・編集・生成・整理できます。

![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go)
![Platform](https://img.shields.io/badge/platform-Linux%20%7C%20macOS%20%7C%20Windows-lightgrey?style=flat-square)
![License](https://img.shields.io/badge/license-MIT-green?style=flat-square)

---

## 機能

- **ブラウズ & プレビュー** — `conf/` 以下の YAML をシンタックスハイライト付きで閲覧
- **インライン編集** — TUI 上で各値を直接書き換え
- **Python クラスから自動生成** — `__init__` シグネチャを解析して Hydra コンフィグを自動生成
- **複製 & 削除** — 確認プロンプト付きでコンフィグをコピー・削除
- **ファイル検索** — コンフィグと Python ファイルをインクリメンタル検索で絞り込み
- **アンドゥ / リドゥ** — すべてのファイル操作を 32 ステップ分保持
- **ターゲット検証** — `_target_` が Python 環境に存在しないクラスを指していると警告

---

## デモ

```
┌─ Files ─────────────────────┐ ┌─ Preview ──────────────────────────────┐
│ conf/config.yaml            │ │ _target_: src.models.vae.MotionVAE     │
│ conf/model/transformer.yaml │ │ latent_dim: 256                        │
│ conf/optimizer/adamw.yaml   │ │ hidden_dim: 512                        │
│ > conf/model/vae.yaml       │ │ num_layers: 4                          │
│                             │ │ dropout: 0.1                           │
└─────────────────────────────┘ └────────────────────────────────────────┘
[e] 編集  [n] Pythonから生成  [d] 削除  [c] 複製  [/] 検索  [u/r] undo/redo
```

---

## 動作要件

- **Go 1.21+**（ビルド時のみ）
- **Python 3**（Hydra を動かしているのと同じ仮想環境）

---

## インストール

### ソースからビルド

```bash
git clone https://github.com/yourname/hydra-config-mixer
cd hydra-config-mixer
go build -o hydra-config-mixer .
```

### `go install` でインストール

```bash
go install github.com/yourname/hydra-config-mixer@latest
```

---

## 使い方

Hydra プロジェクトのルートディレクトリで実行してください。

```bash
hydra-config-mixer
```

ディレクトリを指定する場合：

```bash
hydra-config-mixer --conf config --src src/models
```

| フラグ | デフォルト | 説明 |
|--------|-----------|------|
| `--conf` | `conf` | Hydra コンフィグディレクトリ |
| `--src` | `models` | クラス検索対象の Python ソースディレクトリ |

---

## キーバインド

| キー | 操作 |
|------|------|
| `↑` / `↓` | カーソル移動 |
| `Enter` | 選択 / 確定 |
| `e` | 選択中のコンフィグをインライン編集 |
| `n` | Python クラスから新規コンフィグを生成 |
| `c` | 選択中のコンフィグを複製 |
| `d` | 選択中のコンフィグを削除 |
| `/` | コンフィグを検索 |
| `a` | `_target_` をオートコンプリート |
| `u` | アンドゥ |
| `r` | リドゥ |
| `Esc` | キャンセル / 前の画面に戻る |
| `q` | 終了 |

---

## Python クラスからコンフィグを生成する

`n` を押すと Python ファイルブラウザが開きます。以下の手順で進みます：

1. `--src` ディレクトリの `.py` ファイルを一覧表示
2. ファイルを検索・選択
3. そのファイルに定義されているクラスを一覧表示
4. `__init__` シグネチャを解析し、型とデフォルト値を持つ YAML を自動生成

```python
# src/models/vae.py
class MotionVAE:
    def __init__(self, latent_dim: int, hidden_dim: int = 512, dropout: float = 0.1):
        ...
```

生成される YAML：

```yaml
_target_: src.models.vae.MotionVAE
latent_dim: ???  # 必須項目 # type: int
hidden_dim: 512  # type: int
dropout: 0.1     # type: float
```

保存先はモジュールパスから自動で導出されます（`src.models.vae.MotionVAE` → `conf/models/motion_vae.yaml`）。保存前に編集することも可能です。

---

## プロジェクト構成

```
hydra-config-mixer/
├── main.go
├── internal/
│   ├── config/       # YAML ファイルの読み込みと操作
│   ├── inspector/    # Python クラスのイントロスペクション（go:embed）
│   └── ui/           # Bubble Tea TUI（model / view / update / styles / history）
└── conf/             # Hydra コンフィグのサンプル
```

---

## 使用ライブラリ

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) — TUI フレームワーク
- [Bubbles](https://github.com/charmbracelet/bubbles) — TUI コンポーネント
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) — ターミナルスタイリング
- [Chroma](https://github.com/alecthomas/chroma) — シンタックスハイライト

---

## ライセンス

MIT
