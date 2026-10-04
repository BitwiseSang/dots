package git

import (
	"strings"
	"testing"
)

func TestResolveRepoInput(t *testing.T) {
	// Case 1: GitHub shorthand: "username/repository"
	isRemote, url, local := ResolveRepoInput("BitwiseSang/dotfiles")
	if !isRemote {
		t.Errorf("expected isRemote to be true for shorthand")
	}
	if url != "https://github.com/BitwiseSang/dotfiles.git" {
		t.Errorf("expected https url, got %s", url)
	}
	if !strings.HasSuffix(local, "dotfiles") {
		t.Errorf("expected local path to end with dotfiles, got %s", local)
	}

	// Case 2: Full Git URL
	isRemote, url, _ = ResolveRepoInput("https://github.com/torvalds/linux.git")
	if !isRemote {
		t.Errorf("expected isRemote to be true for https URL")
	}
	if url != "https://github.com/torvalds/linux.git" {
		t.Errorf("expected original url, got %s", url)
	}

	// Case 3: Local path
	isRemote, _, local = ResolveRepoInput("~/Documents/dotfiles")
	if isRemote {
		t.Errorf("expected isRemote to be false for local path")
	}
	if strings.HasPrefix(local, "~") {
		t.Errorf("expected local path to be expanded, got %s", local)
	}
}
