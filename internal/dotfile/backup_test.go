package dotfile

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BitwiseSang/dots/internal/config"
)

func TestDiffAndBackup_Unlinked(t *testing.T) {
	tmpDir := t.TempDir()
	sysDir := filepath.Join(tmpDir, "sys_fish")
	repoDir := filepath.Join(tmpDir, "repo_fish")
	_ = os.MkdirAll(sysDir, 0755)
	_ = os.MkdirAll(repoDir, 0755)

	_ = os.WriteFile(filepath.Join(sysDir, "config.fish"), []byte("set -x VAR 1\n"), 0644)
	_ = os.WriteFile(filepath.Join(repoDir, "config.fish"), []byte("set -x VAR 0\n"), 0644)

	spec := config.DotfileSpec{
		Name:       "fish",
		RepoPath:   "repo_fish",
		SystemPath: sysDir,
		Method:     "rsync",
		IsDir:      true,
	}
	entry := NewEntry(spec, tmpDir)

	diff, err := Diff(entry)
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}
	if !strings.Contains(diff, "-set -x VAR 0") || !strings.Contains(diff, "+set -x VAR 1") {
		t.Errorf("expected diff to show changes, got: %s", diff)
	}

	// Backup should sync sys to repo
	if err := Backup(entry); err != nil {
		t.Fatalf("Backup failed: %v", err)
	}

	repoData, _ := os.ReadFile(filepath.Join(repoDir, "config.fish"))
	if string(repoData) != "set -x VAR 1\n" {
		t.Errorf("expected repo file to be updated by Backup, got %s", string(repoData))
	}

	if status := entry.CheckStatus(); status != StatusInSync {
		t.Errorf("expected StatusInSync after backup, got %v", status)
	}
}

func TestDiffAndBackup_Linked(t *testing.T) {
	repoDir := t.TempDir()

	// Init git repo
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = repoDir
		_ = cmd.Run()
	}
	run("init")
	run("config", "user.name", "Test")
	run("config", "user.email", "test@example.com")

	tmuxRepo := filepath.Join(repoDir, "tmux.conf")
	_ = os.WriteFile(tmuxRepo, []byte("set -g prefix C-a\n"), 0644)
	run("add", ".")
	run("commit", "-m", "init tmux")

	sysDir := t.TempDir()
	sysLink := filepath.Join(sysDir, ".tmux.conf")
	_ = os.Symlink(tmuxRepo, sysLink)

	spec := config.DotfileSpec{
		Name:       "tmux",
		RepoPath:   "tmux.conf",
		SystemPath: sysLink,
		Method:     "copy",
		IsDir:      false,
	}
	entry := NewEntry(spec, repoDir)

	// Initially clean
	d, err := Diff(entry)
	if err != nil {
		t.Fatalf("Diff error: %v", err)
	}
	if d != "" {
		t.Errorf("expected empty diff on clean linked repo, got: %s", d)
	}

	// Modify file
	_ = os.WriteFile(tmuxRepo, []byte("set -g prefix C-b\n"), 0644)

	// Diff should return git diff!
	d, err = Diff(entry)
	if err != nil {
		t.Fatalf("Diff error: %v", err)
	}
	if !strings.Contains(d, "-set -g prefix C-a") || !strings.Contains(d, "+set -g prefix C-b") {
		t.Errorf("expected git diff for modified linked file, got: %s", d)
	}

	// BackupAll should not error on linked files
	results := BackupAll([]Entry{entry})
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Err != nil {
		t.Errorf("expected no error backing up linked entry, got: %v", results[0].Err)
	}
	if results[0].Skipped {
		t.Errorf("expected linked entry to not be marked skipped")
	}
}
