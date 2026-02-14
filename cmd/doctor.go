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
	cfg, _ := config.Load()

	mergeBase, err := git.MergeBase(cfg.BaseBranch)
	if err != nil {
		return fmt.Errorf("failed to find merge base with %s: %w", cfg.BaseBranch, err)
	}

	issues := 0

	fmt.Println("sg doctor — traceability check")
	fmt.Println()

	desc, _ := git.BranchDescription()
	if !printCheck("Branch has description", desc != "") {
		issues++
	}

	allTrailers := checkAllTrailers(mergeBase)
	if !printCheck("All commits have Change-Type trailer", allTrailers) {
		issues++
	}

	hasTicket := checkTicketPresent(mergeBase)
	if !printCheck("Ticket trailer present", hasTicket) {
		issues++
	}

	if !printCheck("Session plan exists", session.Exists()) {
		issues++
	}

	log, _ := git.LogSince(mergeBase)
	if !printCheck("Has commits since fork", log != "") {
		issues++
	}

	fmt.Println()
	if issues == 0 {
		fmt.Println("All checks passed.")
	} else {
		fmt.Printf("%d/%d issues found.\n", issues, 5)
	}

	return nil
}

func checkAllTrailers(mergeBase string) bool {
	log, err := git.LogWithTrailers(mergeBase)
	if err != nil || log == "" {
		return false
	}

	lines := strings.Split(log, "\n")
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		if !strings.Contains(line, "Change-Type") {
			hasCommitOnPrev := i > 0 && isCommitLine(lines[i-1])
			if isCommitLine(line) {
				if i+1 >= len(lines) || strings.TrimSpace(lines[i+1]) == "" {
					return false
				}
			}
			if hasCommitOnPrev {
				return false
			}
		}
	}
	return true
}

func checkTicketPresent(mergeBase string) bool {
	log, err := git.LogWithTrailers(mergeBase)
	if err != nil || log == "" {
		return false
	}
	return strings.Contains(log, "Ticket")
}

func isCommitLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if len(trimmed) < 8 {
		return false
	}
	for _, c := range trimmed[:7] {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return trimmed[7] == ' '
}

func printCheck(label string, ok bool) bool {
	marker := "x"
	if !ok {
		marker = " "
	}
	fmt.Printf("  [%s] %s\n", marker, label)
	return ok
}
