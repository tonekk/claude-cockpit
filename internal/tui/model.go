package tui

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/tonekk/claude-cockpit/internal/config"
	"github.com/tonekk/claude-cockpit/internal/tmux"
	"github.com/tonekk/claude-cockpit/internal/waiting"
	"github.com/tonekk/claude-cockpit/internal/worktree"
)

// Options holds the TUI runtime options (from CLI flags)
type Options struct {
	ServerCommand string
	ServerEnv     map[string]string
	Editor        string
}

// WorktreeItem represents a worktree in the list
type WorktreeItem struct {
	Worktree worktree.Worktree
	Expanded bool
}

// Model is the bubbletea model
type Model struct {
	worktreeItems      []WorktreeItem
	cursor             int
	projectRoot        string
	width              int
	height             int
	err                error
	waitingSessions map[string]bool   // paths with sessions waiting for input
	tmuxWindows     map[string]bool   // paths with tmux windows open
	claudeStatus    map[string]string // "running", "waiting", or "" (no claude)
	options            Options
	config             config.Config
	serverWorktreePath string // path of worktree where server is running
	errorMessage       string // error message to show in popup
	showHelp           bool   // show help popup
	showConfig         bool   // show config popup
	prompts            PromptManager
	newWorktreeDir     string
	showConfirm        bool
	confirmHeader      string
	confirmMessage     string
	confirmHandler     func() (func() tea.Msg, error)
	confirmCancel      func() (func() tea.Msg, error) // optional, runs on any key other than y
	tickCount          int // counts ticks for periodic git status refresh
}

// New creates a new TUI model
func New(projectRoot string, options Options, config config.Config) Model {
	return Model{
		projectRoot: projectRoot,
		options:     options,
		config:      config,
		prompts:     NewPromptManager(),
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.loadWorktrees,
		m.loadServerState,
		tea.Tick(500 * time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) }),
	)
}

func (m Model) loadServerState() tea.Msg {
	if windowName, workDir, found := tmux.FindServerWindow(); found {
		tmux.ServerWindowName = windowName
		return serverStateMsg{
			worktreePath: workDir,
		}
	}
	return nil
}

func (m Model) loadWorktrees() tea.Msg {
	worktrees, err := worktree.List(m.projectRoot)
	if err != nil {
		return errMsg{err}
	}

	var items []WorktreeItem
	for _, wt := range worktrees {
		item := WorktreeItem{Worktree: wt}

		// Load git status
		status, _ := worktree.GetStatus(wt.Path)
		item.Worktree.GitStatus = status

		items = append(items, item)
	}

	return worktreesMsg{items}
}

type worktreesMsg struct {
	items []WorktreeItem
}

type errMsg struct {
	err error
}

type tickMsg time.Time

type serverStateMsg struct {
	worktreePath string
}

// getCurrentPath returns the path of the currently selected item
func (m Model) getCurrentPath() string {
	if m.cursor < len(m.worktreeItems) {
		return m.worktreeItems[m.cursor].Worktree.Path
	}
	return ""
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Always handle ticks and window size, even during prompt input
	switch msg := msg.(type) {
	case tickMsg:
		return m.handleTickMsg()
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// fall through to prompt handling below
	}

	// Handle prompt input mode
	if m.prompts.hasActive() {
		return m.updatePrompt(msg)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// already handled above

	case worktreesMsg:
		m.worktreeItems = msg.items

		// Reset to last worktree item if all were deleted
		if m.cursor > len(m.worktreeItems)-1 {
			m.cursor = len(m.worktreeItems) - 1
		}

	case errMsg:
		m.err = msg.err

	case serverStateMsg:
		m.serverWorktreePath = msg.worktreePath

	case tickMsg:
		return m.handleTickMsg()

	case tea.KeyMsg:
		return m.handleKeyMessage(msg)
	}

	return m, nil
}

func (m Model) handleTickMsg() (tea.Model, tea.Cmd) {
	m.tickCount++

	// Refresh git status every 4 ticks (~2s)
	if m.tickCount%4 == 0 {
		for i := range m.worktreeItems {
			status, _ := worktree.GetStatus(m.worktreeItems[i].Worktree.Path)
			m.worktreeItems[i].Worktree.GitStatus = status
		}
	}

	// Poll for waiting sessions and clean up stale ones
	paths, _ := waiting.ListWaiting(m.projectRoot)
	m.waitingSessions = make(map[string]bool)
	for _, p := range paths {
		name := filepath.Base(p)
		if _, found := tmux.FindWindowByName(name); !found {
			// Window gone — clear stale waiting state
			waiting.ClearWaiting(m.projectRoot, p)
		} else if !tmux.IsClaudeRunningInWindow(name) {
			// Window exists but Claude exited — clear waiting state
			waiting.ClearWaiting(m.projectRoot, p)
		} else {
			m.waitingSessions[p] = true
		}
	}

	// Check if server window was killed externally
	if m.serverWorktreePath != "" && !tmux.IsServerRunning() {
		m.serverWorktreePath = ""
	}

	// Check tmux window and Claude status for all items
	m.tmuxWindows = make(map[string]bool)
	m.claudeStatus = make(map[string]string)
	for _, item := range m.worktreeItems {
		name := item.Worktree.Name
		path := item.Worktree.Path
		if _, found := tmux.FindWindowByName(name); found {
			m.tmuxWindows[path] = true
			if tmux.IsClaudeRunningInWindow(name) {
				if m.waitingSessions[path] {
					m.claudeStatus[path] = "waiting"
				} else {
					m.claudeStatus[path] = "running"
					tmux.SetWindowStatus(name, "running")
				}
			} else {
				tmux.SetWindowStatus(name, "")
			}
		}
	}

	// Continue polling
	return m, tea.Tick(500 * time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// claudeWindowStatus returns "running", "waiting", or "" based on Claude's state in a window.
func claudeWindowStatus(windowName string, isWaiting bool) string {
	if !tmux.IsClaudeRunningInWindow(windowName) {
		return ""
	}
	if isWaiting {
		return "waiting"
	}
	return "running"
}

func (m Model) handleKeyMessage(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// If error popup is showing, dismiss on any key
	if m.errorMessage != "" {
		m.errorMessage = ""
		return m, tea.ClearScreen
	}

	// If help popup is showing, close it but continue processing the key
	if m.showHelp {
		m.showHelp = false
	}

	// If config popup is showing, close it but continue processing the key
	if m.showConfig {
		m.showConfig = false
	}

	if m.showConfirm {
		m.showConfirm = false
		m.confirmHeader = ""
		m.confirmMessage = ""

		run := m.confirmCancel
		if key.Matches(msg, keys.Yes) {
			run = m.confirmHandler
		}
		m.confirmHandler = nil
		m.confirmCancel = nil

		if run == nil {
			return m, tea.ClearScreen
		}

		handler, err := run()
		if err != nil {
			m.errorMessage = err.Error()
			return m, nil
		}

		return m, tea.Batch(handler, tea.ClearScreen)
	}

	switch {
	case key.Matches(msg, keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, keys.Help):
		m.showHelp = !m.showHelp
		return m, nil

	case key.Matches(msg, keys.Config):
		m.showConfig = !m.showConfig
		return m, nil

	case key.Matches(msg, keys.Delete):
		m.showConfirm = true

		wt := m.worktreeItems[m.cursor]
		m.confirmHeader = "Remove Worktree"
		m.confirmMessage = "Remove " + confirmTargetStyle.Render(wt.Worktree.Branch) + "?"
		m.confirmHandler = m.removeSelectedWorktree

		return m, nil

	case key.Matches(msg, keys.ForceDelete):
		if m.cursor < len(m.worktreeItems) {
			wt := m.worktreeItems[m.cursor]
			if wt.Worktree.Branch == "main" {
				m.errorMessage = "Can't remove main worktree"
				return m, nil
			}
			m.showConfirm = true
			m.confirmHeader = "Force Remove Worktree"
			m.confirmMessage = "Discard all changes and remove " + confirmTargetStyle.Render(wt.Worktree.Branch) + "?"
			m.confirmHandler = m.forceRemoveSelectedWorktree
		}
		return m, nil

	case key.Matches(msg, keys.AddWorktree):
		m.prompts.activate("worktreeDir")
		return m, textinput.Blink

	case key.Matches(msg, keys.Expand):
		if m.cursor < len(m.worktreeItems) {
			m.worktreeItems[m.cursor].Expanded = true
		}

	case key.Matches(msg, keys.Collapse):
		if m.cursor < len(m.worktreeItems) {
			m.worktreeItems[m.cursor].Expanded = false
		}

	case key.Matches(msg, keys.Up):
		if m.cursor > 0 {
			m.cursor--
		}

	case key.Matches(msg, keys.Down):
		if m.cursor < len(m.worktreeItems)-1 {
			m.cursor++
		}

	case key.Matches(msg, keys.Enter):
		if m.cursor < len(m.worktreeItems) {
			item := m.worktreeItems[m.cursor]
			name, path := item.Worktree.Name, item.Worktree.Path

			// Window already open, or nothing to set up: just open it
			if _, open := tmux.FindWindowByName(name); open || len(m.config.Setup) == 0 {
				if isNew, _ := tmux.OpenWorktree(name, path); isNew {
					m.config.SendSetupAndClaudeToTmux("claude")
				}
				return m, nil
			}

			// Existing worktree, new window: ask whether to run setup first
			m.showConfirm = true
			m.confirmHeader = "Run Setup"
			m.confirmMessage = "Run setup commands from cockpit.yml in " + confirmTargetStyle.Render(item.Worktree.Branch) + "?"
			m.confirmHandler = func() (func() tea.Msg, error) {
				tmux.OpenWorktree(name, path)
				m.config.SendSetupAndClaudeToTmux("claude")
				return nil, nil
			}
			m.confirmCancel = func() (func() tea.Msg, error) {
				tmux.OpenWorktree(name, path)
				tmux.SendKeys("claude")
				return nil, nil
			}
			return m, nil
		}

	case key.Matches(msg, keys.OpenEditor):
		path := m.getCurrentPath()
		if path != "" {
			return m, openEditor(path, m.options.Editor)
		}

	case key.Matches(msg, keys.Server):
		if m.cursor < len(m.worktreeItems) {
			item := m.worktreeItems[m.cursor]

			// Check if server is running in this worktree - toggle off
			if m.serverWorktreePath == item.Worktree.Path {
				tmux.StopServerWindow()
				m.serverWorktreePath = ""
				return m, nil
			}

			// Check if server command is configured
			if m.options.ServerCommand == "" {
				m.errorMessage = "Error: No server command configured.\nUse -s or --server-command flag, or set server: in cockpit.yml"
				return m, nil
			}

			// Start server (will auto-stop existing one)
			_, err := tmux.StartServerWindow(item.Worktree.Name, item.Worktree.Path, m.options.ServerCommand, m.options.ServerEnv)
			if err != nil {
				m.errorMessage = fmt.Sprintf("Error starting server: %v", err)
				return m, nil
			}
			m.serverWorktreePath = item.Worktree.Path
		}

	case key.Matches(msg, keys.Open):
		if m.cursor < len(m.worktreeItems) {
			item := m.worktreeItems[m.cursor]
			tmux.OpenWorktree(item.Worktree.Name, item.Worktree.Path)
		}

	case key.Matches(msg, keys.Diff):
		path := m.getCurrentPath()
		if path != "" {
			tmux.OpenDiff(path, false)
		}

	case key.Matches(msg, keys.DiffStaged):
		path := m.getCurrentPath()
		if path != "" {
			tmux.OpenDiff(path, true)
		}
	}

	return m, nil
}

func (m Model) updatePrompt(msg tea.Msg) (tea.Model, tea.Cmd) {
	prompt, _ := m.prompts.active()

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			value := prompt.input.Value()
			if value == "" {
				return m, nil
			}
			return prompt.onSubmit(m, value)

		case "esc", "ctrl+c":
			m.prompts.deactivate()
			return m, nil
		}
	}

	var cmd tea.Cmd
	prompt.input, cmd = prompt.input.Update(msg)
	return m, cmd
}

// openEditor opens the configured editor in the given path
func openEditor(path, editor string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command(editor, path)
		_ = cmd.Start()
		return nil
	}
}

func (m Model) removeSelectedWorktree() (func() tea.Msg, error) {
	wt := m.worktreeItems[m.cursor]
	if err := worktree.Remove(wt.Worktree); err != nil {
		return nil, err
	}
	return m.loadWorktrees, nil
}

func (m Model) forceRemoveSelectedWorktree() (func() tea.Msg, error) {
	wt := m.worktreeItems[m.cursor]
	if err := worktree.ForceRemove(wt.Worktree); err != nil {
		return nil, err
	}
	return m.loadWorktrees, nil
}
