# CLAUDE.md

## Project Overview

**claude-cockpit** is a TUI for managing multiple Claude Code sessions across git worktrees and other projects. Your command center for parallel AI pair programming.

## Tech Stack

- **Language:** Go 1.25+
- **TUI Framework:** [bubbletea](https://github.com/charmbracelet/bubbletea) + [lipgloss](https://github.com/charmbracelet/lipgloss)
- **Build:** Standard Go toolchain

## Build Instructions

**Always build AND install after all changes are finished:**
```bash
./rebuild.sh
```

Always rebuild when you are done making changes - don't wait for the user to ask.

Individual commands:
```bash
go build -o claude-cockpit ./cmd/claude-cockpit  # Build local binary
go install ./cmd/claude-cockpit                   # Install to $GOPATH/bin
```

## Running

```bash
claude-cockpit        # Interactive TUI
claude-cockpit --list # Non-interactive list
```

## Project Structure

```
cmd/claude-cockpit/main.go    # Entry point, CLI flags, hook subcommands
internal/
  worktree/worktree.go        # Git worktree scanning & status
  context/context.go          # Claude context file parsing
  tmux/tmux.go                # Tmux window management
  tui/tui.go                  # Bubbletea TUI model & view
  waiting/waiting.go          # State persistence (waiting sessions, server state)
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
- Bubbletea model with items (worktrees)
- Inline expandable file tree with git status indicators
- Keys: j/k navigate, l/→ expand, h/← collapse, Enter opens in tmux, x deletes worktree, q quits
- Shows: issue ID, branch, file counts (+staged, ~modified, ?untracked), context status, waiting indicator

## Key Behaviors

- **Issue ID extraction:** Matches `SH-\d+` pattern in branch name or worktree path
- **Context detection:** Looks for `.claude/contexts/{issueID}.md` in repo root
- **Tmux integration:** Named windows by issue ID, reuses existing windows
- **Main worktree:** Shown as "(main)" when no issue ID found
- **Waiting indicator:** Uses `WORKTREE_DASHBOARD_PROJECT` env var to route waiting state to correct project
- **Startup validation:** Requires `.claude` directory to exist in project root

## Hook Subcommands

The binary includes subcommands for Claude Code hooks:

- `claude-cockpit notify-waiting` - Called by Stop hook, marks session as waiting
- `claude-cockpit clear-waiting` - Called by PreToolUse hook, clears waiting status

Both read `WORKTREE_DASHBOARD_PROJECT` env var (set by tmux windows opened through cockpit) to determine which project's `.claude/waiting/` to update.
