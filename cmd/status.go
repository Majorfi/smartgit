package cmd

import (
	"fmt"

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
	cfg, _ := config.Load()

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
		printFlag("Commits with trailers", false)
		return
	}

	log, _ := git.LogSince(mergeBase)
	printFlag("Has commits", log != "")
}

func printFlag(label string, ok bool) {
	marker := "x"
	if !ok {
		marker = " "
	}
	fmt.Printf("  [%s] %s\n", marker, label)
}
