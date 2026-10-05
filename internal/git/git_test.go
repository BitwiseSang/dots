package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func initTestGitRepo(t *testing.T) string {
	dir := t.TempDir()

	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %s failed: %v, output: %s", strings.Join(args, " "), err, string(out))
		}
	}

	run("init")
	run("config", "user.name", "Dots Test")
	run("config", "user.email", "test@example.com")

	// Initial commit
	initialFile := filepath.Join(dir, "README.md")
	_ = os.WriteFile(initialFile, []byte("# Test Repo\n"), 0644)
	run("add", ".")
	run("commit", "-m", "initial commit")

	return dir
}

func TestGitIsRepo(t *testing.T) {
	repo := initTestGitRepo(t)
	if !IsRepo(repo) {
		t.Errorf("expected IsRepo to be true for git repo")
	}

	nonRepo := t.TempDir()
	if IsRepo(nonRepo) {
		t.Errorf("expected IsRepo to be false for empty directory")
	}
}

func TestGitHasChangesAndDiff(t *testing.T) {
	repo := initTestGitRepo(t)

	// Clean state
	hasChanges, err := HasChanges(repo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hasChanges {
		t.Errorf("expected no changes in fresh repo")
	}

	// Create subdirectory and file for a dotfile config
	nvimDir := filepath.Join(repo, "nvim")
	_ = os.MkdirAll(nvimDir, 0755)
	initLua := filepath.Join(nvimDir, "init.lua")
	_ = os.WriteFile(initLua, []byte("vim.opt.number = true\n"), 0644)

	// Check HasChanges overall
	hasChanges, err = HasChanges(repo)
	if err != nil || !hasChanges {
		t.Errorf("expected HasChanges to be true after adding file")
	}

	// Check HasChangesForPath for "nvim"
	hasNvimChanges, err := HasChangesForPath(repo, "nvim")
	if err != nil || !hasNvimChanges {
		t.Errorf("expected HasChangesForPath('nvim') to be true")
	}

	// Check HasChangesForPath for nonexistent "fish"
	hasFishChanges, err := HasChangesForPath(repo, "fish")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hasFishChanges {
		t.Errorf("expected HasChangesForPath('fish') to be false")
	}

	// DiffPath for "nvim" should show untracked file
	diff, err := DiffPath(repo, "nvim")
	if err != nil {
		t.Fatalf("DiffPath error: %v", err)
	}
	if !strings.Contains(diff, "nvim/init.lua") {
		t.Errorf("expected diff to mention nvim/init.lua, got: %s", diff)
	}

	// Add and commit
	if err := Add(repo); err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	msg := CommitMessage("Test backup")
	if err := Commit(repo, msg); err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	// After commit, repo should be clean
	hasChanges, _ = HasChanges(repo)
	if hasChanges {
		t.Errorf("expected clean repo after commit")
	}

	// Modify file
	_ = os.WriteFile(initLua, []byte("vim.opt.number = false\n"), 0644)
	diffModified, err := DiffPath(repo, "nvim")
	if err != nil {
		t.Fatalf("DiffPath error: %v", err)
	}
	if !strings.Contains(diffModified, "-vim.opt.number = true") || !strings.Contains(diffModified, "+vim.opt.number = false") {
		t.Errorf("expected unified diff for modified file, got: %s", diffModified)
	}
}
