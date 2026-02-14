package cmd

import (
	"fmt"
	"strings"

	"github.com/Majorfi/smartgit/internal/config"
	"github.com/Majorfi/smartgit/internal/git"
	"github.com/Majorfi/smartgit/internal/session"
	"github.com/spf13/cobra"
)

func NewDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Run traceability diagnostic checks",
		Long: `Checks the current branch for traceability best practices:
branch description, commit trailers, ticket references, session plan,
and commit history since the fork point.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDoctor()
		},
	}
}

func runDoctor() error {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Warning: config load: %v\n\n", err)
	}
	if cfg.BaseBranch == "" {
		cfg.BaseBranch = git.DefaultBranch()
	}

	mergeBase, err := git.MergeBase(cfg.BaseBranch)
	if err != nil {
		return fmt.Errorf("failed to find merge base with %s: %w", cfg.BaseBranch, err)
	}

	issues := 0
	totalChecks := 0

	fmt.Println("sg doctor — traceability check")
	fmt.Println()

	desc, _ := git.BranchDescription()
	totalChecks++
	if !printCheck("Branch has description", desc != "") {
		issues++
	}

	log, _ := git.LogSince(mergeBase)
	hasCommits := log != ""
	totalChecks++
	if !printCheck("Has commits since fork", hasCommits) {
		issues++
	}

	if hasCommits {
		totalChecks++
		if !printCheck("All commits have Change-Type trailer", checkAllTrailers(mergeBase)) {
			issues++
		}
		totalChecks++
		if !printCheck("Ticket trailer present", checkTicketPresent(mergeBase)) {
			issues++
		}
	} else {
		fmt.Println("  [-] All commits have Change-Type trailer (skipped, no commits)")
		fmt.Println("  [-] Ticket trailer present (skipped, no commits)")
	}

	totalChecks++
	if !printCheck("Session plan exists", session.Exists()) {
		issues++
	}

	fmt.Println()
	if issues == 0 {
		fmt.Println("All checks passed.")
	} else {
		fmt.Printf("%d/%d issues found.\n", issues, totalChecks)
	}

	return nil
}

func checkAllTrailers(mergeBase string) bool {
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

func checkTicketPresent(mergeBase string) bool {
	values, err := git.TrailerValues(mergeBase, "Ticket")
	if err != nil || len(values) == 0 {
		return false
	}
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return true
		}
	}
	return false
}

func printCheck(label string, ok bool) bool {
	marker := "x"
	if !ok {
		marker = " "
	}
	fmt.Printf("  [%s] %s\n", marker, label)
	return ok
}
