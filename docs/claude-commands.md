# Claude Code Commands

These commands power the worktree-dashboard workflow. Copy them to your project's `.claude/commands/` directory.

---

## `/branch` - Create Worktree from Issue

**File:** `.claude/commands/branch.md`

```markdown
Create a new git worktree for a Linear issue.

## Arguments

$ARGUMENTS = Linear issue ID (e.g., `SH-402` or just `402`)

## Steps

### 1. Parse Issue ID

Extract the numeric ID from the argument. Both `SH-402` and `402` should work.

### 2. Fetch Linear Issue

Use the Linear MCP tools to fetch:

- Issue details: title, description, labels, and parent ID
- Issue comments: for additional context
- Parent issue (if exists): title, description, and labels

Check if the issue or its parent has a `Bug` label.

### 3. Determine Branch Type

- `bug/` — Issue has a "Bug" label
- `chore/` — Issue is devops, CI/CD, infrastructure, tooling, or explicitly maintenance work (look at title, description, labels)
- `feature/` — Everything else (default)

### 4. Generate Branch Name

Create a short, descriptive branch name in this format: `{type}/{issue_number}-{slug}`

Rules for the slug:

- Generate from full issue context (title, description, comments, and parent issue if exists), don't just slugify the title
- English only, even if the issue is in German
- Lowercase, words separated by dashes
- Max 4-5 words
- No special characters
- Examples: `add-comment-replies`, `fix-geocoding-provider`, `update-ci-parallelism`

### 5. Check Current Branch

If not on `main`:

- Ask user: "You're on `{current_branch}`. Branch off this branch or `main`?"
- Options: current branch, main

### 6. Setup Worktree

1. Ensure `.worktrees/` directory exists at repo root
2. Check if `.worktrees` is in `.gitignore`, add if not
3. Worktree path: `.worktrees/{issue_id}/` (e.g., `.worktrees/SH-402/`)

If worktree already exists:

- Just switch to it, skip creation

If branch already exists but no worktree:

- Create worktree using existing branch: `git worktree add .worktrees/{issue_id} {branch_name}`

If neither exists:

- Create new branch and worktree: `git worktree add -b {branch_name} .worktrees/{issue_id} {base_branch}`

### 7. Create Symlinks

In the new worktree, create symlinks to the main repo's untracked files:

```bash
# From inside the worktree directory
ln -s ../../.claude .claude
ln -s ../../.claude/CLAUDE.md CLAUDE.md
```

Skip if symlinks already exist.

### 8. Switch to Worktree

Change the working directory to the new worktree and confirm to the user:

```
Switched to worktree: .worktrees/{issue_id}/
Branch: {branch_name}
Base: {base_branch}
```

### 9. Enter plan mode

Enter plan mode. Ask the user for implementation details before suggesting features.
If they say `continue`, then just proceed.

## Error Handling

- If Linear issue not found: Show error, don't create anything
- If worktree creation fails: Show git error message
- If symlink fails: Continue anyway, warn user
```

---

## `/save-context` - Save Work Context

**File:** `.claude/commands/save-context.md`

```markdown
Save current work context for later restoration.

## Arguments

$ARGUMENTS = Linear issue ID with `SH-` prefix (e.g., `SH-431`). Required if not inferable from context.

## Steps

### 1. Determine Issue ID

The issue ID must always include the `SH-` prefix (e.g., `SH-343`).

Priority:
1. Use argument if provided (must be in format `SH-###`)
2. Extract from current branch name (e.g., `feature/431-product-request-comments` → `SH-431`)
3. **Ask user if neither available** - do not proceed without a valid issue ID

### 2. Gather Context

Use what's already in the conversation. Only fetch if missing:

- **Linear issue**: If not already discussed, fetch via Linear MCP tools
- **Figma links**: Extract from issue content if present
- **Work done**: Summarize from conversation, or analyze `git diff main...HEAD` if unclear

### 3. Write Context File

Create `.claude/contexts/{issue_id}.md`:

```markdown
# SH-{issue_id}: {issue_title}

## Branch

{current_git_branch_name}

## Linear Issue

{issue_description}

### Comments

{formatted_comments, if any}

## Figma

{list of figma links, or "None"}

## Work Done

{summary of implementation decisions and approach}

## Key Files

{list of files created or significantly modified}

## Steps

- [{timestamp}] {initial step describing work done so far, if any}
```

The `## Steps` section tracks progress. If work has been done, add an initial step summarizing it. Use format `- [YYYY-MM-DD HH:MM] message`.

### 4. Confirm

Tell the user the context was saved.
```

---

## `/restore-context` - Restore Work Context

**File:** `.claude/commands/restore-context.md`

```markdown
Restore saved work context for an issue.

## Arguments

$ARGUMENTS = Linear issue ID with `SH-` prefix (e.g., `SH-431`). Required if not inferable from context.

## Steps

### 1. Determine Issue ID

The issue ID must always include the `SH-` prefix (e.g., `SH-343`).

Priority:
1. Use argument if provided (must be in format `SH-###`)
2. Extract from current branch name (e.g., `feature/431-product-request-comments` → `SH-431`)
3. **Ask user if neither available** - do not proceed without a valid issue ID

### 2. Read Context File

Read `.claude/contexts/{issue_id}.md`

If file doesn't exist, tell the user no saved context was found.

### 3. Switch to Worktree (if applicable)

If the context file contains a `## Branch` section:
1. Run `git worktree list` to check if that branch is checked out in a worktree
2. If found, change the working directory to that worktree path
3. Inform the user which worktree you switched to

### 4. Display Context

Output the contents so they become part of the conversation.

If there are steps in the `## Steps` section, highlight the recent ones to show what was done previously.

Ask: "Context restored. What would you like to work on?"
```

---

## `/done` - Record Progress Step

**File:** `.claude/commands/done.md`

```markdown
Record a progress step for the current work context.

## Arguments

$ARGUMENTS = Optional issue ID with `SH-` prefix (e.g., `SH-431`). Only required if not inferable from context.

## Steps

### 1. Determine Issue ID

The issue ID must always include the `SH-` prefix (e.g., `SH-343`).

Priority:
1. Use argument if provided (must be in format `SH-###`)
2. Extract from current branch name (e.g., `feature/431-product-request-comments` → `SH-431`)
3. Extract from current working directory path (e.g., `.worktrees/SH-431/` → `SH-431`)
4. **Ask user if neither available** - do not proceed without a valid issue ID

### 2. Read Existing Context

Read `.claude/contexts/{issue_id}.md`

If file doesn't exist, tell the user no context file was found and offer to create one with `/save-context`.

### 3. Parse Existing Steps

Look for the `## Steps` section in the context file. Note the last step if any exist.

### 4. Compose New Step(s)

Based on the conversation since the last step (or since context was restored):

- Summarize what was accomplished as a **delta** from the previous step
- Keep each step concise (1 line, ~10-15 words)
- Focus on outcomes, not process ("Added user validation" not "Discussed and then implemented validation")
- If multiple distinct things were done, create multiple steps
- Use past tense verbs ("Added", "Fixed", "Implemented", "Refactored")

### 5. Append to Context File

Add steps to the `## Steps` section with timestamps:

```markdown
## Steps

- [2024-01-09 14:30] Added user authentication endpoint
- [2024-01-09 15:45] Implemented login form validation
- [2024-01-09 16:20] Fixed edge case with empty passwords
```

If the `## Steps` section doesn't exist, create it at the end of the file.

Use current timestamp in format `YYYY-MM-DD HH:MM`.

### 6. Confirm

Tell the user the step(s) were recorded.

## Example Usage

After implementing a feature:
- User: `/done`
- Claude: *Records "Implemented product request comments API"*

After fixing a bug:
- User: `/done`
- Claude: *Records "Fixed geocoding timeout issue"*

Multiple steps if a lot happened:
- Claude: *Records:*
  - *"Added comment model and migration"*
  - *"Implemented comments controller with CRUD"*
  - *"Added policy for comment authorization"*
```

---

## Customization

These commands are designed for Linear issue tracking with `SH-` prefixed IDs. To adapt for your workflow:

1. Change the issue ID pattern regex (`SH-\d+`) to match your system
2. Update the Linear MCP tool calls to your issue tracker
3. Adjust branch naming conventions as needed
