package version

import (
	"strings"
	"testing"
)

func TestVersionNotEmpty(t *testing.T) {
	if Version == "" {
		t.Fatal("expected Version to be non-empty")
	}
	parts := strings.Split(Version, ".")
	if len(parts) < 2 {
		t.Fatalf("expected semantic version format (e.g. 0.1.0), got %s", Version)
	}
}
