package dotfile

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/BitwiseSang/dots/internal/git"
)

func Diff(entry Entry) (string, error) {
	sysPath := entry.ResolveSystemPath()
	repoPath := entry.AbsRepoPath()

	// 1. If system path does not exist
	if _, err := os.Stat(sysPath); os.IsNotExist(err) {
		return "", fmt.Errorf("system path does not exist: %s", sysPath)
	}

	// 2. If repo path does not exist yet (brand new config to back up)
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		if entry.IsDir {
			emptyDir, err := os.MkdirTemp("", "dots_diff_*")
			if err == nil {
				defer os.RemoveAll(emptyDir)
				cmd := exec.Command("diff", "-ruN", emptyDir, sysPath)
				out, _ := cmd.Output()
				if len(out) > 0 {
					return string(out), nil
				}
			}
		} else {
			cmd := exec.Command("diff", "-u", "/dev/null", sysPath)
			out, _ := cmd.Output()
			if len(out) > 0 {
				return string(out), nil
			}
		}
		return fmt.Sprintf("+ [new in system] %s", sysPath), nil
	}

	// 3. If entry is linked, check git diff in the repository
	if entry.IsLinked() {
		if git.IsRepo(entry.repoRoot) {
			return git.DiffPath(entry.repoRoot, entry.RepoPath)
		}
		return "", nil
	}

	// 4. Unlinked entry: compare repoPath to sysPath so '+' shows additions in system
	var fsDiff string
	args := []string{"-u"}
	if entry.IsDir {
		args = []string{"-ruN"}
	}
	args = append(args, repoPath, sysPath)

	cmd := exec.Command("diff", args...)
	out, err := cmd.Output()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 1 {
			fsDiff = string(out)
		} else if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() > 1 {
			return "", fmt.Errorf("diff command error (exit code %d): %s", exitError.ExitCode(), string(exitError.Stderr))
		} else {
			return "", fmt.Errorf("diff command failed: %w", err)
		}
	}

	// Also check if the repository path has git changes or untracked files
	var gitDiff string
	if git.IsRepo(entry.repoRoot) {
		gitDiff, _ = git.DiffPath(entry.repoRoot, entry.RepoPath)
	}

	if fsDiff != "" && gitDiff != "" {
		return fsDiff + "\n" + gitDiff, nil
	}
	if fsDiff != "" {
		return fsDiff, nil
	}
	return gitDiff, nil
}
