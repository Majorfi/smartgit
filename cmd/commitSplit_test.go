package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Majorfi/smartgit/internal/ai"
)

func setupGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGitCmd(t, dir, "init")
	runGitCmd(t, dir, "config", "user.email", "test@example.com")
	runGitCmd(t, dir, "config", "user.name", "Test User")

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	return dir
}

func runGitCmd(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, string(out))
	}
	return strings.TrimSpace(string(out))
}

func writeTestFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	path := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func TestSplitCommitGroupCommitsOnlyGroupFiles(t *testing.T) {
	repo := setupGitRepo(t)

	writeTestFile(t, repo, "base.txt", "base\n")
	runGitCmd(t, repo, "add", "base.txt")
	runGitCmd(t, repo, "commit", "-m", "initial")

	writeTestFile(t, repo, "a.txt", "aaa\n")
	writeTestFile(t, repo, "b.txt", "bbb\n")
	writeTestFile(t, repo, "c.txt", "ccc\n")
	runGitCmd(t, repo, "add", "a.txt", "b.txt", "c.txt")

	group := ai.CommitGroup{
		Files:      []string{"a.txt", "b.txt"},
		Subject:    "feat: add a and b",
		Body:       "test body",
		ChangeType: "feature",
		Scope:      "test",
	}

	if err := splitCommitGroup(group); err != nil {
		t.Fatalf("splitCommitGroup: %v", err)
	}

	log := runGitCmd(t, repo, "log", "--oneline", "-1")
	if !strings.Contains(log, "feat: add a and b") {
		t.Errorf("expected commit subject in log, got: %s", log)
	}

	committed := runGitCmd(t, repo, "diff-tree", "--no-commit-id", "--name-only", "-r", "HEAD")
	if !strings.Contains(committed, "a.txt") || !strings.Contains(committed, "b.txt") {
		t.Errorf("expected a.txt and b.txt in commit, got: %s", committed)
	}
	if strings.Contains(committed, "c.txt") {
		t.Errorf("c.txt should not be in commit, got: %s", committed)
	}

	status := runGitCmd(t, repo, "status", "--porcelain")
	if !strings.Contains(status, "c.txt") {
		t.Errorf("c.txt should remain as uncommitted change, got: %s", status)
	}
}

func TestSplitCommitGroupWithStashDoesNotIncludeUnstagedHunks(t *testing.T) {
	repo := setupGitRepo(t)

	writeTestFile(t, repo, "base.txt", "base\n")
	writeTestFile(t, repo, "f.txt", "line1\n")
	runGitCmd(t, repo, "add", ".")
	runGitCmd(t, repo, "commit", "-m", "initial")

	writeTestFile(t, repo, "f.txt", "line1\nstaged-line\n")
	runGitCmd(t, repo, "add", "f.txt")

	writeTestFile(t, repo, "f.txt", "line1\nstaged-line\nunstaged-line\n")

	runGitCmd(t, repo, "stash", "push", "--keep-index", "--quiet")

	group := ai.CommitGroup{
		Files:      []string{"f.txt"},
		Subject:    "feat: update f",
		Body:       "",
		ChangeType: "feature",
	}

	if err := splitCommitGroup(group); err != nil {
		t.Fatalf("splitCommitGroup: %v", err)
	}

	content := runGitCmd(t, repo, "show", "HEAD:f.txt")
	if strings.Contains(content, "unstaged-line") {
		t.Errorf("committed content should not include unstaged hunk, got: %s", content)
	}
	if !strings.Contains(content, "staged-line") {
		t.Errorf("committed content should include staged hunk, got: %s", content)
	}
}

func TestSplitCommitGroupEmptyFilesError(t *testing.T) {
	setupGitRepo(t)

	group := ai.CommitGroup{
		Files:      []string{},
		Subject:    "empty",
		ChangeType: "chore",
	}
	if err := splitCommitGroup(group); err == nil {
		t.Fatal("expected error for empty files")
	}
}
