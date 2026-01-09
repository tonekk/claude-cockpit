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
- 🔄 **Session management** - Add any directory as an additional session
- ⏳ **Waiting indicator** - Know when Claude is waiting for your input
- 🚀 **One-key tmux integration** - Jump into any session instantly
- 📝 **Context restoration** - Pick up exactly where you left off
- 🖥️ **Server splits** - Run your dev server in a tmux pane

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
| `o` | Open in your editor |
| `s` | Toggle server |
| `c` | Shell split |
| `a` | Add session |
| `d` | Delete session |
| `l` / `→` | Expand |
| `h` / `←` | Collapse |
| `j/k` | Navigate |
| `?` | Help |
| `q` | Quit |

## The Workflow

This tool shines when combined with some Claude Code custom commands:

1. **Start work**: `/branch SH-431` - creates a worktree from your issue tracker
2. **Do the thing**: hack away with Claude
3. **Take a break**: `/save-context` - saves your progress
4. **Come back**: Select worktree in cockpit → Enter → `/restore-context`

You'll need to set up these commands yourself (see [docs/claude-commands.md](docs/claude-commands.md)), but trust me, it's worth it.

## Sessions

Not everything is a worktree. Sometimes you're working on a completely different project. That's what sessions are for.

Press `a`, enter a path, and boom - it's in your cockpit, ready to launch. The tmux window opens immediately with Claude waiting for your command.

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

Now you'll see ⏳ next to any session that needs your attention. No more wondering "wait, did Claude finish?"

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
