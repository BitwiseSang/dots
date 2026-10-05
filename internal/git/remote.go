package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/BitwiseSang/dots/internal/config"
)

// ResolveRepoInput parses user input which can be:
// 1. GitHub shorthand: "username/repo" -> returns true, "https://github.com/username/repo.git", localPath
// 2. Full Git URL: "https://github.com/...", "git@github.com:..." -> returns true, url, localPath
// 3. Local filesystem path: "~/dotfiles", "/home/..." -> returns false, "", expandedPath
func ResolveRepoInput(input string) (isRemote bool, remoteURL string, localPath string) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return false, "", ""
	}

	// Full git URL (https://, http://, git@, ssh://)
	if strings.HasPrefix(trimmed, "https://") || strings.HasPrefix(trimmed, "http://") ||
		strings.HasPrefix(trimmed, "git@") || strings.HasPrefix(trimmed, "ssh://") {
		clean := strings.TrimSuffix(trimmed, ".git")
		clean = strings.ReplaceAll(clean, ":", "/")
		parts := strings.Split(clean, "/")
		base := parts[len(parts)-1]
		if base == "" {
			base = "dotfiles"
		}
		return true, trimmed, config.ExpandPath(filepath.Join("~", base))
	}

	// GitHub shorthand: <username>/<repository> (e.g. "BitwiseSang/dotfiles")
	if !strings.HasPrefix(trimmed, "~") && !strings.HasPrefix(trimmed, "/") && !strings.HasPrefix(trimmed, ".") {
		parts := strings.Split(trimmed, "/")
		if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
			remoteURL = fmt.Sprintf("https://github.com/%s/%s.git", parts[0], parts[1])
			return true, remoteURL, config.ExpandPath(filepath.Join("~", parts[1]))
		}
	}

	// Local filesystem path
	return false, "", config.ExpandPath(trimmed)
}

// Clone clones a git repository from remoteURL into destPath.
// If gh (GitHub CLI) is available and remoteURL is shorthand, it can use gh; otherwise git clone.
func Clone(remoteURL, destPath string) error {
	destPath = config.ExpandPath(destPath)
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}
	cmd := exec.Command("git", "clone", remoteURL, destPath)
	return cmd.Run()
}

// InitRepo initializes a new git repository in path.
func InitRepo(path string) error {
	path = config.ExpandPath(path)
	if err := os.MkdirAll(path, 0755); err != nil {
		return err
	}
	cmd := exec.Command("git", "init")
	cmd.Dir = path
	return cmd.Run()
}
