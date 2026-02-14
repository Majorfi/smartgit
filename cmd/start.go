package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/Majorfi/smartgit/internal/ai"
	"github.com/Majorfi/smartgit/internal/git"
	"github.com/Majorfi/smartgit/internal/session"
	"github.com/spf13/cobra"
)

func NewStartCmd() *cobra.Command {
	var flagTicket string

	startCmd := &cobra.Command{
		Use:   "start [description]",
		Short: "Create a branch with AI-generated name and description",
		Long: `Accepts a task description, generates a conventional branch name using AI,
creates the branch, writes a branch description, and saves a session file
with the plan and acceptance criteria.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			description := strings.Join(args, " ")
			return runStart(description, flagTicket)
		},
	}

	startCmd.Flags().StringVarP(&flagTicket, "ticket", "t", "", "Ticket/issue reference (e.g. PROJ-123, #42)")

	return startCmd
}

func runStart(description string, ticket string) error {
	if description == "" {
		fmt.Print("What are you working on? ")
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}
		description = strings.TrimSpace(input)
		if description == "" {
			fmt.Println("No description provided.")
			return nil
		}
	}

	if ticket != "" {
		description = description + "\nTicket: " + ticket
	}

	fmt.Println("Generating branch name...")
	fmt.Println()

	info, err := ai.GenerateBranchInfo(description)
	if err != nil {
		return fmt.Errorf("AI generation failed: %w", err)
	}

	fmt.Printf("  Branch: %s\n", info.BranchName)
	fmt.Printf("  Description: %s\n", info.Description)
	if len(info.Plan) > 0 {
		fmt.Println("  Plan:")
		for _, step := range info.Plan {
			fmt.Printf("    - %s\n", step)
		}
	}
	fmt.Println()

	if err := git.CheckRefFormat(info.BranchName); err != nil {
		return fmt.Errorf("invalid branch name %q: %w", info.BranchName, err)
	}

	action := promptStartAction()
	if action != "y" {
		fmt.Println("Aborted.")
		return nil
	}

	if err := git.SwitchCreate(info.BranchName); err != nil {
		return fmt.Errorf("failed to create branch: %w", err)
	}

	if err := git.SetBranchDescription(info.BranchName, info.Description); err != nil {
		fmt.Printf("Warning: could not set branch description: %v\n", err)
	}

	s := session.Session{
		Branch:      info.BranchName,
		Description: info.Description,
		Plan:        info.Plan,
		Ticket:      ticket,
	}
	if err := session.Save(s); err != nil {
		fmt.Printf("Warning: could not save session file: %v\n", err)
	}

	fmt.Printf("Switched to new branch '%s'\n", info.BranchName)
	return nil
}

func promptStartAction() string {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("  Create this branch? [Y]es / [n]o: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))
		switch input {
		case "y", "yes", "":
			return "y"
		case "n", "no":
			return "n"
		default:
			fmt.Println("  Invalid input.")
		}
	}
}
