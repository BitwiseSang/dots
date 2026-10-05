package dotfile

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/BitwiseSang/dots/internal/config"
	"github.com/BitwiseSang/dots/internal/git"
)

func TestRemove_SymlinkMode(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "dots_remove_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	t.Setenv("HOME", tempDir)

	repoDir := filepath.Join(tempDir, "repo")
	sysDir := filepath.Join(tempDir, "sys")
	_ = os.MkdirAll(repoDir, 0755)
	_ = os.MkdirAll(sysDir, 0755)

	// Create repo file
	repoFile := filepath.Join(repoDir, "nvim.txt")
	_ = os.WriteFile(repoFile, []byte("repo nvim"), 0644)

	// Create symlink
	sysFile := filepath.Join(sysDir, "nvim.txt")
	if err := os.Symlink(repoFile, sysFile); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	cfg := &config.Config{
		RepoPath: repoDir,
		Dotfiles: []config.DotfileSpec{
			{Name: "nvim", RepoPath: "nvim.txt", SystemPath: sysFile},
		},
	}
	entries := LoadEntries(cfg)
	if len(entries) == 0 {
		t.Fatalf("failed to load entries")
	}

	res, err := Remove(entries[0], RemoveModeSymlink, cfg, false, "", false)
	if err != nil {
		t.Fatalf("Remove failed: %v", err)
	}
	if !res.SymlinkRemoved {
		t.Errorf("expected SymlinkRemoved to be true")
	}

	// Verify symlink is gone
	if _, err := os.Lstat(sysFile); !os.IsNotExist(err) {
		t.Errorf("expected sysFile to be removed")
	}
	// Verify repo file still exists!
	if _, err := os.Stat(repoFile); err != nil {
		t.Errorf("expected repoFile to remain intact: %v", err)
	}
	// Verify config still tracks entry
	if len(cfg.Dotfiles) != 1 {
		t.Errorf("expected cfg to still have 1 dotfile in symlink mode")
	}
}

func TestRemove_AllMode(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "dots_remove_all_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	t.Setenv("HOME", tempDir)

	repoDir := filepath.Join(tempDir, "repo")
	sysDir := filepath.Join(tempDir, "sys")
	_ = os.MkdirAll(repoDir, 0755)
	_ = os.MkdirAll(sysDir, 0755)

	// Create repo dir/file
	configRepoDir := filepath.Join(repoDir, "fish")
	_ = os.MkdirAll(configRepoDir, 0755)
	_ = os.WriteFile(filepath.Join(configRepoDir, "config.fish"), []byte("fish config"), 0644)

	// Create symlink
	sysFish := filepath.Join(sysDir, "fish")
	if err := os.Symlink(configRepoDir, sysFish); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	cfg := &config.Config{
		RepoPath: repoDir,
		Dotfiles: []config.DotfileSpec{
			{Name: "fish", RepoPath: "fish", SystemPath: sysFish, IsDir: true},
		},
	}
	entries := LoadEntries(cfg)

	res, err := Remove(entries[0], RemoveModeAll, cfg, false, "", false)
	if err != nil {
		t.Fatalf("Remove failed: %v", err)
	}
	if !res.SymlinkRemoved {
		t.Errorf("expected SymlinkRemoved to be true")
	}
	if !res.RepoFilesRemoved {
		t.Errorf("expected RepoFilesRemoved to be true")
	}
	if !res.ConfigRemoved {
		t.Errorf("expected ConfigRemoved to be true")
	}

	// Verify symlink is removed
	if _, err := os.Lstat(sysFish); !os.IsNotExist(err) {
		t.Errorf("expected sysFish symlink to be removed")
	}
	// Verify repo directory is removed
	if _, err := os.Stat(configRepoDir); !os.IsNotExist(err) {
		t.Errorf("expected configRepoDir to be deleted from repo")
	}
	// Verify entry is removed from cfg
	if len(cfg.Dotfiles) != 0 {
		t.Errorf("expected cfg.Dotfiles to be empty after remove all, got %d", len(cfg.Dotfiles))
	}
}

func TestRemove_SkipRegularFileInSymlinkMode(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "dots_remove_skip_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	t.Setenv("HOME", tempDir)

	repoDir := filepath.Join(tempDir, "repo")
	sysDir := filepath.Join(tempDir, "sys")
	_ = os.MkdirAll(repoDir, 0755)
	_ = os.MkdirAll(sysDir, 0755)

	repoFile := filepath.Join(repoDir, "tmux.conf")
	_ = os.WriteFile(repoFile, []byte("tmux config"), 0644)

	// Create REGULAR file at system path, NOT a symlink
	sysFile := filepath.Join(sysDir, "tmux.conf")
	_ = os.WriteFile(sysFile, []byte("my local unlinked tmux"), 0644)

	cfg := &config.Config{
		RepoPath: repoDir,
		Dotfiles: []config.DotfileSpec{
			{Name: "tmux", RepoPath: "tmux.conf", SystemPath: sysFile},
		},
	}
	entries := LoadEntries(cfg)

	res, err := Remove(entries[0], RemoveModeSymlink, cfg, false, "", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.SymlinkRemoved {
		t.Errorf("expected SymlinkRemoved to be false for regular file")
	}
	if !res.SymlinkSkipped {
		t.Errorf("expected SymlinkSkipped to be true")
	}
	// Verify sys file still exists untouched!
	content, _ := os.ReadFile(sysFile)
	if string(content) != "my local unlinked tmux" {
		t.Errorf("sys file was modified or deleted!")
	}
}

func TestRemove_WithGitCommit(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "dots_remove_git_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	t.Setenv("HOME", tempDir)

	repoDir := filepath.Join(tempDir, "repo")
	_ = os.MkdirAll(repoDir, 0755)

	// Init git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = repoDir
	_ = cmd.Run()
	_ = exec.Command("git", "-C", repoDir, "config", "user.name", "Test").Run()
	_ = exec.Command("git", "-C", repoDir, "config", "user.email", "test@test.com").Run()

	testFile := filepath.Join(repoDir, "kitty.conf")
	_ = os.WriteFile(testFile, []byte("kitty config"), 0644)

	_ = git.Add(repoDir)
	_ = git.Commit(repoDir, "initial commit")

	sysFile := filepath.Join(tempDir, "kitty.conf")
	_ = os.Symlink(testFile, sysFile)

	cfg := &config.Config{
		RepoPath: repoDir,
		Dotfiles: []config.DotfileSpec{
			{Name: "kitty", RepoPath: "kitty.conf", SystemPath: sysFile},
		},
	}
	entries := LoadEntries(cfg)

	res, err := Remove(entries[0], RemoveModeAll, cfg, true, "feat: remove kitty", false)
	if err != nil {
		t.Fatalf("Remove failed: %v", err)
	}
	if !res.GitCommitted {
		t.Errorf("expected GitCommitted to be true")
	}

	// Verify git status is clean (deletion was committed!)
	hasChanges, err := git.HasChanges(repoDir)
	if err != nil {
		t.Fatalf("HasChanges failed: %v", err)
	}
	if hasChanges {
		t.Errorf("expected clean git repo after commit, but has uncommitted changes")
	}
}

func TestRemove_SafetyCheck(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "dots_remove_safety_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	t.Setenv("HOME", tempDir)

	repoDir := filepath.Join(tempDir, "repo")
	_ = os.MkdirAll(repoDir, 0755)

	cfg := &config.Config{
		RepoPath: repoDir,
		Dotfiles: []config.DotfileSpec{
			// Unsafe target attempting to delete repo directory itself
			{Name: "evil", RepoPath: "", SystemPath: "/tmp/evil"},
		},
	}
	entries := LoadEntries(cfg)

	res, _ := Remove(entries[0], RemoveModeAll, cfg, false, "", false)
	if res.Err == nil {
		t.Errorf("expected safety check to fail when RepoPath is empty/repo itself")
	}
}
