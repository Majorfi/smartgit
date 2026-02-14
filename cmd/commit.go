package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewCommitCmd() *cobra.Command {
	var flagAll bool
	var flagDryRun bool
	var flagNoSplit bool

	commitCmd := &cobra.Command{
		Use:   "commit",
		Short: "AI-powered commit with grouping and trailers",
		Long: `Analyzes staged changes and generates structured commit messages using AI.

The AI proposes one or more commit groups (by concern: backend/api/tests/docs).
For each group, you review and approve the suggested commit message and trailers.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("sg commit is not implemented yet.")
			fmt.Println()
			fmt.Printf("  --all:      %v\n", flagAll)
			fmt.Printf("  --dry-run:  %v\n", flagDryRun)
			fmt.Printf("  --no-split: %v\n", flagNoSplit)
			return nil
		},
	}

	commitCmd.Flags().BoolVarP(&flagAll, "all", "a", false, "Include unstaged changes")
	commitCmd.Flags().BoolVar(&flagDryRun, "dry-run", false, "Preview without executing any commits")
	commitCmd.Flags().BoolVar(&flagNoSplit, "no-split", false, "Force a single commit for all changes")

	return commitCmd
}
