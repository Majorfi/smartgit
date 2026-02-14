package git

import (
	"fmt"
	"os/exec"
	"strings"
)

func Run(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("git %s: %w\n%s", strings.Join(args, " "), err, string(exitErr.Stderr))
		}
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimRight(string(out), "\n"), nil
}

func Diff(cached bool) (string, error) {
	args := []string{"diff"}
	if cached {
		args = append(args, "--cached")
	}
	return Run(args...)
}

func DiffNameStatus(cached bool) (string, error) {
	args := []string{"diff", "--name-status", "-M", "-C"}
	if cached {
		args = append(args, "--cached")
	}
	return Run(args...)
}

func DiffStat(cached bool) (string, error) {
	args := []string{"diff", "--stat"}
	if cached {
		args = append(args, "--cached")
	}
	return Run(args...)
}

func Status() (string, error) {
	return Run("status", "--porcelain=v2", "-z")
}

func StatusShort() (string, error) {
	return Run("status", "--short")
}

func HasStagedChanges() (bool, error) {
	return hasDiffChanges("--cached")
}

func HasUnstagedChanges() (bool, error) {
	return hasDiffChanges()
}

func hasDiffChanges(extraArgs ...string) (bool, error) {
	args := append([]string{"diff", "--quiet"}, extraArgs...)
	cmd := exec.Command("git", args...)
	err := cmd.Run()
	if err == nil {
		return false, nil
	}
	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
		return true, nil
	}
	return false, err
}

func BranchName() (string, error) {
	return Run("symbolic-ref", "--short", "HEAD")
}

func BranchDescription() (string, error) {
	branch, err := BranchName()
	if err != nil {
		return "", err
	}
	desc, err := Run("config", fmt.Sprintf("branch.%s.description", branch))
	if err != nil {
		return "", nil
	}
	return desc, nil
}

func RecentCommits(n int) (string, error) {
	return Run("log", "--oneline", fmt.Sprintf("-%d", n))
}

func StashKeepIndex() (bool, error) {
	before, _ := Run("rev-parse", "refs/stash")
	_, err := Run("stash", "push", "--keep-index", "--quiet")
	if err != nil {
		return false, err
	}
	after, _ := Run("rev-parse", "refs/stash")
	return before != after, nil
}

func StashPop() error {
	_, err := Run("stash", "pop", "--quiet")
	return err
}

func StageFiles(files []string) error {
	args := append([]string{"add", "--"}, files...)
	_, err := Run(args...)
	return err
}

func AddAll() error {
	_, err := Run("add", "-A")
	return err
}

func CheckRefFormat(name string) error {
	_, err := Run("check-ref-format", "--branch", name)
	return err
}

func SwitchCreate(branch string) error {
	_, err := Run("switch", "-c", branch)
	return err
}

func SetBranchDescription(branch string, description string) error {
	_, err := Run("config", fmt.Sprintf("branch.%s.description", branch), description)
	return err
}

func DefaultBranch() string {
	ref, err := Run("symbolic-ref", "refs/remotes/origin/HEAD")
	if err == nil {
		parts := strings.SplitN(ref, "/", 4)
		if len(parts) == 4 {
			return parts[3]
		}
	}
	for _, candidate := range []string{"main", "master"} {
		if _, err := Run("rev-parse", "--verify", candidate); err == nil {
			return candidate
		}
	}
	return "main"
}

func MergeBase(base string) (string, error) {
	return Run("merge-base", "HEAD", base)
}

func LogSince(since string) (string, error) {
	return Run("log", "--oneline", fmt.Sprintf("%s..HEAD", since))
}

func LogWithTrailers(since string) (string, error) {
	return Run("log", "--format=%h %s%n%(trailers:key=Change-Type,key=Scope,key=Ticket,separator=%x2C )", fmt.Sprintf("%s..HEAD", since))
}

func TrailerValues(since string, trailerKey string) ([]string, error) {
	cmd := exec.Command("git", "log", "--format=%x00%(trailers:key="+trailerKey+",valueonly,separator=%x01)", fmt.Sprintf("%s..HEAD", since))
	raw, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git log trailers: %w", err)
	}
	out := strings.TrimRight(string(raw), "\n")
	if out == "" {
		return nil, nil
	}
	parts := strings.Split(out, "\x00")
	if len(parts) > 0 && parts[0] == "" {
		parts = parts[1:]
	}
	if len(parts) == 0 {
		return nil, nil
	}
	values := make([]string, len(parts))
	for i, p := range parts {
		values[i] = strings.TrimSpace(p)
	}
	return values, nil
}

func DiffStatRange(base string) (string, error) {
	return Run("diff", "--stat", fmt.Sprintf("%s..HEAD", base))
}

func DiffDirstat(base string) (string, error) {
	return Run("diff", "--dirstat", fmt.Sprintf("%s..HEAD", base))
}

func Shortlog(base string) (string, error) {
	return Run("shortlog", "--group=trailer:Change-Type", fmt.Sprintf("%s..HEAD", base))
}

func SetConfig(key string, value string) error {
	_, err := Run("config", key, value)
	return err
}

func Commit(subject string, body string, trailers map[string]string, files []string) error {
	msg := subject
	if body != "" {
		msg = subject + "\n\n" + body
	}

	args := []string{"commit", "-m", msg}
	for key, val := range trailers {
		args = append(args, "--trailer", fmt.Sprintf("%s: %s", key, val))
	}
	if len(files) > 0 {
		args = append(args, "--")
		args = append(args, files...)
	}

	_, err := Run(args...)
	return err
}
