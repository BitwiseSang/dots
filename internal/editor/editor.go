package editor

import (
	"os"
	"os/exec"
)

// Resolve determines which editor to use.
// Priority: config editor > $EDITOR env var > "nvim" fallback
func Resolve(configEditor string) string {
	if configEditor != "" {
		return configEditor
	}
	if ed := os.Getenv("EDITOR"); ed != "" {
		return ed
	}
	return "nvim"
}

// Open opens the given path in the editor.
// It creates an exec.Cmd that replaces the current process (for TUI suspension).
// Returns the *exec.Cmd ready to run.
func Open(editor, path string) *exec.Cmd {
	cmd := exec.Command(editor, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}

// OpenAndWait opens the path in the editor and waits for it to exit.
// It sets up stdin/stdout/stderr properly for interactive use.
func OpenAndWait(editor, path string) error {
	cmd := Open(editor, path)
	return cmd.Run()
}
