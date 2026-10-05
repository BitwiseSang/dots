package dotfile

import (
	"strings"
	"testing"
)

func TestDefaultIgnorePatterns_NoShellFiles(t *testing.T) {
	patterns := DefaultIgnorePatterns()
	for _, p := range patterns {
		if strings.Contains(p, ".sh") {
			t.Errorf("expected no shell files in default ignore patterns, found: %s", p)
		}
	}
}

func TestDotIgnore_Matching(t *testing.T) {
	di := &DotIgnore{
		Patterns: []string{
			"README*",
			"LICENSE*",
			"build/",
			"*.log",
			"!important.log",
		},
	}

	// Safety invariants
	if !di.Matches(".git", true) || !di.Matches(".git", false) {
		t.Errorf("expected .git to always be ignored")
	}
	if !di.Matches(".dotignore", false) {
		t.Errorf("expected .dotignore to always be ignored")
	}

	// Globs
	if !di.Matches("README.md", false) {
		t.Errorf("expected README.md to be ignored")
	}
	if !di.Matches("readme.txt", false) {
		t.Errorf("expected readme.txt to be ignored (case-insensitive)")
	}
	if !di.Matches("LICENSE", false) {
		t.Errorf("expected LICENSE to be ignored")
	}

	// Directory only
	if !di.Matches("build", true) {
		t.Errorf("expected directory 'build' to be ignored")
	}
	if di.Matches("build", false) {
		t.Errorf("expected regular file 'build' to NOT be ignored by 'build/'")
	}

	// Wildcards & Negation
	if !di.Matches("debug.log", false) {
		t.Errorf("expected debug.log to be ignored by *.log")
	}
	if di.Matches("important.log", false) {
		t.Errorf("expected important.log to be unignored by !important.log")
	}

	// Config files that should NOT be ignored
	if di.Matches(".bashrc", false) {
		t.Errorf("expected .bashrc to NOT be ignored")
	}
	if di.Matches(".zshrc", false) {
		t.Errorf("expected .zshrc to NOT be ignored")
	}
	if di.Matches("nvim", true) {
		t.Errorf("expected nvim directory to NOT be ignored")
	}
	if di.Matches("setup.sh", false) {
		t.Errorf("expected setup.sh to NOT be ignored unless explicitly in patterns")
	}
}

func TestDotIgnore_FilePersistence(t *testing.T) {
	tmpDir := t.TempDir()

	// Initially does not exist
	if DotIgnoreExists(tmpDir) {
		t.Errorf("expected DotIgnoreExists to be false")
	}

	di := LoadDotIgnore(tmpDir)
	if len(di.Patterns) != len(DefaultIgnorePatterns()) {
		t.Errorf("expected default patterns count %d, got %d", len(DefaultIgnorePatterns()), len(di.Patterns))
	}

	// Add custom pattern and save
	err := di.AddPattern("secret.env")
	if err != nil {
		t.Fatalf("AddPattern failed: %v", err)
	}

	if !DotIgnoreExists(tmpDir) {
		t.Errorf("expected DotIgnoreExists to be true after save")
	}

	// Reload from disk
	di2 := LoadDotIgnore(tmpDir)
	if !di2.HasPattern("secret.env") {
		t.Errorf("expected reloaded DotIgnore to have 'secret.env'")
	}
	if !di2.Matches("secret.env", false) {
		t.Errorf("expected secret.env to match ignore")
	}

	// Remove pattern
	err = di2.RemovePattern("secret.env")
	if err != nil {
		t.Fatalf("RemovePattern failed: %v", err)
	}
	if di2.HasPattern("secret.env") {
		t.Errorf("expected secret.env to be removed")
	}

	// Verify on disk
	di3 := LoadDotIgnore(tmpDir)
	if di3.HasPattern("secret.env") {
		t.Errorf("expected disk .dotignore to not have secret.env")
	}
}
