package dotfile

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BitwiseSang/dots/internal/config"
)

func isIgnoredRepoEntry(name string) bool {
	lower := strings.ToLower(name)
	if strings.HasPrefix(lower, ".") {
		return true
	}
	if strings.HasPrefix(lower, "readme") || strings.HasPrefix(lower, "license") || strings.HasPrefix(lower, "licence") {
		return true
	}
	switch lower {
	case "makefile", "justfile", "dots", "dots.toml",
		"setup.sh", "install.sh", "backup.sh", "sync.sh", "bootstrap.sh",
		"go.mod", "go.sum", "shell.nix", "flake.nix", "flake.lock":
		return true
	}
	return false
}

func inferSpecFromEntry(name string, isDir bool) config.DotfileSpec {
	method := "copy"
	systemPath := filepath.Join("~/.config", name)

	if isDir {
		method = "rsync"
	} else {
		switch strings.ToLower(name) {
		case "tmux.conf", ".tmux.conf":
			systemPath = "~/.tmux.conf"
		case "zshrc", ".zshrc":
			systemPath = "~/.zshrc"
		case "bashrc", ".bashrc":
			systemPath = "~/.bashrc"
		case "gitconfig", ".gitconfig":
			systemPath = "~/.gitconfig"
		case "vimrc", ".vimrc":
			systemPath = "~/.vimrc"
		case "xinitrc", ".xinitrc":
			systemPath = "~/.xinitrc"
		case "xprofile", ".xprofile":
			systemPath = "~/.xprofile"
		case "profile", ".profile":
			systemPath = "~/.profile"
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
// inferring DotfileSpecs for non-ignored entries.
func ScanRepoSpecs(repoPath string) ([]config.DotfileSpec, error) {
	absPath := config.ExpandPath(repoPath)
	entries, err := os.ReadDir(absPath)
	if err != nil {
		return nil, err
	}

	var specs []config.DotfileSpec
	for _, e := range entries {
		name := e.Name()
		if isIgnoredRepoEntry(name) {
			continue
		}

		info, err := e.Info()
		if err != nil {
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
// and the configuration is saved to disk.
// Returns the slice of newly added DotfileSpecs and a boolean indicating whether any changes were made.
func RefreshDatabase(cfg *config.Config) ([]config.DotfileSpec, bool) {
	if cfg == nil || cfg.RepoPath == "" {
		return nil, false
	}

	absPath := config.ExpandPath(cfg.RepoPath)
	info, err := os.Stat(absPath)
	if err != nil || !info.IsDir() {
		return nil, false
	}

	var candidates []config.DotfileSpec

	// Check for repo-level config.toml first
	if cfgFile := config.FindRepoConfigFile(absPath); cfgFile != "" {
		if repoCfg, err := config.LoadFromPath(cfgFile); err == nil && len(repoCfg.Dotfiles) > 0 {
			candidates = append(candidates, repoCfg.Dotfiles...)
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
		return nil, false
	}

	cfg.SortDotfiles()
	_ = config.Save(cfg)
	return newlyAdded, true
}
