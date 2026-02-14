package main

import (
	"fmt"
	"os"

	"github.com/Majorfi/smartgit/cmd"
	"github.com/spf13/cobra"
)

var version = "dev"

func main() {
	rootCmd := &cobra.Command{
		Use:   "sg",
		Short: "SmartGit — AI-powered git workflow CLI",
		Long:  "SmartGit augments your git workflow with AI-generated commit messages, branch names, and PR descriptions.",
	}

	rootCmd.Version = version
	rootCmd.SetVersionTemplate(fmt.Sprintf("sg version %s\n", version))

	rootCmd.AddCommand(cmd.NewCommitCmd())
	rootCmd.AddCommand(cmd.NewStartCmd())
	rootCmd.AddCommand(cmd.NewStatusCmd())
	rootCmd.AddCommand(cmd.NewInitCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
