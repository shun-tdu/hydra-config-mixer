# HydraConfigMixer

A terminal UI for managing [Hydra](https://hydra.cc/) configuration files — browse, edit, generate, and organize your ML experiment configs without leaving the terminal.

![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go)
![Platform](https://img.shields.io/badge/platform-Linux%20%7C%20macOS%20%7C%20Windows-lightgrey?style=flat-square)
![License](https://img.shields.io/badge/license-MIT-green?style=flat-square)

[日本語版はこちら](README.ja.md)

---

## Features

- **Browse & preview** — navigate your `conf/` directory with syntax-highlighted YAML
- **Inline editing** — edit individual values directly in the TUI
- **Generate from Python** — pick a class from your source tree and generate a Hydra config from its `__init__` signature automatically
- **Clone & delete** — duplicate or remove configs with confirmation prompts
- **File search** — fuzzy-filter configs and Python source files
- **Undo / Redo** — 32-step history for all file operations
- **Target validation** — warns when `_target_` points to a class that doesn't exist in your Python environment

---

## Demo

```
┌─ Files ─────────────────────┐ ┌─ Preview ──────────────────────────────┐
│ conf/config.yaml            │ │ _target_: src.models.vae.MotionVAE     │
│ conf/model/transformer.yaml │ │ latent_dim: 256                        │
│ conf/optimizer/adamw.yaml   │ │ hidden_dim: 512                        │
│ > conf/model/vae.yaml       │ │ num_layers: 4                          │
│                             │ │ dropout: 0.1                           │
└─────────────────────────────┘ └────────────────────────────────────────┘
[e] edit  [n] new from Python  [d] delete  [c] clone  [/] search  [u/r] undo/redo
```

---

## Requirements

- **Go 1.21+** (for building)
- **Python 3** with your project's dependencies installed (same environment Hydra runs in)

---

## Installation

### Build from source

```bash
git clone https://github.com/yourname/hydra-config-mixer
cd hydra-config-mixer
go build -o hydra-config-mixer .
```

### Install with `go install`

```bash
go install github.com/yourname/hydra-config-mixer@latest
```

---

## Usage

Run from your Hydra project root:

```bash
hydra-config-mixer
```

Custom directories:

```bash
hydra-config-mixer --conf config --src src/models
```

| Flag | Default | Description |
|------|---------|-------------|
| `--conf` | `conf` | Hydra config directory |
| `--src` | `models` | Python source directory for class discovery |

---

## Key Bindings

| Key | Action |
|-----|--------|
| `↑` / `↓` | Move cursor |
| `Enter` | Select / confirm |
| `e` | Edit selected config (inline) |
| `n` | Generate new config from Python class |
| `c` | Clone selected config |
| `d` | Delete selected config |
| `/` | Search configs |
| `a` | Autocomplete `_target_` |
| `u` | Undo |
| `r` | Redo |
| `Esc` | Cancel / go back |
| `q` | Quit |

---

## Generating Configs from Python Classes

Press `n` to open the Python file browser. HydraConfigMixer will:

1. Scan your `--src` directory for `.py` files
2. Let you search and select a file
3. Show all classes defined in that file
4. Read the `__init__` signature and generate a YAML with correct types and defaults

```python
# src/models/vae.py
class MotionVAE:
    def __init__(self, latent_dim: int, hidden_dim: int = 512, dropout: float = 0.1):
        ...
```

Generates:

```yaml
_target_: src.models.vae.MotionVAE
latent_dim: ???  # 必須項目 # type: int
hidden_dim: 512  # type: int
dropout: 0.1     # type: float
```

The save path is derived from the module path automatically (`src.models.vae.MotionVAE` → `conf/models/motion_vae.yaml`) and can be edited before saving.

---

## Project Structure

```
hydra-config-mixer/
├── main.go
├── internal/
│   ├── config/       # YAML file loading and manipulation
│   ├── inspector/    # Python class introspection (go:embed)
│   └── ui/           # Bubble Tea TUI (model, view, update, styles, history)
└── conf/             # Example Hydra configs
```

---

## Built With

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) — TUI framework
- [Bubbles](https://github.com/charmbracelet/bubbles) — TUI components
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) — Terminal styling
- [Chroma](https://github.com/alecthomas/chroma) — Syntax highlighting

---

## License

MIT
