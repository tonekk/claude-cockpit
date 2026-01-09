# worktree-dashboard

A TUI tool for managing git worktrees with Claude Code integration. View all your worktrees at a glance with git status, saved context files, and one-key tmux integration.

![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go&logoColor=white)

![worktree-dashboard screenshot](screenshot.png)

## Features

- List all git worktrees with branch names
- Show git status with breakdown (+staged, ~modified, ?untracked)
- Inline expandable file tree with git status indicators (A, M, D, ?)
- Detect saved Claude context files per issue
- Display progress steps from context files (last 1 collapsed, last 5 expanded)
- Open worktrees in tmux with automatic context restoration
- Open worktrees in your editor (VS Code, etc.)
- Run server commands in tmux split pane (state persists across restarts)
- **Additional Sessions**: Add non-worktree paths to work on other projects in parallel
- **Waiting indicator**: Shows when Claude is waiting for input (⏳)
- Extract issue IDs from branch names (e.g., `feature/SH-431-foo` → `SH-431`)
- Per-project state stored in `.claude/` directory

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

# With server command (runs in tmux split when pressing 's')
worktree-dashboard -s "bin/rails server"

# With server environment variables
worktree-dashboard -s "bin/rails server" -e "RAILS_ENV=development" -e "PORT=3000"

# With custom editor (default: code)
worktree-dashboard -E "cursor"

# Non-interactive list
worktree-dashboard --list

# Show help
worktree-dashboard --help
```

### CLI Options

| Flag | Description |
|------|-------------|
| `-s, --server-command` | Server command to run in worktrees |
| `-e, --env` | Server env var (KEY=VALUE), can be repeated |
| `-E, --editor` | Editor command (default: `code`) |
| `--list` | List worktrees without TUI |
| `-h, --help` | Show help |

### Environment Variables

| Variable | Description |
|----------|-------------|
| `WD_SERVER_COMMAND` | Default server command |
| `WD_EDITOR` | Default editor command |
| `WD_ENV_<KEY>` | Server env vars (e.g., `WD_ENV_RAILS_ENV=development`) |

### Keybindings

| Key | Action |
|-----|--------|
| `Enter` | Open in tmux with Claude |
| `o` | Open in editor |
| `s` | Start/stop server |
| `c` | Open shell in vertical split |
| `a` | Add additional session |
| `d` | Delete additional session |
| `l` / `→` | Expand file tree |
| `h` / `←` | Collapse file tree |
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `C` | Show config |
| `?` | Show help |
| `q` | Quit |

## Required Claude Code Commands

For the full workflow, you need these custom commands in your project's `.claude/commands/` directory:

| Command | Purpose |
|---------|---------|
| `/branch {issue}` | Create a new worktree from a Linear issue |
| `/save-context` | Save current work context to `.claude/contexts/{issue}.md` |
| `/restore-context` | Restore saved context and switch to worktree |
| `/done` | Record progress step to context file |

See [docs/claude-commands.md](docs/claude-commands.md) for the full command definitions you can copy into your project.

## Workflow

1. **Start work**: `/branch SH-431` creates worktree + branch from Linear issue
2. **Do work**: Implement the feature with Claude Code
3. **Take a break**: `/save-context` saves your progress
4. **Resume later**: `worktree-dashboard` → select worktree → Enter opens tmux with `/restore-context`

## Additional Sessions

You can add paths to other projects (not worktrees of the current repo) to work on them in parallel without switching tmux sessions.

- Press `a` to add a session (enter the full path, e.g., `~/dev/other-project`)
- Press `d` to delete an additional session (also kills its tmux window if open)
- Sessions are stored in `.claude/sessions` (one path per line)

## Waiting Indicator Setup

To show the ⏳ indicator when Claude is waiting for input, add these hooks to your global Claude config at `~/.claude/settings.json`:

```json
{
  "hooks": {
    "Stop": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "worktree-dashboard notify-waiting"
          }
        ]
      }
    ],
    "PreToolUse": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "worktree-dashboard clear-waiting"
          }
        ]
      }
    ]
  }
}
```

These hooks work globally for all sessions opened through the dashboard, including additional sessions in other directories.

## How It Works

1. **Worktree scanning**: Parses `git worktree list --porcelain` to find all worktrees
2. **Issue ID extraction**: Matches `SH-\d+` pattern in branch names or paths
3. **Context detection**: Checks for `.claude/contexts/{issueID}.md` in the repo root
4. **Tmux integration**: Creates named windows by issue ID, runs `claude "/restore-context {issueID}"`
5. **State persistence**: Stores server state and waiting indicators in `.claude/` per project

## Project Structure

```
cmd/worktree-dashboard/main.go  # Entry point, CLI flags
internal/
  worktree/worktree.go          # Git worktree scanning & status
  context/context.go            # Claude context file parsing
  sessions/sessions.go          # Additional sessions management
  tmux/tmux.go                  # Tmux window management
  tui/tui.go                    # Bubbletea TUI model & view
  waiting/waiting.go            # State persistence (waiting sessions, server state)
```

## License

MIT
