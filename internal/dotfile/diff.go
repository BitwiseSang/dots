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

	// 1. If entry is linked, check git diff in the repository
	if entry.IsLinked() {
		if git.IsRepo(entry.repoRoot) {
			return git.DiffPath(entry.repoRoot, entry.RepoPath)
		}
		return "", nil
	}

	// 2. Unlinked entry: compare repoPath to sysPath so '+' shows additions in system
	if _, err := os.Stat(sysPath); os.IsNotExist(err) {
		return "", fmt.Errorf("system path does not exist: %s", sysPath)
	}
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		return fmt.Sprintf("+ [new in system] %s", sysPath), nil
	}

	args := []string{"-u"}
	if entry.IsDir {
		args = []string{"-ruN"}
	}
	args = append(args, repoPath, sysPath)

	cmd := exec.Command("diff", args...)
	out, err := cmd.Output()

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if exitError.ExitCode() == 1 {
				return string(out), nil
			}
		}
		return "", fmt.Errorf("diff command failed: %v", err)
	}
	return string(out), nil
}
