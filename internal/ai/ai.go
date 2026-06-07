package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ErrTimeout is returned when the claude CLI exceeds its deadline, letting
// callers fall back to a lighter strategy (e.g. split) instead of failing.
var ErrTimeout = errors.New("claude CLI timed out")

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

type PRDescription struct {
	Title           string `json:"title"`
	Summary         string `json:"summary"`
	WhatChanged     string `json:"whatChanged"`
	Risks           string `json:"risks"`
	TestPlan        string `json:"testPlan"`
	DocsImpact      string `json:"docsImpact"`
	BreakingChanges string `json:"breakingChanges"`
}

var prSchema = `{
	"type": "object",
	"properties": {
		"title": { "type": "string" },
		"summary": { "type": "string" },
		"whatChanged": { "type": "string" },
		"risks": { "type": "string" },
		"testPlan": { "type": "string" },
		"docsImpact": { "type": "string" },
		"breakingChanges": { "type": "string" }
	},
	"required": ["title", "summary", "whatChanged", "risks", "testPlan", "docsImpact", "breakingChanges"]
}`

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

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// The CLI's --json-schema (structured output) flag hangs with haiku, so we
	// ask for JSON in the prompt and parse the result string ourselves.
	fullPrompt := prompt + "\n\nRespond with ONLY a JSON object matching this schema. No markdown fences, no prose:\n" + schema

	cmd := exec.CommandContext(ctx, claudeBin,
		"--print",
		"--output-format", "json",
		"--model", "haiku",
		"--no-session-persistence",
		"--strict-mcp-config",
	)
	cmd.Stdin = strings.NewReader(fullPrompt)

	out, err := cmd.Output()

	// claude --output-format json prints its payload (including API errors) to
	// stdout and exits non-zero, leaving stderr empty. Parse stdout before the
	// process error so the real message surfaces instead of a blank one.
	var response struct {
		IsError bool   `json:"is_error"`
		Result  string `json:"result"`
	}
	parseErr := json.Unmarshal(out, &response)

	if parseErr == nil && response.IsError {
		return nil, fmt.Errorf("claude returned error: %s", response.Result)
	}

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("claude CLI timed out after 2m (diff may be too large): %w", ErrTimeout)
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := strings.TrimSpace(string(exitErr.Stderr))
			if stderr != "" {
				return nil, fmt.Errorf("claude CLI error: %s", stderr)
			}
			return nil, fmt.Errorf("claude CLI exited with code %d and no output", exitErr.ExitCode())
		}
		return nil, fmt.Errorf("claude CLI error: %w", err)
	}

	if parseErr != nil {
		return nil, fmt.Errorf("failed to parse claude response: %w", parseErr)
	}

	return parseResultJSON[T](response.Result)
}

func parseResultJSON[T any](result string) (*T, error) {
	start := strings.Index(result, "{")
	end := strings.LastIndex(result, "}")
	if start < 0 || end <= start {
		return nil, fmt.Errorf("no JSON object in claude response: %q", result)
	}

	var v T
	if err := json.Unmarshal([]byte(result[start:end+1]), &v); err != nil {
		return nil, fmt.Errorf("failed to parse JSON from claude response: %w", err)
	}
	return &v, nil
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

func GeneratePRDescription(branchDesc, commitLog, diffStat, dirStat, shortlog string) (*PRDescription, error) {
	prompt := buildPRPrompt(branchDesc, commitLog, diffStat, dirStat, shortlog)
	return callClaude[PRDescription](prompt, prSchema)
}

func buildPRPrompt(branchDesc, commitLog, diffStat, dirStat, shortlog string) string {
	var b strings.Builder
	b.WriteString("You are a pull request description generator. Analyze the following branch information and produce a structured PR description.\n\n")
	b.WriteString("Rules:\n")
	b.WriteString("- Title should be concise (under 72 chars), in imperative mood\n")
	b.WriteString("- Summary should be 1-3 sentences explaining the purpose\n")
	b.WriteString("- WhatChanged should list the key changes made\n")
	b.WriteString("- Risks should identify potential issues or areas to watch\n")
	b.WriteString("- TestPlan should suggest how to verify the changes\n")
	b.WriteString("- DocsImpact should note any documentation that needs updating, or 'None' if not applicable\n")
	b.WriteString("- BreakingChanges should list any breaking changes, or 'None' if not applicable\n")
	b.WriteString("- Never credit AI in the output\n\n")

	if branchDesc != "" {
		b.WriteString("Branch description:\n")
		b.WriteString(branchDesc)
		b.WriteString("\n\n")
	}

	if commitLog != "" {
		b.WriteString("Commit log:\n```\n")
		b.WriteString(commitLog)
		b.WriteString("\n```\n\n")
	}

	if diffStat != "" {
		b.WriteString("Diff stats:\n```\n")
		b.WriteString(diffStat)
		b.WriteString("\n```\n\n")
	}

	if dirStat != "" {
		b.WriteString("Directory impact:\n```\n")
		b.WriteString(dirStat)
		b.WriteString("\n```\n\n")
	}

	if shortlog != "" {
		b.WriteString("Change type breakdown:\n```\n")
		b.WriteString(shortlog)
		b.WriteString("\n```\n\n")
	}

	return b.String()
}

func GenerateSplitGroups(nameStatus string, diffStat string, context string) (*CommitSuggestion, error) {
	prompt := buildSplitPrompt(nameStatus, diffStat, context)
	return callClaude[CommitSuggestion](prompt, commitSchema)
}

func buildSplitPrompt(nameStatus string, diffStat string, context string) string {
	var b strings.Builder
	b.WriteString("You are a commit splitter. Given a list of files from a single large commit, group them into logical commits by concern.\n\n")
	b.WriteString("Rules:\n")
	b.WriteString("- Group files that belong to the same feature, module, or concern\n")
	b.WriteString("- Each group should be a coherent, self-contained change\n")
	b.WriteString("- Write commit subjects in conventional commit format (e.g. feat(auth): add login flow)\n")
	b.WriteString("- The body should explain WHY the change was made\n")
	b.WriteString("- Keep subjects under 72 characters\n")
	b.WriteString("- Never credit AI in commit messages\n")
	b.WriteString("- Every file must appear in exactly one group\n\n")

	if context != "" {
		b.WriteString("Branch context:\n")
		b.WriteString(context)
		b.WriteString("\n\n")
	}

	b.WriteString("Changed files:\n```\n")
	b.WriteString(nameStatus)
	b.WriteString("\n```\n\n")

	b.WriteString("Diff stats:\n```\n")
	b.WriteString(diffStat)
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
