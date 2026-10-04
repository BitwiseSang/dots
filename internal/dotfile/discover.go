package dotfile

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BitwiseSang/dots/internal/config"
)

type DiscoveredConfig struct {
	Name           string
	SystemPath     string
	RepoPath       string
	IsDir          bool
	Method         string
	AlreadyManaged bool
}

// CommonCandidate definitions for auto-discovery
type candidate struct {
	name   string
	path   string
	isDir  bool
	method string
}

var commonCandidates = []candidate{
	{name: "nvim", path: "~/.config/nvim", isDir: true, method: "rsync"},
	{name: "fish", path: "~/.config/fish", isDir: true, method: "rsync"},
	{name: "kitty", path: "~/.config/kitty", isDir: true, method: "rsync"},
	{name: "ghostty", path: "~/.config/ghostty", isDir: true, method: "rsync"},
	{name: "alacritty", path: "~/.config/alacritty", isDir: true, method: "rsync"},
	{name: "tmux", path: "~/.tmux.conf", isDir: false, method: "copy"},
	{name: "starship", path: "~/.config/starship.toml", isDir: false, method: "copy"},
	{name: "git", path: "~/.gitconfig", isDir: false, method: "copy"},
	{name: "zsh", path: "~/.zshrc", isDir: false, method: "copy"},
	{name: "bash", path: "~/.bashrc", isDir: false, method: "copy"},
	{name: "hypr", path: "~/.config/hypr", isDir: true, method: "rsync"},
	{name: "waybar", path: "~/.config/waybar", isDir: true, method: "rsync"},
	{name: "rofi", path: "~/.config/rofi", isDir: true, method: "rsync"},
	{name: "dunst", path: "~/.config/dunst", isDir: true, method: "rsync"},
	{name: "btop", path: "~/.config/btop", isDir: true, method: "rsync"},
	{name: "fastfetch", path: "~/.config/fastfetch", isDir: true, method: "rsync"},
	{name: "helix", path: "~/.config/helix", isDir: true, method: "rsync"},
	{name: "doom", path: "~/.config/doom", isDir: true, method: "rsync"},
}

// DiscoverSystemConfigs scans the system for existing configuration files and directories,
// comparing them with the currently managed dotfiles in cfg.
func DiscoverSystemConfigs(cfg *config.Config) []DiscoveredConfig {
	managedMap := make(map[string]bool)
	if cfg != nil {
		for _, d := range cfg.Dotfiles {
			managedMap[strings.ToLower(d.Name)] = true
			managedMap[strings.ToLower(config.ExpandPath(d.SystemPath))] = true
		}
	}

	seenPaths := make(map[string]bool)
	var discovered []DiscoveredConfig

	// 1. Check known candidates
	for _, c := range commonCandidates {
		abs := config.ExpandPath(c.path)
		if fi, err := os.Lstat(abs); err == nil {
			seenPaths[abs] = true
			isManaged := managedMap[strings.ToLower(c.name)] || managedMap[strings.ToLower(abs)]
			discovered = append(discovered, DiscoveredConfig{
				Name:           c.name,
				SystemPath:     c.path,
				RepoPath:       c.name,
				IsDir:          fi.IsDir(),
				Method:         c.method,
				AlreadyManaged: isManaged,
			})
		}
	}

	// 2. Scan ~/.config for other top-level directories
	home, err := os.UserHomeDir()
	if err == nil {
		configDir := filepath.Join(home, ".config")
		if entries, err := os.ReadDir(configDir); err == nil {
			for _, e := range entries {
				abs := filepath.Join(configDir, e.Name())
				if seenPaths[abs] {
					continue
				}
				// Skip hidden or system caches
				if strings.HasPrefix(e.Name(), ".") || e.Name() == "pulse" || e.Name() == "dconf" {
					continue
				}

				method := "copy"
				if e.IsDir() {
					method = "rsync"
				}

				isManaged := managedMap[strings.ToLower(e.Name())] || managedMap[strings.ToLower(abs)]
				discovered = append(discovered, DiscoveredConfig{
					Name:           e.Name(),
					SystemPath:     filepath.Join("~/.config", e.Name()),
					RepoPath:       e.Name(),
					IsDir:          e.IsDir(),
					Method:         method,
					AlreadyManaged: isManaged,
				})
			}
		}
	}

	// Sort unmanaged first, then alphabetical
	sort.Slice(discovered, func(i, j int) bool {
		if discovered[i].AlreadyManaged != discovered[j].AlreadyManaged {
			return !discovered[i].AlreadyManaged
		}
		return strings.ToLower(discovered[i].Name) < strings.ToLower(discovered[j].Name)
	})

	return discovered
}
