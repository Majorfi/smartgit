package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func setupRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test User")

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})

	return dir
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(out))
	}
	return strings.TrimSpace(string(out))
}

func writeFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	path := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

func TestHasWorkingTreeChanges(t *testing.T) {
	repo := setupRepo(t)

	writeFile(t, repo, "a.txt", "base\n")
	runGit(t, repo, "add", "a.txt")
	runGit(t, repo, "commit", "-m", "base")

	changed, err := HasWorkingTreeChanges()
	if err != nil {
		t.Fatalf("HasWorkingTreeChanges clean: %v", err)
	}
	if changed {
		t.Fatalf("expected clean working tree")
	}

	writeFile(t, repo, "a.txt", "base\nnext\n")
	changed, err = HasWorkingTreeChanges()
	if err != nil {
		t.Fatalf("HasWorkingTreeChanges dirty: %v", err)
	}
	if !changed {
		t.Fatalf("expected dirty working tree")
	}
}

func TestCommitHasParent(t *testing.T) {
	repo := setupRepo(t)

	writeFile(t, repo, "a.txt", "base\n")
	runGit(t, repo, "add", "a.txt")
	runGit(t, repo, "commit", "-m", "root")

	hasParent, err := CommitHasParent("HEAD")
	if err != nil {
		t.Fatalf("CommitHasParent root: %v", err)
	}
	if hasParent {
		t.Fatalf("root commit should not have a parent")
	}

	writeFile(t, repo, "a.txt", "base\nnext\n")
	runGit(t, repo, "add", "a.txt")
	runGit(t, repo, "commit", "-m", "second")

	hasParent, err = CommitHasParent("HEAD")
	if err != nil {
		t.Fatalf("CommitHasParent second: %v", err)
	}
	if !hasParent {
		t.Fatalf("non-root commit should have a parent")
	}
}

func TestDiffHelpersIncludeRootCommitChanges(t *testing.T) {
	repo := setupRepo(t)

	writeFile(t, repo, "a.txt", "base\n")
	runGit(t, repo, "add", "a.txt")
	runGit(t, repo, "commit", "-m", "root")

	nameStatus, err := DiffTreeNameStatus("HEAD")
	if err != nil {
		t.Fatalf("DiffTreeNameStatus: %v", err)
	}
	if !strings.Contains(nameStatus, "A\ta.txt") {
		t.Fatalf("expected root commit file in name-status, got: %q", nameStatus)
	}

	diffStat, err := DiffStatOfCommit("HEAD")
	if err != nil {
		t.Fatalf("DiffStatOfCommit: %v", err)
	}
	if !strings.Contains(diffStat, "a.txt") {
		t.Fatalf("expected root commit file in diff stat, got: %q", diffStat)
	}
}
