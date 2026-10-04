# dots ╺━━━━━━━╸

A beautiful, modern terminal user interface (TUI) and CLI tool for managing your dotfiles. Built with [Charmbracelet](https://charm.sh) tools: [Bubble Tea](https://github.com/charmbracelet/bubbletea), [Bubbles](https://github.com/charmbracelet/bubbles), and [Lip Gloss](https://github.com/charmbracelet/lipgloss).

---

## ✨ Features

- 🎨 **Beautiful Terminal UI**: Clean typography, elegant borders, responsive layouts, and modern color palette.
- 📦 **Selective Backups**: Choose exactly which configurations to back up from your system into the repository.
- 🔍 **Diff Previews**: Inspect colored, unified diffs before confirming any backup or setup operation.
- 🔗 **Selective Setup**: Safely symlink configs from your repo to your system, automatically archiving existing configs into timestamped backups.
- ✏️ **Frictionless Editing**:
  - **Direct CLI**: Jump straight into editing any config: `dots edit nvim`, `dots edit tmux`, etc. No need to `cd` or remember long paths.
  - **Interactive TUI**: Browse all configs in a menu and open them with a keystroke; editor suspends the TUI and resumes seamlessly upon exit.
- 🌿 **Git Integration**: Interactive prompt to commit with timestamped metadata and push after backups.
- ⚙️ **Configurable**: Fully customizable via `~/.config/dots/config.toml` (editor, repository path, custom dotfile entries).

---

## 🚀 Installation

### Using Make

```bash
cd dots
make install
```

This compiles the binary and copies it to `~/.local/bin/dots`. Ensure `~/.local/bin` is in your `$PATH`.

---

## 📖 Usage

### Interactive TUI

Launch the full interactive interface:

```bash
dots
```

Jump directly to a specific view within the TUI:

```bash
dots backup   # Open backup view
dots setup    # Open setup view
dots edit     # Open edit menu
```

### Direct Editing (Bypass TUI)

Open any config directly in your editor without launching the TUI:

```bash
dots edit nvim          # Opens ~/.config/nvim in $EDITOR
dots edit tmux          # Opens ~/.tmux.conf in $EDITOR
dots edit ghostty       # Opens Ghostty config in $EDITOR
dots edit fish          # Opens Fish config in $EDITOR
```

If you specify an unknown config name, `dots` prints the list of available dotfiles.

### Command-Line Flags

```bash
dots [command] [flags]

Flags:
  -e, --editor string   Editor command to use (overrides config and $EDITOR)
  -r, --repo string     Path to dotfiles repository (overrides config)
  -v, --version         Print version
  -h, --help            Show help
```

---

## ⌨️ Keybindings

### Global
- `q` / `Ctrl+C`: Quit
- `Esc`: Go back to previous menu / view

### Selection Views (Backup / Setup)
- `↑` / `k`: Move up
- `↓` / `j`: Move down
- `Space`: Toggle selection of current item
- `a`: Select / deselect all items
- `Enter`: Proceed to preview / execute

### Diff Preview
- `↑` / `k`: Scroll up
- `↓` / `j`: Scroll down
- `Enter`: Confirm and execute
- `Esc`: Cancel and return to selection

---

## ⚙️ Configuration

Optional configuration file located at `~/.config/dots/config.toml`:

```toml
# Editor command (falls back to $EDITOR, then "nvim")
editor = "nvim"

# Path to your dotfiles repository
repo_path = "~/Documents/dotfiles"

[git]
auto_commit = false     # Automatically commit on backup without prompt
auto_push = false       # Automatically push after commit
commit_prefix = ""      # Optional commit message prefix

# Optional: Add custom dotfile entries
# [[dotfiles]]
# name = "hyprland"
# repo_path = "hyprland"
# system_path = "~/.config/hypr"
# method = "rsync"       # "copy" or "rsync"
# is_dir = true
```

---

## 🏗️ Architecture

```
dots/
├── cmd/
│   └── dots/
│       └── main.go          # Cobra CLI & TUI entrypoint
├── internal/
│   ├── config/              # TOML config loader & default registry
│   ├── dotfile/             # Entry models, backup, setup, diff engine
│   ├── editor/              # Editor launcher and process manager
│   ├── git/                 # Git status, commit, and push routines
│   └── ui/
│       ├── app.go           # Root Bubble Tea model & router
│       ├── theme/           # Lip Gloss palette & styles
│       ├── components/      # Reusable header, status bar, selector
│       └── views/           # Home, Backup, Setup, and Edit views
├── Makefile
└── go.mod
```
