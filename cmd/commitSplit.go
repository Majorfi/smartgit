package cmd

import (
	"fmt"
	"strings"

	"github.com/Majorfi/smartgit/internal/ai"
	"github.com/Majorfi/smartgit/internal/git"
)

func runSplit(dryRun bool) error {
	nameStatus, err := git.DiffTreeNameStatus("HEAD")
	if err != nil {
		return fmt.Errorf("failed to get file list from HEAD: %w", err)
	}
	if strings.TrimSpace(nameStatus) == "" {
		fmt.Println("HEAD commit has no file changes.")
		return nil
	}

	diffStat, err := git.DiffStatOfCommit("HEAD")
	if err != nil {
		return fmt.Errorf("failed to get diff stats from HEAD: %w", err)
	}

	context := gatherSplitContext()

	fileCount := len(strings.Split(strings.TrimSpace(nameStatus), "\n"))
	fmt.Printf("Analyzing %d files from HEAD commit...\n\n", fileCount)

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

	if err := git.SoftReset("HEAD~1"); err != nil {
		return fmt.Errorf("failed to soft reset HEAD: %w", err)
	}
	fmt.Println("Soft-reset HEAD. All changes are now staged.")
	fmt.Println()

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
	hasStagedLeftover, _ := git.HasStagedChanges()
	if hasStagedLeftover {
		fmt.Println("Warning: some files remain staged but uncommitted.")
		fmt.Println("You can commit them manually or run `sg commit`.")
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
