# worktree-dashboard

A TUI tool for managing git worktrees with Claude Code integration. View all your worktrees at a glance with git status, saved context files, and one-key tmux integration.

![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go&logoColor=white)

## Features

- List all git worktrees with branch names
- Show git status with breakdown (+staged, ~modified, ?untracked)
- Inline expandable file tree with git status indicators (A, M, D, ?)
- Detect saved Claude context files per issue
- Open worktrees in tmux with automatic context restoration
- Extract issue IDs from branch names (e.g., `feature/SH-431-foo` → `SH-431`)

## Installation

```bash
go install github.com/tonekk/worktree-dashboard/cmd/worktree-dashboard@latest
```

Or build from source:

```bash
git clone https://github.com/tonekk/worktree-dashboard.git
cd worktree-dashboard
go build -o worktree-dashboard ./cmd/worktree-dashboard
```

## Usage

```bash
# Interactive TUI (run from any git repo with worktrees)
worktree-dashboard

# Non-interactive list
worktree-dashboard --list
```

### Keybindings

| Key | Action |
|-----|--------|
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `l` / `→` | Expand file tree |
| `h` / `←` | Collapse file tree |
| `Enter` | Open worktree in tmux with context restore |
| `q` | Quit |

## Required Claude Code Commands

For the full workflow, you need these custom commands in your project's `.claude/commands/` directory:

| Command | Purpose |
|---------|---------|
| `/branch {issue}` | Create a new worktree from a Linear issue |
| `/save-context` | Save current work context to `.claude/contexts/{issue}.md` |
| `/restore-context` | Restore saved context and switch to worktree |

See [docs/claude-commands.md](docs/claude-commands.md) for the full command definitions you can copy into your project.

## Workflow

1. **Start work**: `/branch SH-431` creates worktree + branch from Linear issue
2. **Do work**: Implement the feature with Claude Code
3. **Take a break**: `/save-context` saves your progress
4. **Resume later**: `worktree-dashboard` → select worktree → Enter opens tmux with `/restore-context`

## How It Works

1. **Worktree scanning**: Parses `git worktree list --porcelain` to find all worktrees
2. **Issue ID extraction**: Matches `SH-\d+` pattern in branch names or paths
3. **Context detection**: Checks for `.claude/contexts/{issueID}.md` in the repo root
4. **Tmux integration**: Creates named windows by issue ID, runs `claude "/restore-context {issueID}"`

## Project Structure

```
cmd/worktree-dashboard/main.go  # Entry point, CLI flags
internal/
  worktree/worktree.go          # Git worktree scanning & status
  context/context.go            # Claude context file parsing
  tmux/tmux.go                  # Tmux window management
  tui/tui.go                    # Bubbletea TUI model & view
```

## License

MIT
