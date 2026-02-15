package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/Majorfi/smartgit/internal/ai"
	"github.com/Majorfi/smartgit/internal/config"
	"github.com/Majorfi/smartgit/internal/git"
	"github.com/spf13/cobra"
)

func NewCommitCmd() *cobra.Command {
	var flagAll bool
	var flagDryRun bool
	var flagNoSplit bool
	var flagSplit bool

	commitCmd := &cobra.Command{
		Use:   "commit",
		Short: "AI-powered commit with grouping and trailers",
		Long: `Analyzes staged changes and generates structured commit messages using AI.

The AI proposes one or more commit groups (by concern: backend/api/tests/docs).
For each group, you review and approve the suggested commit message and trailers.

Use --split to group staged changes using only file names and diff stats (no full diff),
useful when the diff is too large for the normal AI analysis.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if flagSplit {
				return runSplit(flagAll, flagDryRun)
			}
			return runCommit(flagAll, flagDryRun, flagNoSplit)
		},
	}

	commitCmd.Flags().BoolVarP(&flagAll, "all", "a", false, "Include unstaged changes")
	commitCmd.Flags().BoolVar(&flagDryRun, "dry-run", false, "Preview without executing any commits")
	commitCmd.Flags().BoolVar(&flagNoSplit, "no-split", false, "Force a single commit for all changes")
	commitCmd.Flags().BoolVar(&flagSplit, "split", false, "Group staged changes using file names and stats only (for large diffs)")

	return commitCmd
}

func runCommit(all bool, dryRun bool, noSplit bool) error {
	if all {
		if err := git.AddAll(); err != nil {
			return fmt.Errorf("failed to stage changes: %w", err)
		}
	}

	hasStagedChanges, err := git.HasStagedChanges()
	if err != nil {
		return fmt.Errorf("failed to check staged changes: %w", err)
	}
	if !hasStagedChanges {
		fmt.Println("No staged changes. Stage files first or use --all.")
		return nil
	}

	hasUnstaged, _ := git.HasUnstagedChanges()
	if hasUnstaged && !all {
		fmt.Println("Warning: you have unstaged changes that won't be included.")
		fmt.Println("Use --all to include them, or stage them manually.")
		fmt.Println()
	}

	diff, err := git.Diff(true)
	if err != nil {
		return fmt.Errorf("failed to get diff: %w", err)
	}

	cfg, cfgErr := config.Load()
	if cfgErr != nil {
		fmt.Printf("Warning: config load: %v\n\n", cfgErr)
	}
	diff = truncateDiff(diff, cfg.DiffMaxLines)

	context := gatherContext()

	fmt.Println("Generating commit message(s)...")
	fmt.Println()

	suggestion, err := ai.GenerateCommitMessage(diff, context)
	if err != nil {
		return fmt.Errorf("AI generation failed: %w", err)
	}

	if suggestion == nil || len(suggestion.Groups) == 0 {
		fmt.Println("AI returned no commit groups.")
		return nil
	}

	if noSplit && len(suggestion.Groups) > 1 {
		suggestion = mergeGroups(suggestion)
	}

	stashed := false
	if !dryRun {
		hasUnstagedNow, _ := git.HasUnstagedChanges()
		if hasUnstagedNow {
			created, err := git.StashKeepIndex()
			if err != nil {
				return fmt.Errorf("failed to stash unstaged changes: %w", err)
			}
			stashed = created
		}
	}

	committed := 0
	for i, group := range suggestion.Groups {
		fmt.Printf("── Group %d/%d ─────────────────────────────────\n", i+1, len(suggestion.Groups))
		printGroup(group)

		if dryRun {
			fmt.Println("  (dry-run, skipping)")
			fmt.Println()
			continue
		}

		action := promptAction()
		switch action {
		case "y":
			if err := commitGroup(group); err != nil {
				fmt.Printf("  Commit failed: %v\n", err)
				continue
			}
			committed++
			fmt.Println("  Committed.")
		case "s":
			fmt.Println("  Skipped.")
		case "q":
			fmt.Println("  Aborting remaining groups.")
			fmt.Println()
			goto done
		}
		fmt.Println()
	}

done:
	if stashed {
		if err := git.StashPop(); err != nil {
			fmt.Printf("Warning: failed to restore unstaged changes: %v\n", err)
			fmt.Println("Your changes are saved in git stash.")
		}
	}

	if dryRun {
		fmt.Printf("Dry run complete. %d group(s) previewed.\n", len(suggestion.Groups))
	} else {
		fmt.Printf("Done. %d commit(s) created.\n", committed)
	}
	return nil
}

func gatherContext() string {
	var parts []string

	branch, err := git.BranchName()
	if err == nil && branch != "" {
		parts = append(parts, fmt.Sprintf("Branch: %s", branch))
	}

	desc, err := git.BranchDescription()
	if err == nil && desc != "" {
		parts = append(parts, fmt.Sprintf("Branch description: %s", desc))
	}

	recent, err := git.RecentCommits(5)
	if err == nil && recent != "" {
		parts = append(parts, fmt.Sprintf("Recent commits:\n%s", recent))
	}

	nameStatus, err := git.DiffNameStatus(true)
	if err == nil && nameStatus != "" {
		parts = append(parts, fmt.Sprintf("Changed files:\n%s", nameStatus))
	}

	stat, err := git.DiffStat(true)
	if err == nil && stat != "" {
		parts = append(parts, fmt.Sprintf("Diff stats:\n%s", stat))
	}

	return strings.Join(parts, "\n\n")
}

func printGroup(group ai.CommitGroup) {
	fmt.Printf("  Subject: %s\n", group.Subject)
	if group.Body != "" {
		fmt.Printf("  Body:    %s\n", group.Body)
	}
	fmt.Printf("  Type:    %s\n", group.ChangeType)
	if group.Scope != "" {
		fmt.Printf("  Scope:   %s\n", group.Scope)
	}
	fmt.Printf("  Files:\n")
	for _, f := range group.Files {
		fmt.Printf("    - %s\n", f)
	}
	fmt.Println()
}

func promptAction() string {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("  [Y]es / [s]kip / [q]uit: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))
		switch input {
		case "y", "yes", "":
			return "y"
		case "s", "skip":
			return "s"
		case "q", "quit":
			return "q"
		default:
			fmt.Println("  Invalid input.")
		}
	}
}

func commitGroup(group ai.CommitGroup) error {
	if len(group.Files) == 0 {
		return fmt.Errorf("group has no files")
	}

	trailers := map[string]string{
		"Change-Type": group.ChangeType,
	}
	if group.Scope != "" {
		trailers["Scope"] = group.Scope
	}

	return git.Commit(group.Subject, group.Body, trailers, group.Files)
}

func mergeGroups(suggestion *ai.CommitSuggestion) *ai.CommitSuggestion {
	var allFiles []string
	var bodyParts []string
	for _, g := range suggestion.Groups {
		allFiles = append(allFiles, g.Files...)
		if g.Body != "" {
			bodyParts = append(bodyParts, g.Body)
		}
	}

	merged := ai.CommitGroup{
		Files:      allFiles,
		Subject:    suggestion.Groups[0].Subject,
		Body:       strings.Join(bodyParts, "\n\n"),
		ChangeType: suggestion.Groups[0].ChangeType,
		Scope:      suggestion.Groups[0].Scope,
	}

	return &ai.CommitSuggestion{Groups: []ai.CommitGroup{merged}}
}

func truncateDiff(diff string, maxLines int) string {
	if maxLines <= 0 {
		return diff
	}
	lines := strings.Split(diff, "\n")
	if len(lines) <= maxLines {
		return diff
	}
	fmt.Printf("Warning: diff truncated from %d to %d lines.\n", len(lines), maxLines)
	return strings.Join(lines[:maxLines], "\n")
}
