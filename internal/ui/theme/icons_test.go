package theme

import (
	"testing"
)

func TestFileIconStyled(t *testing.T) {
	tests := []struct {
		name     string
		isDir    bool
		wantIcon string
	}{
		{"nvim", true, IconNvim},
		{"fish", true, IconFish},
		{".config", true, IconDirConfig},
		{".git", true, IconDirGit},
		{"some_folder", true, IconDirModern},
		{"init.lua", false, IconLua},
		{"main.go", false, IconGo},
		{"config.fish", false, IconFish},
		{"Makefile", false, IconMakefile},
		{"Cargo.toml", false, IconRust},
		{"ghostty", false, IconGhostty},
		{"kitty", false, IconKitty},
	}

	for _, tc := range tests {
		icon, color := FileIconStyled(tc.name, tc.isDir)
		if icon != tc.wantIcon {
			t.Errorf("FileIconStyled(%q, %v) got icon %q, want %q", tc.name, tc.isDir, icon, tc.wantIcon)
		}
		if color == "" {
			t.Errorf("FileIconStyled(%q, %v) returned empty color", tc.name, tc.isDir)
		}
	}
}
