# CLAUDE.md

## Project Overview

**worktree-dashboard** is a TUI tool for managing git worktrees with Claude Code integration. It displays worktrees, their git status, saved context files, and can open them in tmux windows with Claude Code auto-restoring context.

## Tech Stack

- **Language:** Go 1.25+
- **TUI Framework:** [bubbletea](https://github.com/charmbracelet/bubbletea) + [lipgloss](https://github.com/charmbracelet/lipgloss)
- **Build:** Standard Go toolchain

## Build Instructions

**Always run after making changes:**
```bash
go build -o worktree-dashboard ./cmd/worktree-dashboard && go install ./cmd/worktree-dashboard
```

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
cmd/worktree-dashboard/main.go  # Entry point, CLI flags
internal/
  worktree/worktree.go          # Git worktree scanning & status
  context/context.go            # Claude context file parsing
  tmux/tmux.go                  # Tmux window management
  tui/tui.go                    # Bubbletea TUI model & view
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

### TUI Package (`internal/tui`)
- Bubbletea model with items (worktrees + context info)
- Inline expandable file tree with git status indicators
- Keys: j/k navigate, l/→ expand, h/← collapse, Enter opens in tmux, q quits
- Shows: issue ID, branch, file counts (+staged, ~modified, ?untracked), context status

## Key Behaviors

- **Issue ID extraction:** Matches `SH-\d+` pattern in branch name or worktree path
- **Context detection:** Looks for `.claude/contexts/{issueID}.md` in repo root
- **Tmux integration:** Named windows by issue ID, reuses existing windows
- **Main worktree:** Shown as "(main)" when no issue ID found
