package dotfile

import (
	"os"
	"path/filepath"
	"testing"
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
