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
	if norm := NormalizeRepoPath(""); norm != "~/dotfiles" {
		t.Errorf("expected ~/dotfiles, got %s", norm)
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
