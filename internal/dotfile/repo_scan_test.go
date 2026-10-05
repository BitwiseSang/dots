package dotfile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/BitwiseSang/dots/internal/config"
)

func TestInspectRepositoryWithConfigFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create dots/config.toml inside repo
	dotsDir := filepath.Join(tmpDir, "dots")
	_ = os.MkdirAll(dotsDir, 0755)

	tomlContent := `
editor = "nvim"
repo_path = "~/dotfiles"

[[dotfiles]]
name = "alacritty"
repo_path = "alacritty"
system_path = "~/.config/alacritty"
method = "rsync"
is_dir = true
`
	_ = os.WriteFile(filepath.Join(dotsDir, "config.toml"), []byte(tomlContent), 0644)

	specs, cfg, err := InspectRepository(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(specs) != 1 || specs[0].Name != "alacritty" {
		t.Errorf("expected 1 spec 'alacritty', got %v", specs)
	}
	if cfg.Editor != "nvim" {
		t.Errorf("expected editor 'nvim', got '%s'", cfg.Editor)
	}
}

func TestInspectRepositoryWithoutConfigFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create directories and files
	_ = os.Mkdir(filepath.Join(tmpDir, "nvim"), 0755)
	_ = os.Mkdir(filepath.Join(tmpDir, "fish"), 0755)
	_ = os.WriteFile(filepath.Join(tmpDir, "tmux.conf"), []byte("test"), 0644)
	_ = os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("readme"), 0644) // Should be skipped

	specs, _, err := InspectRepository(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(specs) != 3 {
		t.Fatalf("expected 3 specs (nvim, fish, tmux.conf), got %d", len(specs))
	}

	names := make(map[string]bool)
	for _, s := range specs {
		names[s.Name] = true
		if s.Name == "README" {
			t.Errorf("README should have been ignored!")
		}
	}
	if !names["nvim"] || !names["fish"] || !names["tmux"] {
		t.Errorf("expected nvim, fish, and tmux, got %v", names)
	}
}

func TestRefreshDatabase_NewFolderAndFile(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	repoDir := filepath.Join(tempHome, "dotfiles")
	_ = os.MkdirAll(filepath.Join(repoDir, "nvim"), 0755)
	_ = os.WriteFile(filepath.Join(repoDir, "tmux.conf"), []byte("tmux config"), 0644)
	_ = os.MkdirAll(filepath.Join(repoDir, "alacritty"), 0755)

	// Non-config files that MUST be ignored
	_ = os.WriteFile(filepath.Join(repoDir, "setup.sh"), []byte("#!/bin/sh"), 0755)
	_ = os.WriteFile(filepath.Join(repoDir, "backup.sh"), []byte("#!/bin/sh"), 0755)
	_ = os.WriteFile(filepath.Join(repoDir, "README.md"), []byte("# Docs"), 0644)
	_ = os.WriteFile(filepath.Join(repoDir, "Makefile"), []byte("all:"), 0644)
	_ = os.MkdirAll(filepath.Join(repoDir, ".git"), 0755)
	_ = os.MkdirAll(filepath.Join(repoDir, "dots"), 0755)

	// cfg currently only has "nvim"
	cfg := &config.Config{
		Editor:   "nano",
		RepoPath: repoDir,
		Dotfiles: []config.DotfileSpec{
			{Name: "nvim", RepoPath: "nvim", SystemPath: "~/.config/nvim", Method: "rsync", IsDir: true},
		},
	}
	_ = config.Save(cfg)

	// Run RefreshDatabase
	newlyAdded, updated := RefreshDatabase(cfg)
	if !updated {
		t.Fatalf("expected RefreshDatabase to return updated=true")
	}

	// Should have discovered alacritty and tmux (not setup.sh, backup.sh, README, Makefile, .git, dots)
	if len(newlyAdded) != 2 {
		t.Fatalf("expected 2 newly added configs, got %d: %+v", len(newlyAdded), newlyAdded)
	}

	newNames := map[string]bool{newlyAdded[0].Name: true, newlyAdded[1].Name: true}
	if !newNames["alacritty"] || !newNames["tmux"] {
		t.Errorf("expected alacritty and tmux, got %+v", newlyAdded)
	}

	// Verify all dotfiles in cfg are sorted alphabetically: alacritty, nvim, tmux
	if len(cfg.Dotfiles) != 3 {
		t.Fatalf("expected 3 total dotfiles, got %d", len(cfg.Dotfiles))
	}
	if cfg.Dotfiles[0].Name != "alacritty" || cfg.Dotfiles[1].Name != "nvim" || cfg.Dotfiles[2].Name != "tmux" {
		t.Errorf("expected sorted order [alacritty, nvim, tmux], got [%s, %s, %s]",
			cfg.Dotfiles[0].Name, cfg.Dotfiles[1].Name, cfg.Dotfiles[2].Name)
	}

	// Verify saved to disk
	loadedCfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to reload config from disk: %v", err)
	}
	if len(loadedCfg.Dotfiles) != 3 {
		t.Fatalf("expected 3 dotfiles loaded from disk, got %d", len(loadedCfg.Dotfiles))
	}

	// Run RefreshDatabase a second time: should detect no changes
	newlyAdded2, updated2 := RefreshDatabase(cfg)
	if updated2 || len(newlyAdded2) != 0 {
		t.Errorf("expected no changes on second run, got updated=%v, added=%+v", updated2, newlyAdded2)
	}
}

func TestRefreshDatabase_FromRepoConfigFile(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	repoDir := filepath.Join(tempHome, "my-dots")
	_ = os.MkdirAll(filepath.Join(repoDir, "dots"), 0755)

	repoToml := `
[[dotfiles]]
name = "kitty"
repo_path = "kitty"
system_path = "~/.config/kitty"
method = "rsync"
is_dir = true
`
	_ = os.WriteFile(filepath.Join(repoDir, "dots", "config.toml"), []byte(repoToml), 0644)

	cfg := &config.Config{
		RepoPath: repoDir,
		Dotfiles: []config.DotfileSpec{
			{Name: "zsh", RepoPath: ".zshrc", SystemPath: "~/.zshrc"},
		},
	}
	_ = config.Save(cfg)

	newlyAdded, updated := RefreshDatabase(cfg)
	if !updated || len(newlyAdded) != 1 || newlyAdded[0].Name != "kitty" {
		t.Fatalf("expected kitty to be added from repo config.toml, got %+v", newlyAdded)
	}
}

