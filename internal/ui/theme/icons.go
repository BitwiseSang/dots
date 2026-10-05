package theme

import (
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Nerd Font icons for application actions and items
const (
	// Navigation & Actions
	IconBackup      = "󰁯"
	IconSetup       = ""
	IconEdit        = ""
	IconBrowse      = ""
	IconRemove      = ""
	IconQuit        = ""
	IconCursor      = "❯"
	IconDotsCluster = ":::"
	IconDot         = "•"
	IconBullet      = "·"

	// Status indicators
	IconInSync  = ""
	IconChanged = ""
	IconMissing = ""
	IconLinked  = ""

	// Folder types (eza-style)
	IconDirModern = "" // eza modern directory
	IconDirConfig = "" // config directory
	IconDirGit    = "" // git directory
	IconDirDoc    = "󰈙" // docs folder
	IconDirBin    = "" // bin/scripts folder

	// Application & File types
	IconDir      = ""
	IconFile     = ""
	IconLua      = ""
	IconNvim     = ""
	IconTmux     = ""
	IconFish     = "󰈺"
	IconDoom     = ""
	IconCpp      = ""
	IconC        = ""
	IconKitty    = "󰄛"
	IconGhostty  = "󰞷"
	IconToml     = ""
	IconYaml     = ""
	IconJson     = ""
	IconMarkdown = ""
	IconGit      = ""
	IconShell    = ""
	IconAria2    = ""
	IconGo       = ""
	IconRust     = ""
	IconPython   = ""
	IconMakefile = ""
	IconExe      = "󰡯"
	IconLock     = ""
	IconText     = ""
)

// FileIcon returns the appropriate Nerd Font icon based on filename and directory status.
func FileIcon(name string, isDir bool) string {
	icon, _ := FileIconStyled(name, isDir)
	return icon + " "
}

// FileIconStyled returns the icon and eza-style theme color for a file or directory.
func FileIconStyled(name string, isDir bool) (string, lipgloss.Color) {
	lower := strings.ToLower(name)
	base := filepath.Base(lower)
	ext := filepath.Ext(lower)

	if isDir {
		switch {
		case base == ".config" || base == "config" || base == "configs":
			return IconDirConfig, lipgloss.Color("#06B6D4") // Cyan
		case strings.HasPrefix(base, ".git"):
			return IconDirGit, lipgloss.Color("#F97316") // Orange
		case base == "nvim":
			return IconNvim, lipgloss.Color("#10B981") // Emerald Neovim
		case base == "fish":
			return IconFish, lipgloss.Color("#06B6D4") // Cyan Fish
		case base == "ghostty":
			return IconGhostty, lipgloss.Color("#EC4899") // Pink
		case base == "kitty":
			return IconKitty, lipgloss.Color("#F59E0B") // Amber
		case base == "tmux":
			return IconTmux, lipgloss.Color("#10B981") // Green
		case base == "doom" || base == "emacs":
			return IconDoom, lipgloss.Color("#A855F7") // Violet
		case base == "doc" || base == "docs":
			return IconDirDoc, lipgloss.Color("#60A5FA") // Blue
		case base == "bin" || base == "scripts":
			return IconDirBin, lipgloss.Color("#F43F5E") // Rose
		default:
			return IconDirModern, lipgloss.Color("#38BDF8") // Sky Blue modern folder
		}
	}

	// File lookups
	switch {
	// Exact/Prefix file matches
	case base == "makefile" || base == "justfile":
		return IconMakefile, lipgloss.Color("#F59E0B")
	case base == "dockerfile" || strings.HasPrefix(base, "docker-compose"):
		return "󰡨", lipgloss.Color("#0284C7")
	case base == "cargo.toml" || base == "cargo.lock":
		return IconRust, lipgloss.Color("#EA580C")
	case base == "go.mod" || base == "go.sum":
		return IconGo, lipgloss.Color("#00ADD8")
	case base == "readme.md" || base == "license":
		return IconMarkdown, lipgloss.Color("#EC4899")
	case strings.Contains(base, "ghostty"):
		return IconGhostty, lipgloss.Color("#EC4899")
	case strings.Contains(base, "kitty"):
		return IconKitty, lipgloss.Color("#F59E0B")
	case strings.Contains(base, "tmux"):
		return IconTmux, lipgloss.Color("#10B981")
	case strings.Contains(base, "aria2"):
		return IconAria2, lipgloss.Color("#06B6D4")
	case strings.Contains(base, "clang-format") || ext == ".cpp" || ext == ".hpp" || ext == ".cc":
		return IconCpp, lipgloss.Color("#3B82F6")
	case ext == ".c" || ext == ".h":
		return IconC, lipgloss.Color("#60A5FA")
	case ext == ".lua" || strings.Contains(base, "nvim"):
		return IconLua, lipgloss.Color("#60A5FA")
	case ext == ".fish" || strings.Contains(base, "fish"):
		return IconFish, lipgloss.Color("#06B6D4")
	case ext == ".el" || strings.Contains(base, "doom"):
		return IconDoom, lipgloss.Color("#A855F7")
	case ext == ".go":
		return IconGo, lipgloss.Color("#00ADD8")
	case ext == ".rs":
		return IconRust, lipgloss.Color("#EA580C")
	case ext == ".py":
		return IconPython, lipgloss.Color("#FACC15")
	case ext == ".sh" || ext == ".bash" || ext == ".zsh":
		return IconShell, lipgloss.Color("#10B981")
	case ext == ".json" || ext == ".jsonc":
		return IconJson, lipgloss.Color("#FBBF24")
	case ext == ".toml":
		return IconToml, lipgloss.Color("#9CA3AF")
	case ext == ".yaml" || ext == ".yml":
		return IconYaml, lipgloss.Color("#F43F5E")
	case ext == ".conf" || ext == ".ini":
		return IconToml, lipgloss.Color("#A855F7")
	case ext == ".md" || ext == ".markdown":
		return IconMarkdown, lipgloss.Color("#F472B6")
	case ext == ".txt":
		return IconText, lipgloss.Color("#9CA3AF")
	case ext == ".lock":
		return IconLock, lipgloss.Color("#EF4444")
	case strings.HasPrefix(base, ".git"):
		return IconGit, lipgloss.Color("#F97316")
	default:
		return IconFile, lipgloss.Color("#D1D5DB")
	}
}
