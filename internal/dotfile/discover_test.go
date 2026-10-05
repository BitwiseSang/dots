package dotfile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/BitwiseSang/dots/internal/config"
)

func TestDiscoverSystemConfigs_DynamicDiscovery(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	customConfigDir := filepath.Join(tempHome, ".custom_config")
	t.Setenv("XDG_CONFIG_HOME", customConfigDir)

	// Create custom XDG config entries
	_ = os.MkdirAll(filepath.Join(customConfigDir, "myapp"), 0755)
	_ = os.WriteFile(filepath.Join(customConfigDir, "starship.toml"), []byte("# starship"), 0644)
	// System caches that should be excluded
	_ = os.MkdirAll(filepath.Join(customConfigDir, "pulse"), 0755)
	_ = os.MkdirAll(filepath.Join(customConfigDir, "dconf"), 0755)
	_ = os.MkdirAll(filepath.Join(customConfigDir, ".hidden"), 0755)

	// Create root home dotfiles
	_ = os.WriteFile(filepath.Join(tempHome, ".bashrc"), []byte("# bashrc"), 0644)
	_ = os.WriteFile(filepath.Join(tempHome, ".tmux.conf"), []byte("# tmux"), 0644)

	// Config already manages "myapp"
	cfg := &config.Config{
		Dotfiles: []config.DotfileSpec{
			{Name: "myapp", SystemPath: filepath.Join(customConfigDir, "myapp")},
		},
	}

	discovered := DiscoverSystemConfigs(cfg)

	// We expect: bashrc, starship, tmux (unmanaged), and myapp (managed)
	if len(discovered) != 4 {
		t.Fatalf("expected 4 discovered items, got %d: %+v", len(discovered), discovered)
	}

	// Verify unmanaged items come first
	for i := 0; i < 3; i++ {
		if discovered[i].AlreadyManaged {
			t.Errorf("expected discovered[%d] to be unmanaged, got managed: %+v", i, discovered[i])
		}
	}

	// The managed item (myapp) should be last
	if !discovered[3].AlreadyManaged || discovered[3].Name != "myapp" {
		t.Errorf("expected last discovered item to be managed 'myapp', got %+v", discovered[3])
	}

	// Verify items found
	names := make(map[string]bool)
	for _, d := range discovered {
		names[d.Name] = true
	}
	if !names["bashrc"] || !names["starship"] || !names["tmux"] || !names["myapp"] {
		t.Errorf("missing expected discovered item names, got: %+v", names)
	}
}
