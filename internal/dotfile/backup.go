package dotfile

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

type BackupResult struct {
	Entry   Entry
	Err     error
	Skipped bool
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}

	info, err := os.Stat(src)
	if err == nil {
		os.Chmod(dst, info.Mode())
	}
	return nil
}

func Backup(entry Entry) error {
	sysPath := entry.ResolveSystemPath()
	repoPath := entry.AbsRepoPath()

	if entry.IsLinked() {
		// When linked, sysPath is a symlink pointing to repoPath.
		// The files already reside in repoPath, so filesystem sync is complete.
		return nil
	}

	if _, err := os.Stat(sysPath); os.IsNotExist(err) {
		return fmt.Errorf("system path does not exist: %s", sysPath)
	}

	if err := os.MkdirAll(filepath.Dir(repoPath), 0755); err != nil {
		return err
	}

	if entry.Method == SyncRsync {
		src := sysPath
		if entry.IsDir && src[len(src)-1] != '/' {
			src += "/"
		}
		dst := repoPath
		if entry.IsDir && dst[len(dst)-1] != '/' {
			dst += "/"
		}
		cmd := exec.Command("rsync", "-ac", "--delete", src, dst)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("rsync error: %w: %s", err, string(out))
		}
		return nil
	}

	return copyFile(sysPath, repoPath)
}

func BackupAll(entries []Entry) []BackupResult {
	var results []BackupResult
	for _, entry := range entries {
		status := entry.CheckStatus()
		if status == StatusMissing {
			results = append(results, BackupResult{Entry: entry, Skipped: true, Err: fmt.Errorf("system path missing")})
			continue
		}

		err := Backup(entry)
		results = append(results, BackupResult{Entry: entry, Err: err, Skipped: false})
	}
	return results
}
