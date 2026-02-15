package cmd

import (
	"fmt"
	"strings"

	"github.com/Majorfi/smartgit/internal/ai"
	"github.com/Majorfi/smartgit/internal/git"
)

func runSplit(all bool, dryRun bool) error {
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

	nameStatus, err := git.DiffNameStatus(true)
	if err != nil {
		return fmt.Errorf("failed to get staged file list: %w", err)
	}

	diffStat, err := git.DiffStat(true)
	if err != nil {
		return fmt.Errorf("failed to get staged diff stats: %w", err)
	}

	context := gatherSplitContext()

	fileCount := len(strings.Split(strings.TrimSpace(nameStatus), "\n"))
	fmt.Printf("Analyzing %d staged files...\n\n", fileCount)

	suggestion, err := ai.GenerateSplitGroups(nameStatus, diffStat, context)
	if err != nil {
		return fmt.Errorf("AI generation failed: %w", err)
	}

	if suggestion == nil || len(suggestion.Groups) == 0 {
		fmt.Println("AI returned no commit groups.")
		return nil
	}

	fmt.Printf("AI proposes %d groups from %d files.\n\n", len(suggestion.Groups), fileCount)

	if dryRun {
		for i, group := range suggestion.Groups {
			fmt.Printf("── Group %d/%d ─────────────────────────────────\n", i+1, len(suggestion.Groups))
			printGroup(group)
		}
		fmt.Printf("Dry run complete. %d group(s) previewed.\n", len(suggestion.Groups))
		return nil
	}

	stashed := false
	hasUnstaged, _ := git.HasUnstagedChanges()
	if hasUnstaged {
		created, err := git.StashKeepIndex()
		if err != nil {
			return fmt.Errorf("failed to stash unstaged changes: %w", err)
		}
		stashed = created
	}

	committed := 0
	for i, group := range suggestion.Groups {
		fmt.Printf("── Group %d/%d ─────────────────────────────────\n", i+1, len(suggestion.Groups))
		printGroup(group)

		action := promptAction()
		switch action {
		case "y":
			if err := splitCommitGroup(group); err != nil {
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

	hasStagedLeftover, _ := git.HasStagedChanges()
	hasUnstagedLeftover, _ := git.HasUnstagedChanges()
	if hasStagedLeftover || hasUnstagedLeftover {
		fmt.Println("Warning: some changes from the original commit remain uncommitted.")
		if status, err := git.StatusShort(); err == nil && status != "" {
			fmt.Println(status)
		}
		fmt.Println("You can finish committing them manually or run `sg commit`.")
		fmt.Println()
	}

	fmt.Printf("Done. %d commit(s) created.\n", committed)
	return nil
}

func gatherSplitContext() string {
	var parts []string

	branch, err := git.BranchName()
	if err == nil && branch != "" {
		parts = append(parts, fmt.Sprintf("Branch: %s", branch))
	}

	desc, err := git.BranchDescription()
	if err == nil && desc != "" {
		parts = append(parts, fmt.Sprintf("Branch description: %s", desc))
	}

	return strings.Join(parts, "\n\n")
}

func splitCommitGroup(group ai.CommitGroup) error {
	if len(group.Files) == 0 {
		return fmt.Errorf("group has no files")
	}

	if err := git.ResetHead(); err != nil {
		return fmt.Errorf("failed to unstage files: %w", err)
	}

	if err := git.StageFiles(group.Files); err != nil {
		return fmt.Errorf("failed to stage group files: %w", err)
	}

	trailers := map[string]string{
		"Change-Type": group.ChangeType,
	}
	if group.Scope != "" {
		trailers["Scope"] = group.Scope
	}

	return git.Commit(group.Subject, group.Body, trailers, nil)
}
