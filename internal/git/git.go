package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/BitwiseSang/dots/internal/config"
)

func IsRepo(path string) bool {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	cmd.Dir = config.ExpandPath(path)
	err := cmd.Run()
	return err == nil
}

func HasChanges(repoPath string) (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = config.ExpandPath(repoPath)
	out, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return len(strings.TrimSpace(string(out))) > 0, nil
}

// HasChangesForPath returns true if a specific subdirectory/file has unstaged, staged, or untracked changes.
func HasChangesForPath(repoPath, subPath string) (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain", "-u", "--", subPath)
	cmd.Dir = config.ExpandPath(repoPath)
	out, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return len(strings.TrimSpace(string(out))) > 0, nil
}

// DiffPath returns the unified diff for a path within the repository, including untracked files.
func DiffPath(repoPath, subPath string) (string, error) {
	absRepo := config.ExpandPath(repoPath)
	cmd := exec.Command("git", "diff", "HEAD", "--", subPath)
	cmd.Dir = absRepo
	out, err := cmd.Output()
	if err != nil {
		cmd = exec.Command("git", "diff", "--", subPath)
		cmd.Dir = absRepo
		out, err = cmd.Output()
		if err != nil {
			return "", err
		}
	}

	diffStr := strings.TrimSpace(string(out))

	statusCmd := exec.Command("git", "status", "--porcelain", "-u", "--", subPath)
	statusCmd.Dir = absRepo
	statusOut, _ := statusCmd.Output()

	var untrackedDiffs []string
	lines := strings.Split(string(statusOut), "\n")
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if strings.HasPrefix(l, "?? ") {
			relFile := strings.TrimSpace(strings.TrimPrefix(l, "?? "))
			fullFile := filepath.Join(repoPath, relFile)
			fi, err := os.Stat(fullFile)
			if err == nil && !fi.IsDir() {
				diffCmd := exec.Command("git", "diff", "--no-index", "/dev/null", relFile)
				diffCmd.Dir = repoPath
				uOut, _ := diffCmd.Output()
				if len(uOut) > 0 {
					untrackedDiffs = append(untrackedDiffs, strings.TrimSpace(string(uOut)))
				} else {
					untrackedDiffs = append(untrackedDiffs, "+ [untracked] "+relFile)
				}
			} else {
				untrackedDiffs = append(untrackedDiffs, "+ [untracked] "+relFile)
			}
		}
	}

	if len(untrackedDiffs) > 0 {
		if diffStr != "" {
			diffStr += "\n"
		}
		diffStr += strings.Join(untrackedDiffs, "\n")
	}

	return diffStr, nil
}

// Add stages all changed files in the repository.
func Add(repoPath string) error {
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = config.ExpandPath(repoPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg != "" {
			return fmt.Errorf("git add failed: %s", msg)
		}
		return err
	}
	return nil
}

// AddPaths stages specific files or directories in the repository.
func AddPaths(repoPath string, paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	args := append([]string{"add", "--"}, paths...)
	cmd := exec.Command("git", args...)
	cmd.Dir = config.ExpandPath(repoPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg != "" {
			return fmt.Errorf("git add failed: %s", msg)
		}
		return err
	}
	return nil
}

func Commit(repoPath, message string) error {
	cmd := exec.Command("git", "commit", "-m", message)
	cmd.Dir = config.ExpandPath(repoPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg != "" {
			return fmt.Errorf("git commit failed: %s", msg)
		}
		return err
	}
	return nil
}

func Push(repoPath string) error {
	cmd := exec.Command("git", "push")
	cmd.Dir = config.ExpandPath(repoPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg != "" {
			return fmt.Errorf("git push failed: %s", msg)
		}
		return err
	}
	return nil
}

func CommitMessage(prefix string) string {
	return CommitMessageWithEntries(prefix, nil)
}

func CommitMessageWithEntries(prefix string, names []string) string {
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown"
	}
	if prefix == "" {
		prefix = "Config backup"
	}
	timestamp := time.Now().Format("2006-01-02 at 15:04:05")
	if len(names) > 0 {
		return fmt.Sprintf("%s (%s) on %s from %s", prefix, strings.Join(names, ", "), timestamp, hostname)
	}
	return fmt.Sprintf("%s on %s from %s", prefix, timestamp, hostname)
}
