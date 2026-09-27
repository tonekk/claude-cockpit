# claude-cockpit

Your command center for wrangling multiple Claude Code sessions. Because one Claude is never enough. 🛫

![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go&logoColor=white)

![claude-cockpit screenshot](screenshot.png)

## What's this?

A TUI dashboard for managing parallel Claude Code sessions across git worktrees and other projects. Think of it as air traffic control for your AI pair programming adventures.

**Opinionated by design.** This tool assumes you're:
- Using git worktrees (if you're not, you should be)
- Running everything in tmux (because tabs are for browsers)
- Working with Claude Code as your daily driver
- Possibly juggling multiple features/bugs at once without losing your mind

## Features

- 📂 **Worktree overview** - All your worktrees at a glance with git status
- 🟢🔴 **Claude status** - Green when Claude is working, red when waiting for input
- 🚀 **One-key tmux integration** - Jump into any worktree instantly
- 📝 **Context restoration** - Pick up exactly where you left off
- 🖥️ **Server window** - Run your dev server in a dedicated tmux window
- 🔌 **Server indicator** - See which worktree has a server running right in the tmux status bar
- 📊 **Diff view** - Open git diff in a split with one key

## Installation

```bash
go install github.com/tonekk/claude-cockpit/cmd/claude-cockpit@latest
```

Or the old-fashioned way:

```bash
git clone git@github.com:tonekk/claude-cockpit.git
cd claude-cockpit
go build -o claude-cockpit ./cmd/claude-cockpit
```

## Usage

```bash
# Fire it up (run from any git repo)
claude-cockpit

# With a server command (press 's' to toggle)
claude-cockpit -s "bin/rails server"

# With env vars for your server
claude-cockpit -s "npm run dev" -e "PORT=3000"

# Different editor (default: code)
claude-cockpit -E "cursor"
```

### Keybindings

| Key | What it does |
|-----|--------------|
| `Enter` | Open in tmux with Claude |
| `o` | Open in tmux (no Claude) |
| `e` | Open in your editor |
| `s` | Toggle server |
| `D` | Open git diff in split |
| `w` | Add worktree |
| `d` | Delete worktree |
| `c` | Config |
| `l` / `→` | Expand |
| `h` / `←` | Collapse |
| `j/k` | Navigate |
| `?` | Help |
| `q` | Quit |

## Waiting Indicator Setup

Want to know when Claude is waiting for you across all your sessions? Add these hooks to `~/.claude/settings.json`:

```json
{
  "hooks": {
    "Stop": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "claude-cockpit notify-waiting"
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
            "command": "claude-cockpit clear-waiting"
          }
        ]
      }
    ]
  }
}
```

Now your tmux window names show Claude's status: 🟢 when actively working, 🔴 when waiting for your input. No more wondering "wait, did Claude finish?"

## Environment Variables

| Variable | What for |
|----------|----------|
| `WD_SERVER_COMMAND` | Default server command |
| `WD_EDITOR` | Default editor |
| `WD_ENV_<KEY>` | Server env vars |

## Requirements

- Go 1.21+
- tmux (non-negotiable)
- Claude Code
- A love for terminal UIs

## License

MIT - do whatever you want with it.

---

*Built for developers who think "I'll just check on that other branch real quick" and end up with 12 Claude sessions.*
