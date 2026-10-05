package dotfile

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BitwiseSang/dots/internal/config"
)

func TestDiffAndBackup_Unlinked(t *testing.T) {
	tmpDir := t.TempDir()
	sysDir := filepath.Join(tmpDir, "sys_fish")
	repoDir := filepath.Join(tmpDir, "repo_fish")
	_ = os.MkdirAll(sysDir, 0755)
	_ = os.MkdirAll(repoDir, 0755)

	_ = os.WriteFile(filepath.Join(sysDir, "config.fish"), []byte("set -x VAR 1\n"), 0644)
	_ = os.WriteFile(filepath.Join(repoDir, "config.fish"), []byte("set -x VAR 0\n"), 0644)

	spec := config.DotfileSpec{
		Name:       "fish",
		RepoPath:   "repo_fish",
		SystemPath: sysDir,
		Method:     "rsync",
		IsDir:      true,
	}
	entry := NewEntry(spec, tmpDir)

	diff, err := Diff(entry)
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}
	if !strings.Contains(diff, "-set -x VAR 0") || !strings.Contains(diff, "+set -x VAR 1") {
		t.Errorf("expected diff to show changes, got: %s", diff)
	}

	// Backup should sync sys to repo
	if err := Backup(entry); err != nil {
		t.Fatalf("Backup failed: %v", err)
	}

	repoData, _ := os.ReadFile(filepath.Join(repoDir, "config.fish"))
	if string(repoData) != "set -x VAR 1\n" {
		t.Errorf("expected repo file to be updated by Backup, got %s", string(repoData))
	}

	if status := entry.CheckStatus(); status != StatusInSync {
		t.Errorf("expected StatusInSync after backup, got %v", status)
	}
}

func TestDiffAndBackup_Linked(t *testing.T) {
	repoDir := t.TempDir()

	// Init git repo
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = repoDir
		_ = cmd.Run()
	}
	run("init")
	run("config", "user.name", "Test")
	run("config", "user.email", "test@example.com")

	tmuxRepo := filepath.Join(repoDir, "tmux.conf")
	_ = os.WriteFile(tmuxRepo, []byte("set -g prefix C-a\n"), 0644)
	run("add", ".")
	run("commit", "-m", "init tmux")

	sysDir := t.TempDir()
	sysLink := filepath.Join(sysDir, ".tmux.conf")
	_ = os.Symlink(tmuxRepo, sysLink)

	spec := config.DotfileSpec{
		Name:       "tmux",
		RepoPath:   "tmux.conf",
		SystemPath: sysLink,
		Method:     "copy",
		IsDir:      false,
	}
	entry := NewEntry(spec, repoDir)

	// Initially clean
	d, err := Diff(entry)
	if err != nil {
		t.Fatalf("Diff error: %v", err)
	}
	if d != "" {
		t.Errorf("expected empty diff on clean linked repo, got: %s", d)
	}

	// Modify file
	_ = os.WriteFile(tmuxRepo, []byte("set -g prefix C-b\n"), 0644)

	// Diff should return git diff!
	d, err = Diff(entry)
	if err != nil {
		t.Fatalf("Diff error: %v", err)
	}
	if !strings.Contains(d, "-set -g prefix C-a") || !strings.Contains(d, "+set -g prefix C-b") {
		t.Errorf("expected git diff for modified linked file, got: %s", d)
	}

	// BackupAll should not error on linked files
	results := BackupAll([]Entry{entry})
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Err != nil {
		t.Errorf("expected no error backing up linked entry, got: %v", results[0].Err)
	}
	if results[0].Skipped {
		t.Errorf("expected linked entry to not be marked skipped")
	}
}

func TestDiff_NewConfig(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	_ = os.MkdirAll(repoDir, 0755)

	sysDir := filepath.Join(tmpDir, "sys")
	_ = os.MkdirAll(sysDir, 0755)

	// Single file new config
	sysFile := filepath.Join(sysDir, "config.toml")
	_ = os.WriteFile(sysFile, []byte("theme = 'dark'\n"), 0644)

	spec := config.DotfileSpec{
		Name:       "mytool",
		RepoPath:   "mytool.toml",
		SystemPath: sysFile,
		Method:     "copy",
		IsDir:      false,
	}
	entry := NewEntry(spec, repoDir)

	d, err := Diff(entry)
	if err != nil {
		t.Fatalf("Diff failed for new config: %v", err)
	}
	if !strings.Contains(d, "+theme = 'dark'") {
		t.Errorf("expected diff to show file content additions, got: %s", d)
	}

	// Directory new config
	sysAppDir := filepath.Join(sysDir, "myapp")
	_ = os.MkdirAll(sysAppDir, 0755)
	_ = os.WriteFile(filepath.Join(sysAppDir, "app.conf"), []byte("port = 8080\n"), 0644)

	specDir := config.DotfileSpec{
		Name:       "myapp",
		RepoPath:   "myapp",
		SystemPath: sysAppDir,
		Method:     "rsync",
		IsDir:      true,
	}
	entryDir := NewEntry(specDir, repoDir)

	dDir, err := Diff(entryDir)
	if err != nil {
		t.Fatalf("Diff failed for new dir config: %v", err)
	}
	if !strings.Contains(dDir, "+port = 8080") {
		t.Errorf("expected dir diff to show directory file content additions, got: %s", dDir)
	}
}

func TestSetup_NoBackupNameCollision(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	repoDir := filepath.Join(tmpDir, "dotfiles")
	_ = os.MkdirAll(filepath.Join(repoDir, "ghostty"), 0755)
	_ = os.MkdirAll(filepath.Join(repoDir, "sway"), 0755)
	_ = os.WriteFile(filepath.Join(repoDir, "ghostty", "config"), []byte("ghostty repo\n"), 0644)
	_ = os.WriteFile(filepath.Join(repoDir, "sway", "config"), []byte("sway repo\n"), 0644)

	sysGhostty := filepath.Join(tmpDir, ".config", "ghostty", "config")
	sysSway := filepath.Join(tmpDir, ".config", "sway", "config")
	_ = os.MkdirAll(filepath.Dir(sysGhostty), 0755)
	_ = os.MkdirAll(filepath.Dir(sysSway), 0755)
	_ = os.WriteFile(sysGhostty, []byte("ghostty existing system\n"), 0644)
	_ = os.WriteFile(sysSway, []byte("sway existing system\n"), 0644)

	backupDir, err := CreateBackupDir()
	if err != nil {
		t.Fatalf("failed to create backup dir: %v", err)
	}

	entry1 := NewEntry(config.DotfileSpec{
		Name:       "ghostty",
		RepoPath:   "ghostty/config",
		SystemPath: sysGhostty,
	}, repoDir)

	entry2 := NewEntry(config.DotfileSpec{
		Name:       "sway",
		RepoPath:   "sway/config",
		SystemPath: sysSway,
	}, repoDir)

	backedUp1, path1, err := Setup(entry1, backupDir)
	if err != nil || !backedUp1 {
		t.Fatalf("expected entry1 to be backed up, err: %v", err)
	}

	backedUp2, path2, err := Setup(entry2, backupDir)
	if err != nil || !backedUp2 {
		t.Fatalf("expected entry2 to be backed up, err: %v", err)
	}

	if path1 == path2 {
		t.Fatalf("backup paths collided! path1: %s, path2: %s", path1, path2)
	}

	c1, err := os.ReadFile(path1)
	if err != nil || string(c1) != "ghostty existing system\n" {
		t.Errorf("expected ghostty backup to be preserved, got %s (err: %v)", string(c1), err)
	}

	c2, err := os.ReadFile(path2)
	if err != nil || string(c2) != "sway existing system\n" {
		t.Errorf("expected sway backup to be preserved, got %s (err: %v)", string(c2), err)
	}
}
