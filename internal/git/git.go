package git

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

func IsRepo(path string) bool {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	cmd.Dir = path
	err := cmd.Run()
	return err == nil
}

func HasChanges(repoPath string) (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return len(strings.TrimSpace(string(out))) > 0, nil
}

// HasChangesForPath returns true if a specific subdirectory/file has unstaged, staged, or untracked changes.
func HasChangesForPath(repoPath, subPath string) (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain", "--", subPath)
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return len(strings.TrimSpace(string(out))) > 0, nil
}

// DiffPath returns the unified diff for a path within the repository, including untracked files.
func DiffPath(repoPath, subPath string) (string, error) {
	cmd := exec.Command("git", "diff", "HEAD", "--", subPath)
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		cmd = exec.Command("git", "diff", "--", subPath)
		cmd.Dir = repoPath
		out, err = cmd.Output()
		if err != nil {
			return "", err
		}
	}

	diffStr := strings.TrimSpace(string(out))

	statusCmd := exec.Command("git", "status", "--porcelain", "-u", "--", subPath)
	statusCmd.Dir = repoPath
	statusOut, _ := statusCmd.Output()

	var untracked []string
	lines := strings.Split(string(statusOut), "\n")
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if strings.HasPrefix(l, "?? ") {
			untracked = append(untracked, "+ [untracked] "+strings.TrimPrefix(l, "?? "))
		}
	}

	if len(untracked) > 0 {
		if diffStr != "" {
			diffStr += "\n"
		}
		diffStr += strings.Join(untracked, "\n")
	}

	return diffStr, nil
}

func Add(repoPath string) error {
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = repoPath
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
	cmd.Dir = repoPath
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
	cmd.Dir = repoPath
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
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown"
	}
	if prefix == "" {
		prefix = "Config backup"
	}
	timestamp := time.Now().Format("2006-01-02 at 15:04:05")
	return fmt.Sprintf("%s on %s from %s", prefix, timestamp, hostname)
}
