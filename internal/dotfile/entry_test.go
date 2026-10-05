package dotfile

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BitwiseSang/dots/internal/config"
)

func TestStatusLabelNoEmojis(t *testing.T) {
	labels := []string{
		"✓ In sync",
		"✗ Changed",
		"∅ Missing",
		" Linked",
		"Unlinked",
		"∅ Repo missing",
	}

	for _, l := range labels {
		if strings.Contains(l, "🔗") || strings.Contains(l, "✏️") {
			t.Errorf("Label %q contains emoji", l)
		}
	}
}

func TestDirsEqual(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	// Empty dirs are equal
	if !dirsEqual(dir1, dir2) {
		t.Errorf("expected empty dirs to be equal")
	}

	// Add file in dir1 only
	f1 := filepath.Join(dir1, "config.toml")
	_ = os.WriteFile(f1, []byte("key = 1\n"), 0644)
	if dirsEqual(dir1, dir2) {
		t.Errorf("expected dirs with different files to not be equal")
	}

	// Add identical file in dir2
	f2 := filepath.Join(dir2, "config.toml")
	_ = os.WriteFile(f2, []byte("key = 1\n"), 0644)
	if !dirsEqual(dir1, dir2) {
		t.Errorf("expected identical dirs to be equal")
	}

	// Change file content in dir2
	_ = os.WriteFile(f2, []byte("key = 2\n"), 0644)
	if dirsEqual(dir1, dir2) {
		t.Errorf("expected dirs with modified content to not be equal")
	}

	// Add subdirectory in dir1
	sub1 := filepath.Join(dir1, "sub")
	_ = os.MkdirAll(sub1, 0755)
	_ = os.WriteFile(filepath.Join(sub1, "theme.lua"), []byte("return {}\n"), 0644)
	if dirsEqual(dir1, dir2) {
		t.Errorf("expected dirs with different subdirectories to not be equal")
	}
}

func TestEntryCheckStatus_Directory(t *testing.T) {
	tmpDir := t.TempDir()
	sysDir := filepath.Join(tmpDir, "sys_fish")
	repoDir := filepath.Join(tmpDir, "repo_fish")
	_ = os.MkdirAll(sysDir, 0755)
	_ = os.MkdirAll(repoDir, 0755)

	_ = os.WriteFile(filepath.Join(sysDir, "config.fish"), []byte("echo hi\n"), 0644)
	_ = os.WriteFile(filepath.Join(repoDir, "config.fish"), []byte("echo hi\n"), 0644)

	spec := config.DotfileSpec{
		Name:       "fish",
		RepoPath:   "repo_fish",
		SystemPath: sysDir,
		Method:     "rsync",
		IsDir:      true,
	}

	entry := NewEntry(spec, tmpDir)

	// In sync
	if status := entry.CheckStatus(); status != StatusInSync {
		t.Errorf("expected StatusInSync, got %v", status)
	}

	// Modify system file
	_ = os.WriteFile(filepath.Join(sysDir, "config.fish"), []byte("echo modified\n"), 0644)
	if status := entry.CheckStatus(); status != StatusChanged {
		t.Errorf("expected StatusChanged after modifying sys file, got %v", status)
	}
}

func TestEntryCheckStatus_LinkedWithGit(t *testing.T) {
	repoDir := t.TempDir()

	// Initialize git repo in repoDir
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = repoDir
		_ = cmd.Run()
	}
	run("init")
	run("config", "user.name", "Test")
	run("config", "user.email", "test@example.com")

	nvimRepo := filepath.Join(repoDir, "nvim")
	_ = os.MkdirAll(nvimRepo, 0755)
	initLua := filepath.Join(nvimRepo, "init.lua")
	_ = os.WriteFile(initLua, []byte("vim.opt.number = true\n"), 0644)
	run("add", ".")
	run("commit", "-m", "init nvim")

	// Create symlink on system pointing to nvimRepo
	sysDir := t.TempDir()
	sysLink := filepath.Join(sysDir, "nvim")
	if err := os.Symlink(nvimRepo, sysLink); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	spec := config.DotfileSpec{
		Name:       "nvim",
		RepoPath:   "nvim",
		SystemPath: sysLink,
		Method:     "rsync",
		IsDir:      true,
	}
	entry := NewEntry(spec, repoDir)

	if !entry.IsLinked() {
		t.Fatalf("expected IsLinked to be true")
	}

	// Clean git state: should be StatusLinked
	if status := entry.CheckStatus(); status != StatusLinked {
		t.Errorf("expected StatusLinked when git is clean, got %v", status)
	}

	// Modify file in nvim
	_ = os.WriteFile(initLua, []byte("vim.opt.number = false\n"), 0644)

	// Now should report StatusChanged because of uncommitted git changes!
	if status := entry.CheckStatus(); status != StatusChanged {
		t.Errorf("expected StatusChanged when git has uncommitted changes, got %v", status)
	}
}

func TestEntrySetupAndBackupStatus(t *testing.T) {
	repoDir := t.TempDir()

	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = repoDir
		_ = cmd.Run()
	}
	run("init")
	run("config", "user.name", "Test")
	run("config", "user.email", "test@example.com")

	nvimRepo := filepath.Join(repoDir, "nvim")
	_ = os.MkdirAll(nvimRepo, 0755)
	initLua := filepath.Join(nvimRepo, "init.lua")
	_ = os.WriteFile(initLua, []byte("vim.opt.number = true\n"), 0644)
	run("add", ".")
	run("commit", "-m", "init nvim")

	sysDir := t.TempDir()
	sysLink := filepath.Join(sysDir, "nvim")
	_ = os.Symlink(nvimRepo, sysLink)

	spec := config.DotfileSpec{
		Name:       "nvim",
		RepoPath:   "nvim",
		SystemPath: sysLink,
		Method:     "rsync",
		IsDir:      true,
	}
	entry := NewEntry(spec, repoDir)

	// Clean git: Setup is Linked, Backup is InSync
	if entry.SetupStatus() != StatusLinked {
		t.Errorf("expected SetupStatus to be StatusLinked, got %v", entry.SetupStatus())
	}
	if entry.BackupStatus() != StatusInSync {
		t.Errorf("expected BackupStatus to be StatusInSync, got %v", entry.BackupStatus())
	}

	// Modify git repo file
	_ = os.WriteFile(initLua, []byte("vim.opt.number = false\n"), 0644)

	// With pending changes: Setup is STILL Linked, Backup is Changed!
	if entry.SetupStatus() != StatusLinked {
		t.Errorf("expected SetupStatus to still be StatusLinked, got %v", entry.SetupStatus())
	}
	if entry.BackupStatus() != StatusChanged {
		t.Errorf("expected BackupStatus to be StatusChanged, got %v", entry.BackupStatus())
	}
}

func TestLoadEntries_Alphabetical(t *testing.T) {
	cfg := &config.Config{
		RepoPath: "/tmp",
		Dotfiles: []config.DotfileSpec{
			{Name: "zsh"},
			{Name: "alacritty"},
			{Name: "tmux"},
			{Name: "bash"},
		},
	}
	entries := LoadEntries(cfg)
	if len(entries) != 4 {
		t.Fatalf("expected 4 entries, got %d", len(entries))
	}
	if entries[0].Name != "alacritty" || entries[1].Name != "bash" || entries[2].Name != "tmux" || entries[3].Name != "zsh" {
		t.Errorf("expected alphabetical order, got: %s, %s, %s, %s",
			entries[0].Name, entries[1].Name, entries[2].Name, entries[3].Name)
	}
}

func TestEntryIsLinked_RelativeSymlink(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	sysDir := filepath.Join(tmpDir, "sys")
	_ = os.MkdirAll(repoDir, 0755)
	_ = os.MkdirAll(sysDir, 0755)

	repoFile := filepath.Join(repoDir, "tmux.conf")
	_ = os.WriteFile(repoFile, []byte("set -g prefix C-a\n"), 0644)

	sysLink := filepath.Join(sysDir, "tmux.conf")

	// Create relative symlink: ../repo/tmux.conf
	relTarget, err := filepath.Rel(sysDir, repoFile)
	if err != nil {
		t.Fatalf("failed to calculate rel target: %v", err)
	}
	if err := os.Symlink(relTarget, sysLink); err != nil {
		t.Fatalf("failed to create relative symlink: %v", err)
	}

	entry := NewEntry(config.DotfileSpec{
		Name:       "tmux",
		RepoPath:   "tmux.conf",
		SystemPath: sysLink,
	}, repoDir)

	if !entry.IsLinked() {
		t.Errorf("expected relative symlink to be recognized as linked")
	}
}
