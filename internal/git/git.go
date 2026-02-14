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

func Status() (string, error) {
	return Run("status", "--porcelain=v2", "-z")
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
	return Run("log", fmt.Sprintf("--oneline"), fmt.Sprintf("-%d", n))
}
