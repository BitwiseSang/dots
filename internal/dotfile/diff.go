package dotfile

import (
	"fmt"
	"os/exec"
)

func Diff(entry Entry) (string, error) {
	sysPath := entry.ResolveSystemPath()
	repoPath := entry.AbsRepoPath()

	args := []string{"-u"}
	if entry.IsDir {
		args = []string{"-ruN"}
	}
	args = append(args, sysPath, repoPath)

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
