package cmd

import (
	"fmt"
	"strings"

	"github.com/Majorfi/smartgit/internal/config"
	"github.com/Majorfi/smartgit/internal/git"
	"github.com/Majorfi/smartgit/internal/session"
	"github.com/spf13/cobra"
)

func NewStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show traceability breadcrumb trail",
		Long: `Displays the full traceability breadcrumb for the current branch:
branch name and description, commits since fork point, staged/unstaged
changes, remaining plan items, and readiness flags.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus()
		},
	}
}

func runStatus() error {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Warning: config load: %v\n\n", err)
	}
	if cfg.BaseBranch == "" {
		cfg.BaseBranch = git.DefaultBranch()
	}

	branch, err := git.BranchName()
	if err != nil {
		return fmt.Errorf("not on a branch: %w", err)
	}

	fmt.Printf("Branch: %s\n", branch)

	desc, _ := git.BranchDescription()
	if desc != "" {
		fmt.Printf("Description: %s\n", desc)
	}
	fmt.Println()

	printCommitsSinceFork(cfg.BaseBranch)
	printWorkingTree()
	printSession()
	printReadiness(cfg.BaseBranch)

	return nil
}

func printCommitsSinceFork(baseBranch string) {
	mergeBase, err := git.MergeBase(baseBranch)
	if err != nil {
		return
	}

	log, err := git.LogWithTrailers(mergeBase)
	if err != nil || log == "" {
		fmt.Println("Commits: (none since fork)")
		fmt.Println()
		return
	}

	fmt.Println("Commits since fork:")
	fmt.Println(log)
	fmt.Println()
}

func printWorkingTree() {
	hasStagedChanges, _ := git.HasStagedChanges()
	hasUnstaged, _ := git.HasUnstagedChanges()

	if !hasStagedChanges && !hasUnstaged {
		fmt.Println("Working tree: clean")
	} else {
		fmt.Println("Working tree:")
		if hasStagedChanges {
			stat, _ := git.DiffStat(true)
			fmt.Printf("  Staged:\n%s\n", stat)
		}
		if hasUnstaged {
			stat, _ := git.DiffStat(false)
			fmt.Printf("  Unstaged:\n%s\n", stat)
		}
	}
	fmt.Println()
}

func printSession() {
	if !session.Exists() {
		return
	}

	s, err := session.Load()
	if err != nil {
		return
	}

	if len(s.Plan) > 0 {
		fmt.Println("Plan:")
		for _, step := range s.Plan {
			fmt.Printf("  [ ] %s\n", step)
		}
		fmt.Println()
	}

	if s.Ticket != "" {
		fmt.Printf("Ticket: %s\n", s.Ticket)
		fmt.Println()
	}
}

func printReadiness(baseBranch string) {
	fmt.Println("Readiness:")

	desc, _ := git.BranchDescription()
	printFlag("Branch description", desc != "")

	mergeBase, err := git.MergeBase(baseBranch)
	if err != nil {
		printFlag("Has commits", false)
		printFlag("All commits have Change-Type", false)
		return
	}

	log, _ := git.LogSince(mergeBase)
	hasCommits := log != ""
	printFlag("Has commits", hasCommits)

	if hasCommits {
		printFlag("All commits have Change-Type", checkTrailersPresent(mergeBase))
	} else {
		printFlag("All commits have Change-Type", false)
	}
}

func printFlag(label string, ok bool) {
	marker := "x"
	if !ok {
		marker = " "
	}
	fmt.Printf("  [%s] %s\n", marker, label)
}

func checkTrailersPresent(mergeBase string) bool {
	values, err := git.TrailerValues(mergeBase, "Change-Type")
	if err != nil || len(values) == 0 {
		return false
	}
	for _, v := range values {
		if strings.TrimSpace(v) == "" {
			return false
		}
	}
	return true
}
