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

// standardHomeDotfiles lists common root-level dotfiles in $HOME to inspect during discovery.
var standardHomeDotfiles = []string{
	".bashrc",
	".bash_profile",
	".bash_aliases",
	".zshrc",
	".zshenv",
	".zprofile",
	".tmux.conf",
	".gitconfig",
	".vimrc",
	".profile",
	".xinitrc",
	".xprofile",
	".inputrc",
	".nanorc",
	".clang-format",
	".editorconfig",
}

// UserConfigDir resolves the active user configuration directory, honoring $XDG_CONFIG_HOME
// and falling back to ~/.config.
func UserConfigDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return config.ExpandPath(xdg)
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".config")
	}
	return ""
}

func cleanConfigName(name string) string {
	clean := strings.TrimPrefix(name, ".")
	ext := filepath.Ext(clean)
	if ext != "" {
		clean = strings.TrimSuffix(clean, ext)
	}
	if clean == "" {
		return name
	}
	return clean
}

// DiscoverSystemConfigs dynamically scans the system for existing configuration files
// and directories across $XDG_CONFIG_HOME (default ~/.config) and $HOME, comparing them
// with currently managed dotfiles in cfg.
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

	home, _ := os.UserHomeDir()

	// 1. Scan $XDG_CONFIG_HOME (or ~/.config) for user configurations
	configDir := UserConfigDir()
	if configDir != "" {
		if entries, err := os.ReadDir(configDir); err == nil {
			for _, e := range entries {
				abs := filepath.Join(configDir, e.Name())
				if seenPaths[abs] {
					continue
				}
				// Skip hidden entries or system caches
				if strings.HasPrefix(e.Name(), ".") || e.Name() == "pulse" || e.Name() == "dconf" {
					continue
				}

				seenPaths[abs] = true
				method := "copy"
				if e.IsDir() {
					method = "rsync"
				}

				name := cleanConfigName(e.Name())
				sysPath := abs
				if home != "" && strings.HasPrefix(abs, home+string(filepath.Separator)) {
					sysPath = filepath.Join("~", strings.TrimPrefix(abs, home+string(filepath.Separator)))
				}

				isManaged := managedMap[strings.ToLower(name)] || managedMap[strings.ToLower(abs)]
				discovered = append(discovered, DiscoveredConfig{
					Name:           name,
					SystemPath:     sysPath,
					RepoPath:       e.Name(),
					IsDir:          e.IsDir(),
					Method:         method,
					AlreadyManaged: isManaged,
				})
			}
		}
	}

	// 2. Scan $HOME for standard root dotfiles
	if home != "" {
		for _, file := range standardHomeDotfiles {
			abs := filepath.Join(home, file)
			if seenPaths[abs] {
				continue
			}
			if fi, err := os.Lstat(abs); err == nil {
				seenPaths[abs] = true
				name := cleanConfigName(file)
				sysPath := filepath.Join("~", file)
				isManaged := managedMap[strings.ToLower(name)] || managedMap[strings.ToLower(abs)]
				discovered = append(discovered, DiscoveredConfig{
					Name:           name,
					SystemPath:     sysPath,
					RepoPath:       file,
					IsDir:          fi.IsDir(),
					Method:         "copy",
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
