package main

import (
	"fmt"
	"os"

	"github.com/Majorfi/smartgit/cmd"
	"github.com/spf13/cobra"
)

var (
	version = "dev"
	date    = "unknown"
)

func main() {
	rootCmd := &cobra.Command{
		Use:          "sg",
		Short:        "SmartGit — AI-powered git workflow CLI",
		Long:         "SmartGit augments your git workflow with AI-generated commit messages, branch names, and PR descriptions.",
		SilenceUsage: true,
	}

	rootCmd.Version = version
	rootCmd.SetVersionTemplate(fmt.Sprintf("sg version %s (built %s)\n", version, date))

	rootCmd.AddCommand(cmd.NewCommitCmd())
	rootCmd.AddCommand(cmd.NewStartCmd())
	rootCmd.AddCommand(cmd.NewStatusCmd())
	rootCmd.AddCommand(cmd.NewInitCmd())
	rootCmd.AddCommand(cmd.NewPRCmd())
	rootCmd.AddCommand(cmd.NewDoctorCmd())
	rootCmd.AddCommand(cmd.NewInstallCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
