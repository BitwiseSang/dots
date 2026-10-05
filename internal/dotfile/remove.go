package dotfile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BitwiseSang/dots/internal/config"
	"github.com/BitwiseSang/dots/internal/git"
)

// RemoveMode represents the strategy for removing a configuration.
type RemoveMode int

const (
	// RemoveModeSymlink unlinks the system symlink only, preserving repository files and tracking.
	RemoveModeSymlink RemoveMode = iota
	// RemoveModeAll unlinks the system symlink, deletes files from the repository, and untracks from dots.
	RemoveModeAll
)

func (m RemoveMode) String() string {
	switch m {
	case RemoveModeSymlink:
		return "symlink"
	case RemoveModeAll:
		return "all"
	default:
		return "unknown"
	}
}

// RemoveResult records the actions taken during configuration removal.
type RemoveResult struct {
	Entry            Entry
	Mode             RemoveMode
	SymlinkRemoved   bool
	SymlinkSkipped   bool
	RepoFilesRemoved bool
	ConfigRemoved    bool
	GitCommitted     bool
	GitPushed        bool
	Err              error
	Message          string
}

// Remove removes a single configuration according to the specified mode.
func Remove(entry Entry, mode RemoveMode, cfg *config.Config, commitGit bool, commitMsg string, pushGit bool) (*RemoveResult, error) {
	results, err := RemoveAll([]Entry{entry}, mode, cfg, commitGit, commitMsg, pushGit)
	if len(results) > 0 {
		return results[0], err
	}
	return nil, err
}

// RemoveAll removes multiple configurations according to the specified mode.
func RemoveAll(entries []Entry, mode RemoveMode, cfg *config.Config, commitGit bool, commitMsg string, pushGit bool) ([]*RemoveResult, error) {
	var results []*RemoveResult
	var stagedPaths []string
	var removedNames []string

	absRepo := ""
	if cfg != nil && cfg.RepoPath != "" {
		absRepo = filepath.Clean(config.ExpandPath(cfg.RepoPath))
	}

	for _, entry := range entries {
		res := &RemoveResult{
			Entry: entry,
			Mode:  mode,
		}

		// 1. Unlink system symlink
		sysPath := entry.ResolveSystemPath()
		if info, err := os.Lstat(sysPath); err == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				if err := os.Remove(sysPath); err != nil {
					res.Err = fmt.Errorf("failed to remove symlink at %s: %w", sysPath, err)
					results = append(results, res)
					continue
				}
				res.SymlinkRemoved = true
			} else {
				// Regular file/directory: skip deletion in symlink mode to protect user files
				res.SymlinkSkipped = true
				res.Message = fmt.Sprintf("system path %s is not a symlink; skipped", sysPath)
			}
		}

		// 2. Also check alternative paths for symlinks pointing to this entry
		for _, alt := range entry.AltPaths {
			expAlt := config.ExpandPath(alt)
			if info, err := os.Lstat(expAlt); err == nil && (info.Mode()&os.ModeSymlink != 0) {
				_ = os.Remove(expAlt)
			}
		}

		// 3. If mode is RemoveModeAll, remove repository files
		if mode == RemoveModeAll {
			repoPath := entry.AbsRepoPath()
			absTarget := filepath.Clean(repoPath)

			// Safety check: ensure target is strictly a subdirectory/file within repo, not repo or system root
			home, _ := os.UserHomeDir()
			if absRepo == "" || absTarget == absRepo || absTarget == "/" || (home != "" && absTarget == home) {
				res.Err = fmt.Errorf("safety check failed: refusing to delete root or repo root: %s", absTarget)
				results = append(results, res)
				continue
			}
			if !strings.HasPrefix(absTarget, absRepo+string(filepath.Separator)) {
				res.Err = fmt.Errorf("safety check failed: %s is not inside repo %s", absTarget, absRepo)
				results = append(results, res)
				continue
			}

			if _, err := os.Stat(absTarget); err == nil {
				if err := os.RemoveAll(absTarget); err != nil {
					res.Err = fmt.Errorf("failed to delete repository files at %s: %w", absTarget, err)
					results = append(results, res)
					continue
				}
				res.RepoFilesRemoved = true
			}

			if entry.RepoPath != "" {
				stagedPaths = append(stagedPaths, entry.RepoPath)
			}
			removedNames = append(removedNames, entry.Name)

			if cfg != nil {
				cfg.RemoveDotfile(entry.Name)
				res.ConfigRemoved = true
			}
		}

		results = append(results, res)
	}

	// If mode is RemoveModeAll and configs were removed, save config files and stage git
	if mode == RemoveModeAll && cfg != nil && len(removedNames) > 0 {
		_ = config.Save(cfg)

		// Also update repo-level config if present
		if repoCfgPath := config.FindRepoConfigFile(cfg.RepoPath); repoCfgPath != "" {
			if rCfg, err := config.LoadFromPath(repoCfgPath); err == nil {
				for _, name := range removedNames {
					rCfg.RemoveDotfile(name)
				}
				_ = config.SaveToPath(rCfg, repoCfgPath)
				if rel, err := filepath.Rel(absRepo, repoCfgPath); err == nil {
					stagedPaths = append(stagedPaths, rel)
				}
			}
		}

		// Git stage and commit
		if commitGit && absRepo != "" && git.IsRepo(absRepo) && len(stagedPaths) > 0 {
			if err := git.AddPaths(absRepo, stagedPaths); err == nil {
				if commitMsg == "" {
					commitMsg = fmt.Sprintf("feat(config): remove %s config", strings.Join(removedNames, ", "))
				}
				if err := git.Commit(absRepo, commitMsg); err == nil {
					for _, res := range results {
						res.GitCommitted = true
					}
					if pushGit {
						if err := git.Push(absRepo); err == nil {
							for _, res := range results {
								res.GitPushed = true
							}
						}
					}
				}
			}
		}
	}

	return results, nil
}
