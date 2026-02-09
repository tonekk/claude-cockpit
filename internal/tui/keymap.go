package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines keyboard shortcuts
type KeyMap struct {
	Up          key.Binding
	Down        key.Binding
	Enter       key.Binding
	Open        key.Binding
	OpenEditor  key.Binding
	Server      key.Binding
	Expand      key.Binding
	Collapse    key.Binding
	Help        key.Binding
	Config      key.Binding
	AddSession  key.Binding
	Delete      key.Binding
	Diff        key.Binding
	AddWorktree key.Binding
	Yes         key.Binding
	Quit        key.Binding
}

func (km KeyMap) AllBindings() []key.Binding {
	return []key.Binding{
		km.Up,
		km.Down,
		km.Enter,
		km.Open,
		km.OpenEditor,
		km.Server,
		km.Expand,
		km.Collapse,
		km.Help,
		km.Config,
		km.AddSession,
		km.Delete,
		km.Diff,
		km.AddWorktree,
		km.Yes,
		km.Quit,
	}
}

var keys = KeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "open in tmux"),
	),
	Open: key.NewBinding(
		key.WithKeys("o"),
		key.WithHelp("o", "open tmux (no claude)"),
	),
	OpenEditor: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "open in editor"),
	),
	Server: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "start/stop server"),
	),
	Expand: key.NewBinding(
		key.WithKeys("l", "right"),
		key.WithHelp("l/→", "expand"),
	),
	Collapse: key.NewBinding(
		key.WithKeys("h", "left"),
		key.WithHelp("h/←", "collapse"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	Config: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "config"),
	),
	AddSession: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "add session"),
	),
	Delete: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "delete worktree/session"),
	),
	Diff: key.NewBinding(
		key.WithKeys("D"),
		key.WithHelp("D", "open diff"),
	),
	AddWorktree: key.NewBinding(
		key.WithKeys("w"),
		key.WithHelp("w", "add worktree"),
	),
	Yes: key.NewBinding(
		key.WithKeys("y"),
		key.WithHelp("y", "Yes / confirm"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}
