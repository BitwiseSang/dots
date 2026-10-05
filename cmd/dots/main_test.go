package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/BitwiseSang/dots/internal/config"
	"github.com/BitwiseSang/dots/internal/dotfile"
	"github.com/spf13/cobra"
)

func TestConfigArgsFunction(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "dots_main_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Custom home
	t.Setenv("HOME", tempDir)

	cfg := &config.Config{
		RepoPath: tempDir,
		Dotfiles: []config.DotfileSpec{
			{Name: "nvim", RepoPath: "nvim", SystemPath: "~/.config/nvim"},
			{Name: "tmux", RepoPath: "tmux", SystemPath: "~/.tmux.conf"},
		},
	}
	_ = config.Save(cfg)

	cmd := &cobra.Command{}
	names, directive := configArgsFunction(cmd, nil, "")
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("expected NoFileComp directive, got %v", directive)
	}
	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %d", len(names))
	}
	if names[0] != "nvim" || names[1] != "tmux" {
		t.Errorf("unexpected names: %v", names)
	}
}

func TestRunDirectSetup(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "dots_setup_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Setenv("HOME", tempDir)

	repoDir := filepath.Join(tempDir, "dotfiles")
	sysDir := filepath.Join(tempDir, ".config")
	_ = os.MkdirAll(repoDir, 0755)
	_ = os.MkdirAll(sysDir, 0755)

	repoNvim := filepath.Join(repoDir, "nvim")
	_ = os.WriteFile(repoNvim, []byte("nvim config"), 0644)
	sysNvim := filepath.Join(sysDir, "nvim")

	cfg := &config.Config{
		RepoPath: repoDir,
		Dotfiles: []config.DotfileSpec{
			{Name: "nvim", RepoPath: "nvim", SystemPath: sysNvim},
		},
	}
	_ = config.Save(cfg)

	// Test unknown dotfile
	err = runDirectSetup("ghostty")
	if err == nil {
		t.Errorf("expected error for unknown dotfile")
	}

	// Test successful setup
	err = runDirectSetup("nvim")
	if err != nil {
		t.Fatalf("runDirectSetup failed: %v", err)
	}

	// Verify symlink was created
	fi, err := os.Lstat(sysNvim)
	if err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("expected symlink at %s", sysNvim)
	}
}

func TestRunDirectRemove_Flags(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "dots_remove_flags_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Setenv("HOME", tempDir)

	repoDir := filepath.Join(tempDir, "dotfiles")
	sysDir := filepath.Join(tempDir, ".config")
	_ = os.MkdirAll(repoDir, 0755)
	_ = os.MkdirAll(sysDir, 0755)

	repoFish := filepath.Join(repoDir, "fish")
	_ = os.WriteFile(repoFish, []byte("fish config"), 0644)
	sysFish := filepath.Join(sysDir, "fish")
	_ = os.Symlink(repoFish, sysFish)

	cfg := &config.Config{
		RepoPath: repoDir,
		Dotfiles: []config.DotfileSpec{
			{Name: "fish", RepoPath: "fish", SystemPath: sysFish},
		},
	}
	_ = config.Save(cfg)

	// Test symlink only flag removal
	err = runDirectRemove("fish", removeFlags{symlinks: true})
	if err != nil {
		t.Fatalf("runDirectRemove symlinks failed: %v", err)
	}

	// Verify symlink deleted
	if _, err := os.Lstat(sysFish); !os.IsNotExist(err) {
		t.Errorf("expected sysFish symlink to be deleted")
	}
	// Verify repo file still intact
	if _, err := os.Stat(repoFish); err != nil {
		t.Errorf("expected repoFish to remain intact")
	}

	// Now recreate symlink and test --all --force removal
	_ = os.Symlink(repoFish, sysFish)
	err = runDirectRemove("fish", removeFlags{all: true, force: true})
	if err != nil {
		t.Fatalf("runDirectRemove all failed: %v", err)
	}

	// Verify both deleted
	if _, err := os.Lstat(sysFish); !os.IsNotExist(err) {
		t.Errorf("expected sysFish to be deleted")
	}
	if _, err := os.Stat(repoFish); !os.IsNotExist(err) {
		t.Errorf("expected repoFish to be deleted")
	}

	// Verify config updated
	loadedCfg, _ := config.Load()
	entries := dotfile.LoadEntries(loadedCfg)
	if len(entries) != 0 {
		t.Errorf("expected 0 entries remaining, got %d", len(entries))
	}
}
