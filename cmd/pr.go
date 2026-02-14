package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Majorfi/smartgit/internal/ai"
	"github.com/Majorfi/smartgit/internal/config"
	"github.com/Majorfi/smartgit/internal/git"
	"github.com/spf13/cobra"
)

func NewPRCmd() *cobra.Command {
	var flagBase string
	var flagDryRun bool
	var flagLocal bool
	var flagTitle string

	prCmd := &cobra.Command{
		Use:   "pr",
		Short: "Create a PR with AI-generated description",
		Long: `Gathers branch context (commits, diffs, descriptions) and uses AI to
generate a structured PR description. Then creates the PR via gh CLI.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPR(flagBase, flagDryRun, flagLocal, flagTitle)
		},
	}

	prCmd.Flags().StringVar(&flagBase, "base", "", "Base branch (default from config)")
	prCmd.Flags().BoolVar(&flagDryRun, "dry-run", false, "Preview PR body without creating")
	prCmd.Flags().BoolVar(&flagLocal, "local", false, "Write PR description to .sg/pr.md instead of creating")
	prCmd.Flags().StringVar(&flagTitle, "title", "", "Override AI-generated title")

	return prCmd
}

func runPR(base string, dryRun bool, local bool, titleOverride string) error {
	if base == "" {
		cfg, err := config.Load()
		if err != nil {
			fmt.Printf("Warning: config load: %v\n\n", err)
		}
		base = cfg.BaseBranch
		if base == "" {
			base = git.DefaultBranch()
		}
	}

	mergeBase, err := git.MergeBase(base)
	if err != nil {
		return fmt.Errorf("failed to find merge base with %s: %w", base, err)
	}

	branchDesc, _ := git.BranchDescription()
	commitLog, err := git.LogWithTrailers(mergeBase)
	if err != nil {
		return fmt.Errorf("failed to read commit log: %w", err)
	}
	if commitLog == "" {
		fmt.Println("No commits found since fork point. Nothing to create a PR for.")
		return nil
	}

	diffStat, _ := git.DiffStatRange(mergeBase)
	dirStat, _ := git.DiffDirstat(mergeBase)
	shortlog, _ := git.Shortlog(mergeBase)

	fmt.Println("Generating PR description...")
	fmt.Println()

	desc, err := ai.GeneratePRDescription(branchDesc, commitLog, diffStat, dirStat, shortlog)
	if err != nil {
		return fmt.Errorf("AI generation failed: %w", err)
	}

	title := desc.Title
	if titleOverride != "" {
		title = titleOverride
	}

	body := formatPRBody(desc)

	fmt.Printf("Title: %s\n\n", title)
	fmt.Println(body)
	fmt.Println()

	if dryRun {
		fmt.Println("(dry-run, PR not created)")
		return nil
	}

	if local {
		return writePRLocal(title, body)
	}

	action := promptPRAction()
	if action != "y" {
		fmt.Println("Aborted.")
		return nil
	}

	return createPR(base, title, body)
}

func formatPRBody(desc *ai.PRDescription) string {
	var b strings.Builder
	b.WriteString("## Summary\n")
	b.WriteString(desc.Summary)
	b.WriteString("\n\n")
	b.WriteString("## What changed\n")
	b.WriteString(desc.WhatChanged)
	b.WriteString("\n\n")
	b.WriteString("## Risks\n")
	b.WriteString(desc.Risks)
	b.WriteString("\n\n")
	b.WriteString("## Test plan\n")
	b.WriteString(desc.TestPlan)
	b.WriteString("\n\n")
	b.WriteString("## Docs impact\n")
	b.WriteString(desc.DocsImpact)
	b.WriteString("\n\n")
	b.WriteString("## Breaking changes\n")
	b.WriteString(desc.BreakingChanges)
	return b.String()
}

func writePRLocal(title string, body string) error {
	dir := ".sg"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create %s directory: %w", dir, err)
	}

	path := filepath.Join(dir, "pr.md")
	content := "# " + title + "\n\n" + body + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}

	fmt.Printf("PR description written to %s\n", path)
	return nil
}

func createPR(base string, title string, body string) error {
	ghBin, err := exec.LookPath("gh")
	if err != nil {
		return fmt.Errorf("gh CLI not found in PATH: %w", err)
	}

	cmd := exec.Command(ghBin, "pr", "create",
		"--base", base,
		"--title", title,
		"--body", body,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gh pr create failed: %w", err)
	}

	return nil
}

func promptPRAction() string {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Create this PR? [Y]es / [n]o: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))
		switch input {
		case "y", "yes", "":
			return "y"
		case "n", "no":
			return "n"
		default:
			fmt.Println("Invalid input.")
		}
	}
}
