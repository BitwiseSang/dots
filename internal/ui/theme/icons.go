package theme

import (
	"path/filepath"
	"strings"
)

// Nerd Font icons for application actions and items
const (
	// Navigation & Actions
	IconBackup      = "󰁯"
	IconSetup       = ""
	IconEdit        = ""
	IconBrowse      = ""
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

	// File & Folder types
	IconDir      = ""
	IconFile     = ""
	IconLua      = ""
	IconTmux     = ""
	IconFish     = "󰈺"
	IconDoom     = ""
	IconCpp      = ""
	IconKitty    = "󰄛"
	IconGhostty  = "󰞷"
	IconToml     = ""
	IconMarkdown = ""
	IconGit      = ""
	IconShell    = ""
	IconAria2    = ""
)

// FileIcon returns the appropriate Nerd Font icon based on filename and directory status.
func FileIcon(name string, isDir bool) string {
	if isDir {
		return IconDir + " "
	}

	lower := strings.ToLower(name)
	base := filepath.Base(lower)
	ext := filepath.Ext(lower)

	switch {
	case strings.Contains(base, "ghostty"):
		return IconGhostty + " "
	case strings.Contains(base, "kitty"):
		return IconKitty + " "
	case strings.Contains(base, "tmux"):
		return IconTmux + " "
	case strings.Contains(base, "clang-format") || ext == ".cpp" || ext == ".c" || ext == ".h" || ext == ".hpp":
		return IconCpp + " "
	case strings.Contains(base, "aria2"):
		return IconAria2 + " "
	case ext == ".lua" || strings.Contains(base, "nvim"):
		return IconLua + " "
	case ext == ".el" || strings.Contains(base, "doom"):
		return IconDoom + " "
	case ext == ".fish" || strings.Contains(base, "fish"):
		return IconFish + " "
	case ext == ".sh" || ext == ".bash" || ext == ".zsh":
		return IconShell + " "
	case ext == ".toml" || ext == ".yaml" || ext == ".yml" || ext == ".json":
		return IconToml + " "
	case ext == ".md":
		return IconMarkdown + " "
	case strings.HasPrefix(base, ".git"):
		return IconGit + " "
	default:
		return IconFile + " "
	}
}
