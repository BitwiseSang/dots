package dotfile

import (
	"bytes"
	"os"
	"path/filepath"

	"github.com/BitwiseSang/dots/internal/config"
)

type SyncMethod int

const (
	SyncCopy SyncMethod = iota
	SyncRsync
)

type Status int

const (
	StatusUnknown Status = iota
	StatusInSync
	StatusChanged
	StatusMissing
	StatusLinked
	StatusUnlinked
	StatusRepoMissing
)

type Entry struct {
	Name       string
	RepoPath   string // Relative to repo root
	SystemPath string // Absolute path on system
	AltPaths   []string // Alternative system paths
	Method     SyncMethod
	IsDir      bool
	repoRoot   string
}

func NewEntry(spec config.DotfileSpec, repoRoot string) Entry {
	method := SyncCopy
	if spec.Method == "rsync" {
		method = SyncRsync
	}
	
	altPaths := make([]string, len(spec.AltPaths))
	for i, p := range spec.AltPaths {
		altPaths[i] = config.ExpandPath(p)
	}

	return Entry{
		Name:       spec.Name,
		RepoPath:   spec.RepoPath,
		SystemPath: config.ExpandPath(spec.SystemPath),
		AltPaths:   altPaths,
		Method:     method,
		IsDir:      spec.IsDir,
		repoRoot:   config.ExpandPath(repoRoot),
	}
}

func LoadEntries(cfg *config.Config) []Entry {
	var entries []Entry
	repoRoot := cfg.RepoPath
	for _, spec := range cfg.Dotfiles {
		entries = append(entries, NewEntry(spec, repoRoot))
	}
	return entries
}

func (e Entry) ResolveSystemPath() string {
	if _, err := os.Lstat(e.SystemPath); err == nil {
		return e.SystemPath
	}
	for _, alt := range e.AltPaths {
		if _, err := os.Lstat(alt); err == nil {
			return alt
		}
	}
	return e.SystemPath
}

func (e Entry) AbsRepoPath() string {
	return filepath.Join(e.repoRoot, e.RepoPath)
}

func (e Entry) CheckStatus() Status {
	sysPath := e.ResolveSystemPath()
	repoPath := e.AbsRepoPath()

	sysInfo, sysErr := os.Lstat(sysPath)
	if sysErr != nil && os.IsNotExist(sysErr) {
		return StatusMissing
	}

	if sysInfo != nil && sysInfo.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(sysPath)
		if err == nil {
			if filepath.Clean(target) == filepath.Clean(repoPath) {
				return StatusLinked
			}
		}
	}

	repoInfo, repoErr := os.Stat(repoPath)
	if repoErr != nil && os.IsNotExist(repoErr) {
		return StatusRepoMissing
	}

	if e.IsDir {
		if sysErr == nil && repoErr == nil && sysInfo.IsDir() && repoInfo.IsDir() {
			return StatusInSync
		}
		return StatusChanged
	}

	if sysErr != nil || repoErr != nil {
		return StatusChanged
	}

	sysData, err1 := os.ReadFile(sysPath)
	repoData, err2 := os.ReadFile(repoPath)
	if err1 != nil || err2 != nil {
		return StatusChanged
	}

	if bytes.Equal(sysData, repoData) {
		return StatusInSync
	}
	return StatusChanged
}

func (e Entry) StatusLabel() string {
	switch e.CheckStatus() {
	case StatusInSync:
		return "✓ In sync"
	case StatusChanged:
		return "✗ Changed"
	case StatusMissing:
		return "∅ Missing"
	case StatusLinked:
		return "🔗 Linked"
	case StatusUnlinked:
		return "Unlinked"
	case StatusRepoMissing:
		return "∅ Repo missing"
	default:
		return "? Unknown"
	}
}

func (e Entry) EditPath() string {
	status := e.CheckStatus()
	if status == StatusLinked {
		return e.ResolveSystemPath()
	}
	return e.AbsRepoPath()
}
