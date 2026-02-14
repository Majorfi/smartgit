# SmartGit (`sg`)

AI-powered git workflow CLI that maintains traceability from intent to pull request.

## What it does

- **Augments git, doesn't replace it** — plugs into native git mechanisms (trailers, branch descriptions, hooks)
- **AI-powered commits** — analyzes staged changes, groups them by concern, generates structured commit messages with trailers
- **AI-powered branches & PRs** — generates conventional branch names from task descriptions and structured PR bodies from branch context
- **Traceability chain** — maintains the link: Intent → Branch → Commits → PR → Docs

## Requirements

| Dependency | Version | Purpose |
|---|---|---|
| [Go](https://go.dev/) | 1.24+ | Build from source |
| [git](https://git-scm.com/) | any | Version control |
| [Claude CLI](https://docs.anthropic.com/en/docs/claude-cli) | any | AI features (`sg commit`, `sg start`, `sg pr`) |
| [GitHub CLI (`gh`)](https://cli.github.com/) | any | PR creation (`sg pr`) |

## Installation

```bash
go install github.com/Majorfi/smartgit@latest
```

Or build from source:

```bash
git clone https://github.com/Majorfi/smartgit.git
cd smartgit
go build -o sg .
./sg install
```

`sg install` copies the binary to a directory in your `PATH`.

## Quick start

```bash
# Set up SmartGit in your repo
sg init

# Start a new task (creates branch + session)
sg start "add user authentication" --ticket PROJ-42

# Make your changes, then commit with AI
sg commit

# Check traceability status
sg status

# Create a PR with AI-generated description
sg pr
```

## Commands

| Command | Description | Key flags |
|---|---|---|
| `sg init` | Set up SmartGit in the current repository | — |
| `sg start [description]` | Create a branch with AI-generated name | `--ticket`, `-t` |
| `sg commit` | AI-powered commit with grouping and trailers | `--all`, `--dry-run`, `--no-split` |
| `sg status` | Show traceability breadcrumb trail | — |
| `sg pr` | Create a PR with AI-generated description | `--base`, `--title`, `--dry-run`, `--local` |
| `sg doctor` | Run traceability diagnostic checks | — |
| `sg install` | Install sg binary into your PATH | `--dir` |

### `sg init`

Configures git settings for SmartGit: enables branch descriptions (`merge.branchdesc`), sets commit cleanup to `whitespace`, configures trailer aliases for `Change-Type`, `Scope`, and `Ticket`, and creates the `.sg/` directory.

### `sg start`

Accepts a task description, generates a conventional branch name using AI, creates the branch, writes a branch description, and saves a session file (`.sg/session.json`) with the plan and acceptance criteria.

### `sg commit`

Analyzes staged changes and generates structured commit messages. The AI proposes one or more commit groups (by concern: backend/api/tests/docs). For each group, you review and approve the suggested message and trailers.

- `--all` / `-a` — stage all changes before committing
- `--dry-run` — preview without executing any commits
- `--no-split` — force a single commit for all changes

### `sg pr`

Gathers branch context (commits, diffs, descriptions) and uses AI to generate a structured PR body with Summary, What Changed, Risks, Test Plan, Docs Impact, and Breaking Changes sections.

- `--base` — override the base branch
- `--title` — override the AI-generated title
- `--dry-run` — preview PR body without creating
- `--local` — write PR description to `.sg/pr.md` instead of creating

### `sg doctor`

Runs 5 traceability checks on the current branch: branch description, commit trailers, ticket references, session plan, and commit history since the fork point. No AI required.

### `sg status`

Displays the full traceability breadcrumb for the current branch: branch name and description, commits since fork point, staged/unstaged changes, remaining plan items, and readiness flags.

## Configuration

SmartGit uses a 3-layer config system. Each layer overrides the previous:

1. **Global** — `~/.config/sg/config.json` (user defaults)
2. **Project** — `.sg/config.json` (shared with team, committed)
3. **Local** — `.sg/config.local.json` (personal overrides, gitignored)

### Config keys

| Key | Type | Default | Description |
|---|---|---|---|
| `diffMaxLines` | int | `2000` | Max diff lines sent to AI |
| `commitStyle` | string | `"conventional"` | Commit message convention |
| `requiredTrailers` | string[] | `[]` | Trailers that must be present |
| `optionalTrailers` | string[] | `[]` | Trailers suggested but not enforced |
| `branchPrefixes` | string[] | `[]` | Allowed branch name prefixes |
| `baseBranch` | string | `""` | Default base branch for PR/diff |
| `prTemplate` | string | `""` | PR markdown template name |
| `docsPaths` | string[] | `[]` | Paths that count as documentation |

### Example

```json
{
  "commitStyle": "conventional",
  "requiredTrailers": ["Change-Type", "Ticket"],
  "optionalTrailers": ["Scope"],
  "branchPrefixes": ["feat/", "fix/", "chore/", "docs/"],
  "baseBranch": "main"
}
```

## Trailers

SmartGit uses git trailers for structured metadata in commit messages:

| Trailer | Values | Purpose |
|---|---|---|
| `Change-Type` | `feature`, `fix`, `refactor`, `docs`, `test`, `chore`, `perf` | Changelog grouping |
| `Scope` | Free-form (`auth`, `api`, `ui`, etc.) | Area of codebase affected |
| `Ticket` | Issue reference (`PROJ-123`, `#42`) | Link to tracking system |
| `Breaking-Change` | `yes` / description | Flag breaking changes |
| `Docs-Updated` | `yes` / `no` / `n/a` | Track whether docs were updated |

## License

[MIT](./LICENSE)
