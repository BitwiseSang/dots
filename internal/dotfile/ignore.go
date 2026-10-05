package dotfile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BitwiseSang/dots/internal/config"
)

// DefaultIgnorePatterns returns the default patterns to ignore when scanning a dotfiles repository.
// Note: Shell files (*.sh) are intentionally omitted from defaults as instructed.
func DefaultIgnorePatterns() []string {
	return []string{
		".git",
		".git*",
		".github",
		".dotignore",
		"README*",
		"LICENSE*",
		"LICENCE*",
	}
}

// DotIgnore manages ignore patterns for a dotfiles repository.
type DotIgnore struct {
	Patterns []string
	RepoPath string
}

// LoadDotIgnore loads .dotignore from repoPath, combining default ignore patterns
// with any custom patterns or negations defined in the repository's .dotignore file.
func LoadDotIgnore(repoPath string) *DotIgnore {
	absPath := config.ExpandPath(repoPath)
	ignoreFile := filepath.Join(absPath, ".dotignore")

	data, err := os.ReadFile(ignoreFile)
	if err != nil {
		return &DotIgnore{
			Patterns: DefaultIgnorePatterns(),
			RepoPath: repoPath,
		}
	}

	patterns := DefaultIgnorePatterns()
	seen := make(map[string]bool)
	for _, p := range patterns {
		seen[strings.ToLower(p)] = true
	}

	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		// If negation (e.g. !README* or !Makefile), remove the positive default so negation takes effect cleanly
		if strings.HasPrefix(trimmed, "!") {
			pos := strings.TrimPrefix(trimmed, "!")
			for i, p := range patterns {
				if strings.EqualFold(p, pos) {
					patterns = append(patterns[:i], patterns[i+1:]...)
					break
				}
			}
			patterns = append(patterns, trimmed)
			continue
		}

		if !seen[strings.ToLower(trimmed)] {
			patterns = append(patterns, trimmed)
			seen[strings.ToLower(trimmed)] = true
		}
	}

	return &DotIgnore{
		Patterns: patterns,
		RepoPath: repoPath,
	}
}

// DotIgnoreExists reports whether .dotignore file exists in the repository.
func DotIgnoreExists(repoPath string) bool {
	absPath := config.ExpandPath(repoPath)
	ignoreFile := filepath.Join(absPath, ".dotignore")
	info, err := os.Stat(ignoreFile)
	return err == nil && !info.IsDir()
}

// Matches returns true if the specified name (file or directory) matches any ignore pattern.
func (d *DotIgnore) Matches(name string, isDir bool) bool {
	lower := strings.ToLower(name)
	// Safety invariants: .git and .dotignore are always ignored
	if lower == ".git" || lower == ".dotignore" {
		return true
	}

	matched := false
	for _, pattern := range d.Patterns {
		pat := strings.TrimSpace(pattern)
		if pat == "" || strings.HasPrefix(pat, "#") {
			continue
		}

		negated := strings.HasPrefix(pat, "!")
		if negated {
			pat = strings.TrimPrefix(pat, "!")
		}

		dirOnly := strings.HasSuffix(pat, "/")
		if dirOnly {
			pat = strings.TrimSuffix(pat, "/")
		}

		if dirOnly && !isDir {
			continue
		}

		if matchPattern(pat, name) {
			matched = !negated
		}
	}

	return matched
}

func matchPattern(pattern, name string) bool {
	pLower := strings.ToLower(pattern)
	nLower := strings.ToLower(name)

	if pLower == nLower {
		return true
	}

	if ok, _ := filepath.Match(pLower, nLower); ok {
		return true
	}

	return false
}

// HasPattern reports whether pattern is already in Patterns.
func (d *DotIgnore) HasPattern(pattern string) bool {
	for _, p := range d.Patterns {
		if strings.EqualFold(p, pattern) {
			return true
		}
	}
	return false
}

// AddPattern adds a pattern to d.Patterns if not already present and saves to disk.
func (d *DotIgnore) AddPattern(pattern string) error {
	trimmed := strings.TrimSpace(pattern)
	if trimmed == "" {
		return nil
	}
	if !d.HasPattern(trimmed) {
		d.Patterns = append(d.Patterns, trimmed)
	}
	return d.Save()
}

// RemovePattern removes a pattern from d.Patterns and saves to disk.
func (d *DotIgnore) RemovePattern(pattern string) error {
	trimmed := strings.TrimSpace(pattern)
	var newPatterns []string
	for _, p := range d.Patterns {
		if !strings.EqualFold(p, trimmed) {
			newPatterns = append(newPatterns, p)
		}
	}
	d.Patterns = newPatterns
	return d.Save()
}

// Save writes the current patterns to the .dotignore file in the repository.
func (d *DotIgnore) Save() error {
	if d.RepoPath == "" {
		return nil
	}

	absPath := config.ExpandPath(d.RepoPath)
	if err := os.MkdirAll(absPath, 0755); err != nil {
		return err
	}

	ignoreFile := filepath.Join(absPath, ".dotignore")

	var sb strings.Builder
	sb.WriteString("# .dotignore - Configurations and files ignored by dots\n")
	sb.WriteString("# Standard glob patterns supported (e.g. *.md, temp/, !keep.md)\n\n")

	for _, p := range d.Patterns {
		sb.WriteString(p)
		sb.WriteString("\n")
	}

	tmpFile := filepath.Join(absPath, fmt.Sprintf(".dotignore_%d.tmp", os.Getpid()))
	if err := os.WriteFile(tmpFile, []byte(sb.String()), 0644); err != nil {
		return err
	}
	return os.Rename(tmpFile, ignoreFile)
}
