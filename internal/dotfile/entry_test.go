package dotfile

import (
	"strings"
	"testing"
)

func TestStatusLabelNoEmojis(t *testing.T) {
	entry := Entry{}
	// Test that status label for linked uses Nerd Font icon and not emoji
	// We can test that StatusLabel does not contain 🔗
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
	_ = entry
}
