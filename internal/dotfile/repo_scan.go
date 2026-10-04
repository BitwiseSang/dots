package dotfile

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BitwiseSang/dots/internal/config"
)

// InspectRepository checks a repository path for dotfile configurations.
// 1. If dots/config.toml, .config/dots/config.toml, or config.toml exists in the repo,
//    it loads that configuration and returns its dotfiles.
// 2. Otherwise, it inspects top-level directories and files inside the repository,
//    inferring system targets.
func InspectRepository(repoPath string) ([]config.DotfileSpec, *config.Config, error) {
	absPath := config.ExpandPath(repoPath)

	// Case 1: Check for existing config.toml in repo
	if cfgFile := config.FindRepoConfigFile(absPath); cfgFile != "" {
		if cfg, err := config.LoadFromPath(cfgFile); err == nil && len(cfg.Dotfiles) > 0 {
			cfg.RepoPath = repoPath
			return cfg.Dotfiles, cfg, nil
		}
	}

	// Case 2: Scan repository directory
	entries, err := os.ReadDir(absPath)
	if err != nil {
		return nil, nil, err
	}

	var specs []config.DotfileSpec
	for _, e := range entries {
		name := e.Name()

		// Skip hidden files/dirs (like .git, .github) and common non-config files
		if strings.HasPrefix(name, ".") || strings.EqualFold(name, "license") ||
			strings.HasPrefix(strings.ToLower(name), "readme") ||
			strings.EqualFold(name, "makefile") ||
			strings.EqualFold(name, "dots") {
			continue
		}

		info, err := e.Info()
		if err != nil {
			continue
		}

		isDir := info.IsDir()
		method := "copy"
		systemPath := filepath.Join("~/.config", name)

		if isDir {
			method = "rsync"
			systemPath = filepath.Join("~/.config", name)
		} else {
			// Special handling for common dotfiles at repo root
			switch strings.ToLower(name) {
			case "tmux.conf", ".tmux.conf":
				systemPath = "~/.tmux.conf"
			case "zshrc", ".zshrc":
				systemPath = "~/.zshrc"
			case "bashrc", ".bashrc":
				systemPath = "~/.bashrc"
			case "gitconfig", ".gitconfig":
				systemPath = "~/.gitconfig"
			}
		}

		cleanName := strings.TrimPrefix(name, ".")
		cleanName = strings.TrimSuffix(cleanName, filepath.Ext(cleanName))
		if cleanName == "" {
			cleanName = name
		}

		specs = append(specs, config.DotfileSpec{
			Name:       cleanName,
			RepoPath:   name,
			SystemPath: systemPath,
			Method:     method,
			IsDir:      isDir,
		})
	}

	sort.Slice(specs, func(i, j int) bool {
		return strings.ToLower(specs[i].Name) < strings.ToLower(specs[j].Name)
	})

	cfg := config.DefaultConfig()
	cfg.RepoPath = repoPath
	cfg.Dotfiles = specs

	return specs, cfg, nil
}
