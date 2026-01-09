# CLAUDE.md

## Project Overview

**worktree-dashboard** is a TUI tool for managing git worktrees with Claude Code integration. It displays worktrees, their git status, saved context files, and can open them in tmux windows with Claude Code auto-restoring context.

## Tech Stack

- **Language:** Go 1.25+
- **TUI Framework:** [bubbletea](https://github.com/charmbracelet/bubbletea) + [lipgloss](https://github.com/charmbracelet/lipgloss)
- **Build:** Standard Go toolchain

## Build Instructions

**Always build AND install after making changes:**
```bash
go build -o worktree-dashboard ./cmd/worktree-dashboard && go install ./cmd/worktree-dashboard
```

Never just build without installing - the user runs the installed binary.

Individual commands:
```bash
go build -o worktree-dashboard ./cmd/worktree-dashboard  # Build local binary
go install ./cmd/worktree-dashboard                       # Install to $GOPATH/bin
```

## Running

```bash
worktree-dashboard        # Interactive TUI
worktree-dashboard --list # Non-interactive list
```

## Project Structure

```
cmd/worktree-dashboard/main.go  # Entry point, CLI flags, hook subcommands
internal/
  worktree/worktree.go          # Git worktree scanning & status
  context/context.go            # Claude context file parsing
  sessions/sessions.go          # Additional sessions management (.claude/sessions)
  tmux/tmux.go                  # Tmux window management
  tui/tui.go                    # Bubbletea TUI model & view
  waiting/waiting.go            # State persistence (waiting sessions, server state)
```

## Architecture

### Worktree Package (`internal/worktree`)
- `List(repoPath)` - Parses `git worktree list --porcelain`
- `GetStatus(path)` - Returns `Status` with `[]FileStatus` (path + staged/unstaged indicators), counts, ahead/behind
- Extracts issue IDs (e.g., `SH-429`) from branch names or paths

### Context Package (`internal/context`)
- `Load(repoPath, issueID)` - Reads `.claude/contexts/{issueID}.md`
- `Exists(repoPath, issueID)` - Checks if context file exists
- Parses markdown sections: Title, Branch, Work Done, Key Files

### Sessions Package (`internal/sessions`)
- `List(projectRoot)` - Returns additional sessions from `.claude/sessions`
- `Add(projectRoot, path)` - Adds a path to sessions file
- `Remove(projectRoot, path)` - Removes a path from sessions file
- Sessions file is plain text, one path per line

### Tmux Package (`internal/tmux`)
- `OpenWorktree(issueID, path)` - Creates/switches to tmux window, runs `claude "/restore-context {issueID}"`
- `WindowExists(name)` - Checks for existing window to avoid duplicates
- `KillWindow(name)` - Kills a tmux window by name
- `ProjectRoot` - Global var set by TUI, used to set `WORKTREE_DASHBOARD_PROJECT` env var in new windows

### Waiting Package (`internal/waiting`)
- `MarkWaiting(projectRoot, cwd)` - Records a session as waiting for input
- `ClearWaiting(projectRoot, cwd)` - Clears waiting status
- `ListWaiting(projectRoot)` - Returns all paths with waiting sessions
- Stores state in `.claude/waiting/` directory

### TUI Package (`internal/tui`)
- Bubbletea model with items (worktrees) and sessionItems (additional sessions)
- Inline expandable file tree with git status indicators
- Keys: j/k navigate, l/→ expand, h/← collapse, Enter opens in tmux, a adds session, d deletes session, q quits
- Shows: issue ID, branch, file counts (+staged, ~modified, ?untracked), context status, waiting indicator

## Key Behaviors

- **Issue ID extraction:** Matches `SH-\d+` pattern in branch name or worktree path
- **Context detection:** Looks for `.claude/contexts/{issueID}.md` in repo root
- **Tmux integration:** Named windows by issue ID, reuses existing windows
- **Main worktree:** Shown as "(main)" when no issue ID found
- **Additional sessions:** Non-worktree paths stored in `.claude/sessions`, opens Claude in tmux
- **Waiting indicator:** Uses `WORKTREE_DASHBOARD_PROJECT` env var to route waiting state to correct project
- **Startup validation:** Requires `.claude` directory to exist in project root

## Hook Subcommands

The binary includes subcommands for Claude Code hooks:

- `worktree-dashboard notify-waiting` - Called by Stop hook, marks session as waiting
- `worktree-dashboard clear-waiting` - Called by PreToolUse hook, clears waiting status

Both read `WORKTREE_DASHBOARD_PROJECT` env var (set by tmux windows opened through dashboard) to determine which project's `.claude/waiting/` to update.
