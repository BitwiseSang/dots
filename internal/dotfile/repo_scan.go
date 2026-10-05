package dotfile

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BitwiseSang/dots/internal/config"
)

func inferSpecFromEntry(name string, isDir bool) config.DotfileSpec {
	method := "copy"
	var systemPath string

	if isDir {
		method = "rsync"
		if strings.HasPrefix(name, ".") {
			// Top-level dot-directories map to user's home (e.g. .ssh -> ~/.ssh, .vim -> ~/.vim)
			systemPath = filepath.Join("~", name)
		} else {
			switch strings.ToLower(name) {
			case "ssh":
				systemPath = "~/.ssh"
			case "gnupg":
				systemPath = "~/.gnupg"
			default:
				// If ~/.<name> exists in home directory and ~/.config/<name> does not,
				// infer ~/.<name> as system target
				home, err := os.UserHomeDir()
				if err == nil {
					homeTarget := filepath.Join(home, "."+name)
					configTarget := filepath.Join(home, ".config", name)
					if _, err := os.Stat(homeTarget); err == nil {
						if _, err := os.Stat(configTarget); os.IsNotExist(err) {
							systemPath = filepath.Join("~", "."+name)
						}
					}
				}
				if systemPath == "" {
					systemPath = filepath.Join("~/.config", name)
				}
			}
		}
	} else {
		if strings.HasPrefix(name, ".") {
			// Top-level files starting with a dot map to user's home directory (e.g. .bashrc -> ~/.bashrc)
			systemPath = filepath.Join("~", name)
		} else {
			// Non-dot top-level files: check common root configs or default to ~/.config/<name>
			switch strings.ToLower(name) {
			case "tmux.conf":
				systemPath = "~/.tmux.conf"
			case "zshrc":
				systemPath = "~/.zshrc"
			case "bashrc":
				systemPath = "~/.bashrc"
			case "bash_profile":
				systemPath = "~/.bash_profile"
			case "bash_aliases":
				systemPath = "~/.bash_aliases"
			case "zprofile":
				systemPath = "~/.zprofile"
			case "zshenv":
				systemPath = "~/.zshenv"
			case "gitconfig":
				systemPath = "~/.gitconfig"
			case "vimrc":
				systemPath = "~/.vimrc"
			case "xinitrc":
				systemPath = "~/.xinitrc"
			case "xprofile":
				systemPath = "~/.xprofile"
			case "profile":
				systemPath = "~/.profile"
			case "inputrc":
				systemPath = "~/.inputrc"
			case "nanorc":
				systemPath = "~/.nanorc"
			default:
				home, err := os.UserHomeDir()
				if err == nil {
					homeTarget := filepath.Join(home, "."+name)
					configTarget := filepath.Join(home, ".config", name)
					if _, err := os.Stat(homeTarget); err == nil {
						if _, err := os.Stat(configTarget); os.IsNotExist(err) {
							systemPath = filepath.Join("~", "."+name)
						}
					}
				}
				if systemPath == "" {
					systemPath = filepath.Join("~/.config", name)
				}
			}
		}
	}

	cleanName := strings.TrimPrefix(name, ".")
	cleanName = strings.TrimSuffix(cleanName, filepath.Ext(cleanName))
	if cleanName == "" {
		cleanName = name
	}

	return config.DotfileSpec{
		Name:       cleanName,
		RepoPath:   name,
		SystemPath: systemPath,
		Method:     method,
		IsDir:      isDir,
	}
}

// ScanRepoSpecs scans top-level files and directories inside repoPath,
// inferring DotfileSpecs for non-ignored entries using .dotignore.
func ScanRepoSpecs(repoPath string) ([]config.DotfileSpec, error) {
	absPath := config.ExpandPath(repoPath)
	entries, err := os.ReadDir(absPath)
	if err != nil {
		return nil, err
	}

	dotIgnore := LoadDotIgnore(absPath)

	var specs []config.DotfileSpec
	for _, e := range entries {
		name := e.Name()
		info, err := e.Info()
		if err != nil {
			continue
		}

		if dotIgnore.Matches(name, info.IsDir()) {
			continue
		}

		spec := inferSpecFromEntry(name, info.IsDir())
		specs = append(specs, spec)
	}

	sort.Slice(specs, func(i, j int) bool {
		return strings.ToLower(specs[i].Name) < strings.ToLower(specs[j].Name)
	})

	return specs, nil
}

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
	specs, err := ScanRepoSpecs(repoPath)
	if err != nil {
		return nil, nil, err
	}

	cfg := config.DefaultConfig()
	cfg.RepoPath = repoPath
	cfg.Dotfiles = specs

	return specs, cfg, nil
}

func isAlreadyTracked(dotfiles []config.DotfileSpec, candidate config.DotfileSpec) bool {
	candCleanRepo := strings.TrimPrefix(filepath.Clean(candidate.RepoPath), ".")
	for _, existing := range dotfiles {
		if strings.EqualFold(existing.Name, candidate.Name) {
			return true
		}
		existCleanRepo := strings.TrimPrefix(filepath.Clean(existing.RepoPath), ".")
		if strings.EqualFold(existCleanRepo, candCleanRepo) {
			return true
		}
	}
	return false
}

// RefreshDatabase inspects the repository configured in cfg.RepoPath for any configurations
// (defined in repo config.toml or added as new files/directories) that are not yet tracked
// in cfg.Dotfiles.
//
// Newly discovered configurations are appended, cfg.Dotfiles is sorted alphabetically,
// RefreshDatabase inspects the repository configured in cfg.RepoPath for any configurations
// that exist on disk but are not yet tracked in the database (cfg.Dotfiles).
// If untracked configurations are discovered, they are appended to cfg.Dotfiles, sorted,
// and the configuration is saved to disk.
// Returns the slice of newly added DotfileSpecs and any error encountered.
func RefreshDatabase(cfg *config.Config) ([]config.DotfileSpec, error) {
	if cfg == nil || cfg.RepoPath == "" {
		return nil, nil
	}

	absPath := config.ExpandPath(cfg.RepoPath)
	info, err := os.Stat(absPath)
	if err != nil || !info.IsDir() {
		return nil, nil
	}

	dotIgnore := LoadDotIgnore(absPath)
	var candidates []config.DotfileSpec

	// Check for repo-level config.toml first
	if cfgFile := config.FindRepoConfigFile(absPath); cfgFile != "" {
		if repoCfg, err := config.LoadFromPath(cfgFile); err == nil && len(repoCfg.Dotfiles) > 0 {
			for _, s := range repoCfg.Dotfiles {
				if !dotIgnore.Matches(s.RepoPath, s.IsDir) {
					candidates = append(candidates, s)
				}
			}
		}
	}

	// Scan filesystem entries
	scanned, err := ScanRepoSpecs(absPath)
	if err == nil {
		candidates = append(candidates, scanned...)
	}

	var newlyAdded []config.DotfileSpec
	for _, cand := range candidates {
		if isAlreadyTracked(cfg.Dotfiles, cand) || isAlreadyTracked(newlyAdded, cand) {
			continue
		}
		newlyAdded = append(newlyAdded, cand)
		cfg.Dotfiles = append(cfg.Dotfiles, cand)
	}

	if len(newlyAdded) == 0 {
		return nil, nil
	}

	cfg.SortDotfiles()
	if err := config.Save(cfg); err != nil {
		return newlyAdded, fmt.Errorf("failed to save refreshed config: %w", err)
	}
	return newlyAdded, nil
}
