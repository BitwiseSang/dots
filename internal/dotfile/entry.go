package dotfile

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BitwiseSang/dots/internal/config"
	"github.com/BitwiseSang/dots/internal/git"
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
	sort.Slice(entries, func(i, j int) bool {
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})
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

// filesEqual compares two files in streaming chunks of 32KB without loading the entire files into memory.
func filesEqual(p1, p2 string) bool {
	f1, err := os.Open(p1)
	if err != nil {
		return false
	}
	defer f1.Close()

	f2, err := os.Open(p2)
	if err != nil {
		return false
	}
	defer f2.Close()

	b1 := make([]byte, 32*1024)
	b2 := make([]byte, 32*1024)
	for {
		n1, err1 := io.ReadFull(f1, b1)
		n2, err2 := io.ReadFull(f2, b2)
		if n1 != n2 || !bytes.Equal(b1[:n1], b2[:n2]) {
			return false
		}
		if err1 == io.EOF || err1 == io.ErrUnexpectedEOF {
			return err2 == io.EOF || err2 == io.ErrUnexpectedEOF
		}
		if err1 != nil || err2 != nil {
			return false
		}
	}
}

// dirsEqual recursively compares all files and subdirectories between two directory trees.
// Returns true only if file counts, relative structure, sizes, and contents match completely.
func dirsEqual(dir1, dir2 string) bool {
	files1 := make(map[string]os.FileInfo)
	_ = filepath.Walk(dir1, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		rel, err := filepath.Rel(dir1, path)
		if err != nil || rel == "." {
			return nil
		}
		files1[rel] = info
		return nil
	})

	files2 := make(map[string]os.FileInfo)
	_ = filepath.Walk(dir2, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		rel, err := filepath.Rel(dir2, path)
		if err != nil || rel == "." {
			return nil
		}
		files2[rel] = info
		return nil
	})

	if len(files1) != len(files2) {
		return false
	}

	for rel, info1 := range files1 {
		info2, ok := files2[rel]
		if !ok {
			return false
		}
		if info1.IsDir() != info2.IsDir() {
			return false
		}
		if !info1.IsDir() {
			if info1.Size() != info2.Size() {
				return false
			}
			if info1.Size() == 0 {
				continue
			}
			if !filesEqual(filepath.Join(dir1, rel), filepath.Join(dir2, rel)) {
				return false
			}
		}
	}
	return true
}

// IsLinked returns true if the system path is a symlink pointing to the repository path.
// Handles both absolute and relative symlink targets correctly.
func (e Entry) IsLinked() bool {
	sysPath := e.ResolveSystemPath()
	repoPath := e.AbsRepoPath()

	sysInfo, err := os.Lstat(sysPath)
	if err == nil && sysInfo.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(sysPath)
		if err == nil {
			if !filepath.IsAbs(target) {
				target = filepath.Join(filepath.Dir(sysPath), target)
			}
			if filepath.Clean(target) == filepath.Clean(repoPath) {
				return true
			}
			evalTarget, err1 := filepath.EvalSymlinks(sysPath)
			evalRepo, err2 := filepath.EvalSymlinks(repoPath)
			if err1 == nil && err2 == nil && filepath.Clean(evalTarget) == filepath.Clean(evalRepo) {
				return true
			}
		}
	}
	return false
}

// HasGitChanges returns true if this entry has uncommitted or untracked changes in the git repository.
func (e Entry) HasGitChanges() bool {
	if git.IsRepo(e.repoRoot) {
		hasChanges, _ := git.HasChangesForPath(e.repoRoot, e.RepoPath)
		return hasChanges
	}
	return false
}

func (e Entry) CheckStatus() Status {
	sysPath := e.ResolveSystemPath()
	repoPath := e.AbsRepoPath()

	sysInfo, sysErr := os.Lstat(sysPath)
	if sysErr != nil && os.IsNotExist(sysErr) {
		return StatusMissing
	}

	repoInfo, repoErr := os.Stat(repoPath)
	if repoErr != nil && os.IsNotExist(repoErr) {
		return StatusRepoMissing
	}

	if e.IsLinked() {
		// When linked, check if the repository files have uncommitted git changes
		if e.HasGitChanges() {
			return StatusChanged
		}
		return StatusLinked
	}

	if e.IsDir {
		if sysErr == nil && repoErr == nil && sysInfo.IsDir() && repoInfo.IsDir() {
			if !dirsEqual(sysPath, repoPath) {
				return StatusChanged
			}
			if e.HasGitChanges() {
				return StatusChanged
			}
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

	if !bytes.Equal(sysData, repoData) {
		return StatusChanged
	}

	if e.HasGitChanges() {
		return StatusChanged
	}

	return StatusInSync
}

// HasChanges returns true if the system and repository differ, or if there are uncommitted git changes.
func (e Entry) HasChanges() bool {
	sysPath := e.ResolveSystemPath()
	repoPath := e.AbsRepoPath()

	sysInfo, sysErr := os.Lstat(sysPath)
	if sysErr != nil {
		return false
	}
	repoInfo, repoErr := os.Stat(repoPath)
	if repoErr != nil {
		return true // New in system, missing from repo
	}

	if e.IsLinked() {
		return e.HasGitChanges()
	}

	if e.IsDir {
		if sysInfo.IsDir() && repoInfo.IsDir() {
			if !dirsEqual(sysPath, repoPath) {
				return true
			}
		} else {
			return true
		}
	} else {
		sysData, err1 := os.ReadFile(sysPath)
		repoData, err2 := os.ReadFile(repoPath)
		if err1 != nil || err2 != nil || !bytes.Equal(sysData, repoData) {
			return true
		}
	}

	return e.HasGitChanges()
}

// SetupStatus returns the status relevant for the setup view (focused on symlink state).
func (e Entry) SetupStatus() Status {
	repoPath := e.AbsRepoPath()
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		return StatusRepoMissing
	}
	if e.IsLinked() {
		return StatusLinked
	}
	sysPath := e.ResolveSystemPath()
	if _, err := os.Lstat(sysPath); os.IsNotExist(err) {
		return StatusMissing
	}
	return StatusUnlinked
}

func (e Entry) SetupStatusLabel() string {
	switch e.SetupStatus() {
	case StatusLinked:
		return " Linked"
	case StatusUnlinked:
		return "⊘ Unlinked"
	case StatusMissing:
		return "∅ Missing"
	case StatusRepoMissing:
		return "∅ Repo missing"
	default:
		return "? Unknown"
	}
}

// BackupStatus returns the status relevant for the backup view (focused on pending changes).
func (e Entry) BackupStatus() Status {
	sysPath := e.ResolveSystemPath()
	if _, err := os.Lstat(sysPath); os.IsNotExist(err) {
		return StatusMissing
	}

	repoPath := e.AbsRepoPath()
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		return StatusRepoMissing
	}

	if e.HasChanges() {
		return StatusChanged
	}
	return StatusInSync
}

func (e Entry) BackupStatusLabel() string {
	switch e.BackupStatus() {
	case StatusInSync:
		return "✓ In sync"
	case StatusChanged:
		return "✗ Changed"
	case StatusMissing:
		return "∅ Missing"
	case StatusRepoMissing:
		return "+ New"
	default:
		return "? Unknown"
	}
}

// RemoveStatus returns the status relevant for the remove view.
func (e Entry) RemoveStatus() Status {
	repoPath := e.AbsRepoPath()
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		return StatusRepoMissing
	}
	if e.IsLinked() {
		return StatusLinked
	}
	return StatusUnlinked
}

func (e Entry) RemoveStatusLabel() string {
	switch e.RemoveStatus() {
	case StatusLinked:
		return " Linked"
	case StatusUnlinked:
		return "⊘ Unlinked"
	case StatusRepoMissing:
		return "∅ Repo missing"
	default:
		return "? Unknown"
	}
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
		return " Linked"
	case StatusUnlinked:
		return "⊘ Unlinked"
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
