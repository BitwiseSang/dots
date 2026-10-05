package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCollapsePath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("skipping test without home dir")
	}

	testPath := filepath.Join(home, "dotfiles")
	collapsed := CollapsePath(testPath)
	if collapsed != "~/dotfiles" {
		t.Errorf("expected ~/dotfiles, got %s", collapsed)
	}

	otherPath := "/var/log/syslog"
	if CollapsePath(otherPath) != otherPath {
		t.Errorf("expected untouched path, got %s", CollapsePath(otherPath))
	}
}

func TestNormalizeRepoPath(t *testing.T) {
	// Remote shorthand
	if norm := NormalizeRepoPath("bitwisesang/dotfiles"); norm != "~/dotfiles" {
		t.Errorf("expected ~/dotfiles, got %s", norm)
	}

	// Full git url
	if norm := NormalizeRepoPath("https://github.com/torvalds/linux.git"); norm != "~/linux" {
		t.Errorf("expected ~/linux, got %s", norm)
	}

	// Empty
	if norm := NormalizeRepoPath(""); norm != "" {
		t.Errorf("expected empty string, got %s", norm)
	}

	// Normal path
	if norm := NormalizeRepoPath("~/Documents/dotfiles"); norm != "~/Documents/dotfiles" {
		t.Errorf("expected ~/Documents/dotfiles, got %s", norm)
	}
}

func TestIsRemoteRepoInput(t *testing.T) {
	if !IsRemoteRepoInput("bitwisesang/dotfiles") {
		t.Errorf("expected true for shorthand")
	}
	if !IsRemoteRepoInput("https://github.com/foo/bar.git") {
		t.Errorf("expected true for https url")
	}
	if !IsRemoteRepoInput("git@github.com:foo/bar.git") {
		t.Errorf("expected true for git@ url")
	}
	if IsRemoteRepoInput("~/dotfiles") {
		t.Errorf("expected false for local ~/dotfiles")
	}
	if IsRemoteRepoInput("/home/user/dotfiles") {
		t.Errorf("expected false for absolute path")
	}
}

func TestAddDotfile(t *testing.T) {
	cfg := &Config{}
	spec1 := DotfileSpec{Name: "nvim", RepoPath: "nvim", SystemPath: "~/.config/nvim"}
	if !cfg.AddDotfile(spec1) {
		t.Errorf("expected true when adding new dotfile")
	}
	if len(cfg.Dotfiles) != 1 {
		t.Errorf("expected 1 dotfile, got %d", len(cfg.Dotfiles))
	}

	spec1Updated := DotfileSpec{Name: "nvim", RepoPath: "nvim", SystemPath: "~/.config/custom_nvim"}
	if cfg.AddDotfile(spec1Updated) {
		t.Errorf("expected false when updating existing dotfile")
	}
	if len(cfg.Dotfiles) != 1 {
		t.Errorf("expected still 1 dotfile, got %d", len(cfg.Dotfiles))
	}
	if cfg.Dotfiles[0].SystemPath != "~/.config/custom_nvim" {
		t.Errorf("expected updated system path")
	}
}

func TestRemoveDotfile(t *testing.T) {
	cfg := &Config{
		Dotfiles: []DotfileSpec{
			{Name: "nvim", RepoPath: "nvim", SystemPath: "~/.config/nvim"},
			{Name: "fish", RepoPath: "fish", SystemPath: "~/.config/fish"},
			{Name: "tmux", RepoPath: "tmux", SystemPath: "~/.tmux.conf"},
		},
	}

	// Remove middle element (case-insensitive)
	if !cfg.RemoveDotfile("FISH") {
		t.Errorf("expected true when removing existing dotfile")
	}
	if len(cfg.Dotfiles) != 2 {
		t.Fatalf("expected 2 dotfiles left, got %d", len(cfg.Dotfiles))
	}
	if cfg.Dotfiles[0].Name != "nvim" || cfg.Dotfiles[1].Name != "tmux" {
		t.Errorf("unexpected dotfiles remaining: %+v", cfg.Dotfiles)
	}

	// Remove non-existent
	if cfg.RemoveDotfile("ghostty") {
		t.Errorf("expected false when removing non-existent dotfile")
	}
	if len(cfg.Dotfiles) != 2 {
		t.Fatalf("expected 2 dotfiles, got %d", len(cfg.Dotfiles))
	}

	// Remove remaining
	if !cfg.RemoveDotfile("nvim") || !cfg.RemoveDotfile("tmux") {
		t.Errorf("failed to remove remaining dotfiles")
	}
	if len(cfg.Dotfiles) != 0 {
		t.Errorf("expected 0 dotfiles, got %d", len(cfg.Dotfiles))
	}
}

func TestSortDotfiles(t *testing.T) {
	cfg := &Config{
		Dotfiles: []DotfileSpec{
			{Name: "zsh"},
			{Name: "alacritty"},
			{Name: "bash"},
		},
	}
	cfg.SortDotfiles()
	if cfg.Dotfiles[0].Name != "alacritty" || cfg.Dotfiles[1].Name != "bash" || cfg.Dotfiles[2].Name != "zsh" {
		t.Errorf("expected alphabetical order, got %+v", cfg.Dotfiles)
	}

	// Adding dotfile automatically maintains sort order
	cfg.AddDotfile(DotfileSpec{Name: "aria2"})
	if cfg.Dotfiles[1].Name != "aria2" {
		t.Errorf("expected aria2 to be sorted at index 1, got %+v", cfg.Dotfiles)
	}
}

func TestLoad_NotInitialized(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	cfg, err := Load()
	if err == nil {
		t.Fatalf("expected ErrNotInitialized, got nil error with cfg: %+v", cfg)
	}
	if err != ErrNotInitialized {
		t.Errorf("expected ErrNotInitialized, got %v", err)
	}
}

func TestDefaultConfig_Empty(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.RepoPath != "" {
		t.Errorf("expected empty RepoPath, got %s", cfg.RepoPath)
	}
	if len(cfg.Dotfiles) != 0 {
		t.Errorf("expected 0 dotfiles in DefaultConfig, got %d", len(cfg.Dotfiles))
	}
}

func TestDefaultDotfiles_Empty(t *testing.T) {
	dots := DefaultDotfiles()
	if len(dots) != 0 {
		t.Errorf("expected 0 dotfiles in DefaultDotfiles, got %d", len(dots))
	}
}
