# SmartGit — Consolidated Specification

## Problem

Working on multiple projects leads to broken traceability:

```
Intent → Branch → Commits → PR → Docs
```

Context gets lost between each step. Branch names are random, commit messages are bad, PRs lack descriptions, and mid-work you forget what a branch was even for.

## Philosophy

1. **Don't replace git. Augment it.** Plug into native git mechanisms (hooks, trailers, branch descriptions) so existing habits keep working.
2. **Git-first, AI-second.** Git metadata is the source of truth. AI handles synthesis and wording.
3. **Human-in-the-loop by default.** No silent auto-commits. Always preview and approve.
4. **Graceful degradation.** If the AI is unavailable, fall back to templates. Never block the developer.
5. **CLI-first, hooks later.** V1 is explicit CLI commands (`sg commit`, not a hidden hook). Hooks are opt-in after the core is proven. This keeps behavior transparent and avoids breaking existing repo workflows.

---

## CLI Commands

```
sg init                Set up hooks, trailers, aliases, config
sg start [desc]        Create branch with AI-generated name + description
sg commit [--all]      AI-powered commit with grouping and trailers
sg pr [--base main]    Generate and create PR from branch history
sg status              Show traceability breadcrumb trail
sg doctor              Check for missing plan/trailers/docs
```

### `sg init`

One-time setup (global or per-repo):
- Configure `core.hooksPath` pointing to SmartGit hooks
- Install `prepare-commit-msg`, `commit-msg`, `post-checkout`, `pre-push` hooks
- Chain to existing project hooks if present (rename originals, call them after SmartGit hooks)
- Configure trailer aliases (`Change-Type`, `Scope`, `Ticket`)
- Set `merge.branchdesc = true`
- Set `commit.cleanup = whitespace`
- Optionally set `init.templateDir` for future repos
- Store API key in user-level config

### `sg start`

1. Accept a description (or prompt interactively for goal, ticket, context)
2. Clean the input (strip conversational fluff: "Can you please..." → imperative form)
3. AI generates a conventional branch name: `feat/oauth2-google-login`, `fix/pagination-null-pointer`
4. Validate branch name: `git check-ref-format --branch <name>`
5. Create and switch: `git switch -c <name>`
6. Write branch description via `git config branch.<name>.description` with:
   - What the branch is for
   - Ticket/issue reference
   - Acceptance criteria or plan
7. Save local session file (`.sg/session.json`) with plan and acceptance criteria

### `sg commit`

Two operating modes:

**Hook mode** (zero habit change): plain `git commit` triggers the `prepare-commit-msg` hook, which calls the AI and pre-fills the message. The developer sees the suggestion in their editor.

**CLI mode** (`sg commit`): full control with interactive approval.

CLI mode workflow:
1. Read staged diff (`git diff --cached`), warn if unstaged changes exist (`git status --porcelain=v2`)
2. Read branch description for context
3. Send diff + context to AI with a structured JSON schema for output
4. AI proposes one or more commit groups (by concern: backend/api/tests/docs)
5. For each group, display:
   - Files in the group
   - Proposed commit message (subject + body, "why" not just "what")
   - Proposed trailers
6. User approves, edits, or skips each group
7. For each approved group: stage files, commit with message and trailers
8. If touching the same area as a previous commit, suggest `--fixup` instead

Flags:
- `--all`: include unstaged changes
- `--dry-run`: preview without executing any commits
- `--no-split`: force a single commit for all changes

### `sg pr`

1. Find divergence point: `git merge-base HEAD <base>`
2. Read branch description: `git config branch.<current>.description`
3. Read all commits + trailers: `git log <base>..HEAD --format`
4. Read diff stats: `git diff --stat <base>..HEAD`, `git diff --dirstat <base>..HEAD`
5. Generate grouped changelog: `git shortlog --group=trailer:Change-Type <base>..HEAD`
6. Optionally use `git request-pull` as a baseline template
7. Send everything to AI to generate PR markdown:
   - Summary
   - What changed and why
   - Risks
   - Test plan
   - Docs impact checklist
   - Breaking changes (from trailers)
8. User reviews and edits the generated markdown
9. Create PR via `gh pr create` with generated title and body

Flags:
- `--base <branch>`: specify base branch (default: `main`)
- `--dry-run`: preview PR body without creating it
- `--changelog`: also generate/update CHANGELOG.md

### `sg status`

Shows the full traceability breadcrumb:
- Current branch name + description (the original intent)
- Commits since fork point (with trailers)
- Staged/unstaged changes
- Remaining plan items from `.sg/session.json` (if present)
- Readiness flags: has description, has trailers, has docs updates

### `sg doctor`

Diagnostic check for traceability completeness:
- Branch has a description?
- All commits have `Change-Type` trailers?
- Ticket trailer present?
- Docs updated flag set?
- Session plan items all addressed?

Reports missing items as a checklist.

---

## Trailers

| Trailer | Values | Purpose |
|---|---|---|
| `Change-Type` | `feature`, `fix`, `refactor`, `docs`, `test`, `chore`, `perf` | Changelog grouping |
| `Scope` | Free-form (`auth`, `api`, `ui`, `database`) | Area of codebase affected |
| `Ticket` | Issue reference (`PROJ-123`, `#42`) | Link to tracking system |
| `Breaking-Change` | `yes` / description | Flag breaking changes |
| `Docs-Updated` | `yes` / `no` / `n/a` | Track whether docs were updated alongside code |

Added automatically by the `commit-msg` hook or `sg commit`. Can also be auto-extracted:
- `Ticket` from branch name via `trailer.ticket.cmd`
- `Change-Type` detected by AI from diff analysis
- Queryable: `git log --format="%(trailers:key=Change-Type,valueonly)"`
- Groupable: `git shortlog --group=trailer:Change-Type main..HEAD`

Trailers survive `git rebase`, `git cherry-pick`, push, and clone. They are visible on GitHub.

---

## Hook Architecture (Phase 4 — Optional)

### `prepare-commit-msg`

```
1. Check if SmartGit is enabled (env var or config)
2. Skip if message was provided via -m flag (source = "message")
3. Read staged diff: git diff --cached
4. Read branch description: git config branch.<current>.description
5. Read recent commits for context: git log --oneline -5
6. Call AI (with timeout) to generate commit message
7. On success: write AI message to the commit message file
8. On failure: log error, leave message empty (graceful fallback)
9. User sees result in their editor, can accept/edit/reject
10. Chain to original prepare-commit-msg hook if it exists
```

### `commit-msg`

```
1. Read the final commit message
2. Validate conventional commit format (warn, don't block)
3. Auto-add missing trailers via git interpret-trailers --in-place
4. Extract ticket from branch name if Ticket trailer is missing
5. Chain to original commit-msg hook if it exists
```

### `post-checkout`

```
1. Check if this is a branch checkout (flag = 1)
2. Check if the branch has a description
3. If new branch without description, suggest running sg start
4. Chain to original post-checkout hook if it exists
```

### `pre-push`

```
1. Read commits being pushed (from stdin)
2. Warn about commits without Change-Type trailers
3. Warn about missing branch description
4. Chain to original pre-push hook if it exists
```

All hooks follow the same principles:
- Never block the developer on AI failure
- Always chain to original hooks
- Skippable via `--no-verify` (standard git behavior)

---

## Git Commands Reference

### Branch setup
- `git switch -c <branch>` — create and switch
- `git check-ref-format --branch <name>` — validate branch name
- `git branch --edit-description` — interactive description editor
- `git config branch.<name>.description "..."` — non-interactive description

### Diff analysis (inputs for AI)
- `git status --porcelain=v2 -z` — machine-readable status
- `git diff --cached` — staged changes (full diff)
- `git diff --name-status -M -C` — file list with rename/copy detection
- `git diff --stat` — human-readable summary
- `git diff --numstat` — machine-parseable stats
- `git diff --dirstat` — per-directory change distribution

### Commit structure
- `git commit -F <file>` — message from file
- `git commit --trailer "Key: Value"` — add trailer inline
- `git commit --fixup=<sha>` — create fixup commit
- `git commit --fixup=reword:<sha>` — reword a previous commit message
- `git interpret-trailers --in-place` — inject trailers into message file
- `git stripspace` — clean up AI-generated text

### PR range analysis
- `git merge-base <base> HEAD` — find divergence point
- `git log <base>..HEAD --format=...` — commits with custom format
- `git shortlog <base>..HEAD` — grouped commit summary
- `git shortlog --group=trailer:Change-Type <base>..HEAD` — changelog by type
- `git request-pull <start> <url>` — baseline PR summary
- `git range-diff` — compare patch series versions (for PR updates)
- `git describe` — version context from nearest tag

### Rework support
- `git commit --fixup <sha>` — fix a previous commit
- `git rebase -i --autosquash` — apply fixups
- `git stash push --staged` — temporarily stash for split workflows

### Config leveraged
- `core.hooksPath` — global hooks directory
- `commit.cleanup = whitespace` — preserve structured content
- `merge.branchdesc = true` — branch descriptions in merge commits
- `trailer.<alias>.key` / `trailer.<alias>.cmd` — auto-generated trailers
- `init.templateDir` — auto-install hooks in new repos
- `includeIf "onbranch:..."` — conditional config per branch pattern
- `rebase.autoSquash = true` — auto-apply fixup commits

---

## Configuration

Three layers, merged field-by-field (later overrides earlier):

| Layer | Location | Contains | Committed? |
|---|---|---|---|
| Global (user) | `~/.config/sg/config.json` | Defaults (diffMaxLines) | No |
| Project (team) | `.sg/config.json` | Branch rules, trailer requirements, PR template, commit style | Yes |
| Project (personal) | `.sg/config.local.json` | Personal overrides | No (gitignored) |

Example project config:
```json
{
  "commitStyle": "conventional",
  "requiredTrailers": ["Change-Type"],
  "optionalTrailers": ["Scope", "Ticket"],
  "branchPrefixes": ["feat", "fix", "chore", "refactor", "docs", "test"],
  "prTemplate": "default",
  "baseBranch": "main"
}
```

Example global config:
```json
{
  "diffMaxLines": 2000
}
```

Note: AI auth and model selection are handled by the Claude Code CLI (`claude`). No API key configuration needed in `sg`.

Full config key reference:

| Key | Layer | Purpose |
|---|---|---|
| `commitStyle` | Project | Commit convention (`conventional`) |
| `requiredTrailers` | Project | Trailers that must be present |
| `optionalTrailers` | Project | Trailers suggested but not enforced |
| `branchPrefixes` | Project | Allowed branch name prefixes |
| `baseBranch` | Project | Default base branch for PR/diff |
| `prTemplate` | Project | PR markdown template name |
| `docsPaths` | Project | Paths that count as documentation (for `Docs-Updated` trailer) |
| `diffMaxLines` | Global | Max diff lines sent to AI |

---

## AI Integration

### Structured Output
All AI calls use a JSON schema contract for responses. This prevents low-quality freeform output.

Example schema for commit grouping:
```json
{
  "groups": [
    {
      "files": ["src/auth/login.ts", "src/auth/token.ts"],
      "subject": "feat(auth): add OAuth2 refresh token flow",
      "body": "Implements automatic token refresh when access token expires...",
      "changeType": "feature",
      "scope": "auth"
    }
  ]
}
```

### Inputs to AI
For every call, we provide machine-readable git data:
- `git diff --cached` (the actual changes)
- `git diff --name-status -M -C` (file-level overview with renames)
- `git diff --stat` (summary stats)
- Branch description (the intent)
- Recent commit history (for consistency)

### Fallback Behavior
- API timeout: 15 seconds (configurable)
- On failure: log the error, proceed without AI (empty message in editor, or template-based)
- Never block the developer
- Rate limit handling: queue and retry with backoff, or skip

---

## Risks and Mitigations

| Risk | Mitigation |
|---|---|
| Incorrect file grouping across concerns | User approval required for each group before commit |
| Overly verbose or inaccurate commit messages | Preview and edit step, structured JSON output |
| Cost/latency on large diffs | Max diff size limit (configurable), truncate with `--stat` summary |
| Sensitive code sent to external API | Ignore-path rules in config, redact patterns, local model option for future |
| Hook conflicts with existing project hooks | Hook chaining (detect, rename, call originals) |
| `core.hooksPath` overrides all repos | Per-repo opt-out via `SG_DISABLED=1` env var |
| Branch descriptions are local-only | Mirror critical context into PR text (the whole point of `sg pr`) |

---

## Technical Stack

| Component | Choice | Rationale |
|---|---|---|
| Language | Go | Single binary, no runtime dependency, fast startup |
| AI Provider | Claude Code CLI (`claude --print`) | Leverages existing auth, structured JSON output via `--json-schema` |
| Git interaction | `os/exec` shelling out to `git` CLI | Always available, no library needed |
| CLI framework | `github.com/spf13/cobra` | De facto standard (kubectl, docker, gh), subcommand support |
| Distribution | Single binary via `go install` or GitHub releases | Zero-dependency install |
| Hooks | Shell scripts calling `sg` binary | Chain-friendly, no runtime assumptions |

---

## Implementation Phases

### Phase 1 — MVP: `sg commit`
- Project scaffolding (Go module, cobra CLI)
- Config loading (global + project + local)
- Git module (shell out, parse diff/status/log)
- AI module (Anthropic API, structured JSON output, timeout/fallback)
- `sg commit` command (full interactive flow with approval)
- Fallback template mode when AI is unavailable

Exit criteria: consistently readable commit history, no workflow blocking on AI issues.

Ship this. If it's reliable, continue.

### Phase 2 — `sg start` + `sg status`
- `sg start` command (branch creation + description)
- `sg status` command (breadcrumb trail)
- `.sg/session.json` session file
- `sg init` command (hook installation, config setup)

Exit criteria: branch intent and acceptance criteria are visible during work.

### Phase 3 — `sg pr`
- PR description generation from branch history
- `gh` CLI integration
- `sg doctor` command

Exit criteria: PR drafts are usable with minimal manual rewriting.

### Phase 4 — Hooks + Polish
- Optional hooks (`prepare-commit-msg`, `commit-msg`, `post-checkout`, `pre-push`)
- Changelog generation from trailers
- `--fixup` suggestions
- Commit splitting improvements
- Per-repo config and overrides
- `init.templateDir` setup

---

## Quality Gates

### Before a commit is accepted
1. Message matches the selected convention (conventional commits)
2. Required trailers are present (configurable per project)
3. Files in a group are semantically coherent (user confirms)

### Before PR markdown is ready
1. Has summary, risks, test plan, and docs impact sections
2. Includes ticket/intent context from branch description
3. Covers all commits in the range

---

## Success Metrics

1. Percentage of commits with required trailers present
2. Percentage of AI-suggested commit messages accepted without rewrite
3. Time from "ready to push" to final PR description
4. Percentage of PRs with docs checklist completed
5. User-reported trust in suggested commit grouping

---

## Lessons From `entireio/cli`

Reviewed [entireio/cli](https://github.com/entireio/cli), a Go tool for recording AI coding sessions. Very different scope (audit trail), much heavier machinery. Five ideas adopted:

1. **Clean user input** — strip conversational fluff before generating branch names or messages
2. **Two-tier config** — team-level (committed) + personal (gitignored), merged field-by-field
3. **Trailers are resilient** — they survive rebase and cherry-pick, making them the right metadata store
4. **Graceful AI fallback** — never block the developer, log errors silently
5. **Hook chaining** — detect and call original project hooks after SmartGit hooks

Skipped: shadow branches, session recording, strategy pattern, agent abstraction, checkpoint/rewind, attribution tracking, transcript chunking, session state machines.

---

## Resolved Decisions

| Question | Decision | Rationale |
|---|---|---|
| Hook-first or CLI-first? | **CLI-first for V1**, hooks in Phase 4 | Faster to build, more transparent, no risk of breaking existing workflows |
| Hooks call AI directly or delegate? | **Delegate to `sg` binary** | Single AI integration point, simpler hook scripts |
| Max automation or human control? | **Human approval required** in V1 | Commit quality and trust are the immediate goals |
| Local metadata or portable? | **Both** — branch descriptions locally, trailers in commits | Branch descriptions for local context, trailers survive push/rebase |
| Rich architecture or small product? | **Small product first** | Defer `core.hooksPath`, `templateDir`, `includeIf`, `notes` to post-MVP |

## Open Questions

1. **AI provider**: Anthropic only for V1, or abstract from the start?
2. **Package name**: `sg`, `smartgit`, `git-smart`? (Need to check Go module name availability)
3. **Rate limits**: how to handle heavy committers hitting API limits?
4. **Local model support**: worth planning the abstraction now for future Ollama/local model support?
