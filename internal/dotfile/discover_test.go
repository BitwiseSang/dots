package dotfile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/BitwiseSang/dots/internal/config"
)

func TestDiscoverSystemConfigs(t *testing.T) {
	tmpDir := t.TempDir()
	// Create mock files
	nvimDir := filepath.Join(tmpDir, "nvim")
	_ = os.Mkdir(nvimDir, 0755)

	cfg := &config.Config{
		Dotfiles: []config.DotfileSpec{
			{Name: "nvim", SystemPath: nvimDir},
		},
	}

	discovered := DiscoverSystemConfigs(cfg)
	// Function returns discovered items without crashing
	t.Logf("Discovered %d items", len(discovered))
	for _, d := range discovered {
		if d.AlreadyManaged {
			t.Logf("Managed item: %s (%s)", d.Name, d.SystemPath)
		}
	}
}
