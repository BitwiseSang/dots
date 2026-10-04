package components

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestStatusBarMaxHeight(t *testing.T) {
	longHint := "very long key description that will certainly exceed eighty columns on any small terminal display • another long description that goes on"
	bar := StatusBar("Browse", longHint, 50)
	h := lipgloss.Height(bar)
	if h != 1 {
		t.Errorf("Expected status bar height to be strictly 1, got %d", h)
	}
}

func TestPlacePinnedStatusBar(t *testing.T) {
	for targetHeight := 10; targetHeight <= 35; targetHeight++ {
		topBlock := "Line 1\nLine 2\nLine 3"
		statusBar := "Status"

		res := PlacePinnedStatusBar(topBlock, statusBar, targetHeight)
		h := lipgloss.Height(res)

		expected := targetHeight
		if targetHeight < 4 {
			expected = 4
		}
		if h != expected {
			t.Errorf("For target %d: expected height %d, got %d", targetHeight, expected, h)
		}
	}
}

func TestRenderShortcutsCompact(t *testing.T) {
	rendered := RenderShortcuts("enter open • q quit")
	if strings.Contains(rendered, "  •  ") {
		t.Errorf("Expected compact ' • ' bullet spacing, got extra spaces in %q", rendered)
	}
}
