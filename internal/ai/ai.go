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

func claudePath() (string, error) {
	path, err := exec.LookPath("claude")
	if err != nil {
		return "", fmt.Errorf("claude CLI not found in PATH: %w", err)
	}
	return path, nil
}

func GenerateCommitMessage(diff string, context string) (*CommitSuggestion, error) {
	claudeBin, err := claudePath()
	if err != nil {
		return nil, err
	}

	prompt := buildCommitPrompt(diff, context)

	cmd := exec.Command(claudeBin,
		"--print",
		"--output-format", "json",
		"--json-schema", commitSchema,
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
		StructuredOutput *CommitSuggestion `json:"structured_output"`
		IsError          bool              `json:"is_error"`
		Result           string            `json:"result"`
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
