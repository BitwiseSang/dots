package components

import (
	"strings"
	"testing"

	"github.com/BitwiseSang/dots/internal/version"
	"github.com/charmbracelet/lipgloss"
)

func TestHeaderHeightHelper(t *testing.T) {
	tests := []struct {
		width    int
		height   int
		expected int
	}{
		{width: 80, height: 40, expected: 13},
		{width: 80, height: 28, expected: 13},
		{width: 80, height: 27, expected: 5},
		{width: 80, height: 20, expected: 5},
		{width: 80, height: 19, expected: 3},
		{width: 80, height: 12, expected: 3},
		{width: 80, height: 0, expected: 13},
		// Narrow width forces compact 5 lines even if tall height
		{width: 40, height: 40, expected: 5},
		{width: 35, height: 28, expected: 5},
		{width: 35, height: 16, expected: 3},
	}

	for _, tc := range tests {
		actual := HeaderHeight(tc.width, tc.height)
		if actual != tc.expected {
			t.Errorf("HeaderHeight(%d, %d) = %d; expected %d", tc.width, tc.height, actual, tc.expected)
		}
	}
}

func TestHeaderPresenceAtAllSizes(t *testing.T) {
	// Heights representing full (>=28), compact (20-27), and ultra-compact (<20)
	heights := []int{40, 28, 25, 20, 16, 12}
	widths := []int{120, 80, 50, 35}

	for _, h := range heights {
		for _, w := range widths {
			rendered := Header(w, h, 0, "~/my-dotfiles")

			// 1. Top bar with dots brand and dynamic version badge must be present
			if !strings.Contains(rendered, "dots™") {
				t.Errorf("expected 'dots™' in header at %dx%d", w, h)
			}
			if !strings.Contains(rendered, "v"+version.Version) {
				t.Errorf("expected version 'v%s' in header at %dx%d", version.Version, w, h)
			}

			// 2. Directory line (breadcrumb) must be present
			if !strings.Contains(rendered, "my-dotfiles") {
				t.Errorf("expected repo path in header at %dx%d", w, h)
			}

			// 3. Three separator dots must be present
			if !strings.Contains(rendered, "•") && !strings.Contains(rendered, "·") {
				t.Errorf("expected separator dots in header at %dx%d", w, h)
			}

			// 4. Exact rendered height must match HeaderHeight(w, h)
			renderedHeight := lipgloss.Height(rendered)
			expectedHeight := HeaderHeight(w, h)
			if renderedHeight != expectedHeight {
				t.Errorf("Header height mismatch at %dx%d: rendered %d lines, expected %d lines",
					w, h, renderedHeight, expectedHeight)
			}
		}
	}
}
