package dotfile

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// movePath renames src to dst, falling back to copy+delete across filesystems.
func movePath(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	fi, err := os.Stat(src)
	if err != nil {
		return err
	}
	if fi.IsDir() {
		cmd := exec.Command("cp", "-a", src, dst)
		if err := cmd.Run(); err != nil {
			return err
		}
		return os.RemoveAll(src)
	}
	if err := copyFile(src, dst); err != nil {
		return err
	}
	return os.Remove(src)
}

type SetupResult struct {
	Entry      Entry
	Err        error
	BackedUp   bool
	BackupPath string
}

func CreateBackupDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	timestamp := time.Now().Format("20060102_150405")
	backupDir := filepath.Join(home, fmt.Sprintf(".dotfile_backups_%s", timestamp))
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", err
	}
	return backupDir, nil
}

func Setup(entry Entry, backupDir string) (bool, string, error) {
	sysPath := entry.ResolveSystemPath()
	repoPath := entry.AbsRepoPath()

	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		return false, "", fmt.Errorf("repository file missing: %s", repoPath)
	}

	if err := os.MkdirAll(filepath.Dir(sysPath), 0755); err != nil {
		return false, "", err
	}

	var backedUp bool
	var backupPath string

	if info, err := os.Lstat(sysPath); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			target, _ := os.Readlink(sysPath)
			if filepath.Clean(target) == filepath.Clean(repoPath) {
				return false, "", nil 
			}
			os.Remove(sysPath)
		} else {
			if backupDir == "" {
				var err error
				backupDir, err = CreateBackupDir()
				if err != nil {
					return false, "", fmt.Errorf("failed to create backup dir: %v", err)
				}
			}
			home, _ := os.UserHomeDir()
			backupPath = filepath.Join(backupDir, entry.Name, filepath.Base(sysPath))
			if home != "" {
				if rel, err := filepath.Rel(home, sysPath); err == nil && !strings.HasPrefix(rel, "..") {
					backupPath = filepath.Join(backupDir, rel)
				}
			}
			if _, err := os.Lstat(backupPath); err == nil {
				backupPath = fmt.Sprintf("%s_%d", backupPath, time.Now().UnixNano())
			}
			if err := movePath(sysPath, backupPath); err != nil {
				return false, "", fmt.Errorf("failed to backup existing file: %v", err)
			}
			backedUp = true
		}
	}

	err := os.Symlink(repoPath, sysPath)
	return backedUp, backupPath, err
}

func SetupAll(entries []Entry, backupDir string) []SetupResult {
	var results []SetupResult
	for _, entry := range entries {
		backedUp, path, err := Setup(entry, backupDir)
		results = append(results, SetupResult{
			Entry:      entry,
			Err:        err,
			BackedUp:   backedUp,
			BackupPath: path,
		})
	}
	return results
}
