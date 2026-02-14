package cmd

import (
	"fmt"
	"os"

	"github.com/Majorfi/smartgit/internal/git"
	"github.com/spf13/cobra"
)

func NewInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Set up SmartGit in the current repository",
		Long: `Configures git settings for SmartGit:
- Sets merge.branchdesc = true (include branch descriptions in merge commits)
- Sets commit.cleanup = whitespace (preserve structured content)
- Configures trailer aliases for Change-Type, Scope, and Ticket
- Creates the .sg/ directory`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit()
		},
	}
}

func runInit() error {
	configs := []struct {
		key   string
		value string
		label string
	}{
		{"merge.branchdesc", "true", "merge.branchdesc = true"},
		{"commit.cleanup", "whitespace", "commit.cleanup = whitespace"},
		{"trailer.Change-Type.key", "Change-Type", "trailer alias: Change-Type"},
		{"trailer.Scope.key", "Scope", "trailer alias: Scope"},
		{"trailer.Ticket.key", "Ticket", "trailer alias: Ticket"},
	}

	for _, c := range configs {
		if err := git.SetConfig(c.key, c.value); err != nil {
			fmt.Printf("  Warning: failed to set %s: %v\n", c.label, err)
		} else {
			fmt.Printf("  Set %s\n", c.label)
		}
	}

	if err := os.MkdirAll(".sg", 0755); err != nil {
		fmt.Printf("  Warning: failed to create .sg/: %v\n", err)
	} else {
		fmt.Println("  Created .sg/ directory")
	}

	fmt.Println()
	fmt.Println("SmartGit initialized. You can now use:")
	fmt.Println("  sg start <description>  — create a branch")
	fmt.Println("  sg commit               — AI-powered commit")
	fmt.Println("  sg status               — traceability overview")

	return nil
}
