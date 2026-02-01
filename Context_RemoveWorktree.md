# Context: Remove Worktrees Feature

## Feature Requirements
- Remove from TUI display
- Actually run `git worktree remove`
- Remove tmux tab if open
- Refuse to delete if dirty (uncommitted changes)
- Prevent deleting main worktree
- Use `d` key (same as sessions)
- Confirmation prompt (not yet implemented)

## Architecture Decisions
- `worktree.Remove(wt Worktree) error` lives in `internal/worktree/worktree.go`
- TUI calls the domain function - separation of concerns
- Git is the source of truth - after removal, TUI refreshes via `worktree.List()`
- Order of operations: check main -> check dirty -> kill tmux -> remove worktree

## Implementation Done

### `internal/worktree/worktree.go` - Remove function
```go
func Remove(wt Worktree) error {
    if wt.Branch == "main" {
        return errors.New("Can't remove main worktree")
    }

    if !wt.GitStatus.Clean {
        return errors.New("Can't remove dirty worktree")
    }

    if tmux.WindowExists(wt.Name) {
        tmux.KillWindow(wt.Name)
    }

    cmd := exec.Command("git", "worktree", "remove", wt.Path)

    return cmd.Run()
}
```

### `internal/tui/model.go` - Key handler (lines 310-318)
```go
} else {
    wt := m.items[m.cursor]

    if err := worktree.Remove(wt.Worktree); err != nil {
        panic(err)
    }

    return m, m.loadWorktrees
}
```

## Still TODO

### 1. Better Error Handling
Current code uses `panic(err)` which crashes the app. Should instead:
- Display error message in the TUI
- Look for `m.error` or similar pattern in the codebase

### 2. Confirmation Prompt
User wanted confirmation before deletion. Options:
- Use existing prompt system (like `m.prompts.activate()`)
- Add a simple y/n confirmation

### 3. Cursor Adjustment
After deleting the last item, cursor might point past the end of the list. See sessions deletion code for reference:
```go
// Adjust cursor if needed
if m.cursor >= len(m.sessionItems)-1 && m.cursor > 0 {
    m.cursor--
}
```

## Go Patterns Learned
- **Separation of concerns**: Domain logic in domain packages, UI in UI package
- **Source of truth**: Don't maintain in-memory state when external system (git) is the truth
- **Early returns**: Check error conditions first, return early
- **Error handling**: Return `error` from functions, don't panic in library code
- **Function signatures**: Pass structs that contain needed data rather than many individual params

## Key Files
- `internal/worktree/worktree.go` - Remove function lives here
- `internal/tui/model.go` - Key handler around line 287-318
- `internal/tmux/tmux.go` - WindowExists, KillWindow functions
