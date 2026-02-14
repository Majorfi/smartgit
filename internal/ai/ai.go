package ai

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type CommitGroup struct {
	Files      []string `json:"files"`
	Subject    string   `json:"subject"`
	Body       string   `json:"body"`
	ChangeType string   `json:"changeType"`
	Scope      string   `json:"scope"`
}

type CommitSuggestion struct {
	Groups []CommitGroup `json:"groups"`
}

type BranchInfo struct {
	BranchName  string   `json:"branchName"`
	Description string   `json:"description"`
	Plan        []string `json:"plan"`
}

var commitSchema = `{
	"type": "object",
	"properties": {
		"groups": {
			"type": "array",
			"items": {
				"type": "object",
				"properties": {
					"files": { "type": "array", "items": { "type": "string" } },
					"subject": { "type": "string" },
					"body": { "type": "string" },
					"changeType": { "type": "string", "enum": ["feature", "fix", "refactor", "docs", "test", "chore", "perf"] },
					"scope": { "type": "string" }
				},
				"required": ["files", "subject", "body", "changeType", "scope"]
			}
		}
	},
	"required": ["groups"]
}`

var branchSchema = `{
	"type": "object",
	"properties": {
		"branchName": { "type": "string" },
		"description": { "type": "string" },
		"plan": { "type": "array", "items": { "type": "string" } }
	},
	"required": ["branchName", "description", "plan"]
}`

func callClaude[T any](prompt string, schema string) (*T, error) {
	claudeBin, err := exec.LookPath("claude")
	if err != nil {
		return nil, fmt.Errorf("claude CLI not found in PATH: %w", err)
	}

	cmd := exec.Command(claudeBin,
		"--print",
		"--output-format", "json",
		"--json-schema", schema,
		"--model", "haiku",
		"--no-session-persistence",
		prompt,
	)

	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("claude CLI error: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("claude CLI error: %w", err)
	}

	var response struct {
		StructuredOutput *T   `json:"structured_output"`
		IsError          bool `json:"is_error"`
		Result           string `json:"result"`
	}
	if err := json.Unmarshal(out, &response); err != nil {
		return nil, fmt.Errorf("failed to parse claude response: %w", err)
	}
	if response.IsError {
		return nil, fmt.Errorf("claude returned error: %s", response.Result)
	}
	if response.StructuredOutput == nil {
		return nil, fmt.Errorf("claude returned no structured output")
	}

	return response.StructuredOutput, nil
}

func GenerateCommitMessage(diff string, context string) (*CommitSuggestion, error) {
	prompt := buildCommitPrompt(diff, context)
	return callClaude[CommitSuggestion](prompt, commitSchema)
}

func GenerateBranchInfo(description string) (*BranchInfo, error) {
	prompt := buildBranchPrompt(description)
	return callClaude[BranchInfo](prompt, branchSchema)
}

func buildCommitPrompt(diff string, context string) string {
	var b strings.Builder
	b.WriteString("You are a commit message generator. Analyze the following git diff and produce structured commit groups.\n\n")
	b.WriteString("Rules:\n")
	b.WriteString("- Group files by concern (e.g. backend, frontend, tests, docs)\n")
	b.WriteString("- Write commit subjects in conventional commit format (e.g. feat(auth): add login flow)\n")
	b.WriteString("- The body should explain WHY the change was made, not just WHAT changed\n")
	b.WriteString("- Keep subjects under 72 characters\n")
	b.WriteString("- Never credit AI in commit messages\n\n")

	if context != "" {
		b.WriteString("Branch context:\n")
		b.WriteString(context)
		b.WriteString("\n\n")
	}

	b.WriteString("Git diff:\n```\n")
	b.WriteString(diff)
	b.WriteString("\n```")

	return b.String()
}

func buildBranchPrompt(description string) string {
	var b strings.Builder
	b.WriteString("You are a git branch name generator. Given a task description, produce a branch name, description, and plan.\n\n")
	b.WriteString("Rules:\n")
	b.WriteString("- Branch name must use format: prefix/short-kebab-description\n")
	b.WriteString("- Valid prefixes: feat, fix, chore, refactor, docs, test\n")
	b.WriteString("- Strip conversational fluff from the input (e.g. 'Can you please...' → imperative form)\n")
	b.WriteString("- Description should explain what the branch is for and any acceptance criteria\n")
	b.WriteString("- Plan should be a list of concrete implementation steps\n")
	b.WriteString("- Keep branch names short (under 50 chars total)\n\n")
	b.WriteString("Task description:\n")
	b.WriteString(description)

	return b.String()
}
