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

func Add(repoPath string) error {
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = repoPath
	return cmd.Run()
}

func Commit(repoPath, message string) error {
	cmd := exec.Command("git", "commit", "-m", message)
	cmd.Dir = repoPath
	return cmd.Run()
}

func Push(repoPath string) error {
	cmd := exec.Command("git", "push")
	cmd.Dir = repoPath
	return cmd.Run()
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
