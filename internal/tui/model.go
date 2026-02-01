package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/tonekk/claude-cockpit/internal/config"
	"github.com/tonekk/claude-cockpit/internal/context"
	"github.com/tonekk/claude-cockpit/internal/sessions"
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

// Item represents a worktree with its context info
type Item struct {
	Worktree   worktree.Worktree
	HasContext bool
	Context    *context.Context
	Expanded   bool
}

// SessionItem represents an additional session (non-worktree)
type SessionItem struct {
	Session   sessions.Session
	TmuxOpen  bool // whether tmux window is currently open
	Expanded  bool
	GitStatus *worktree.Status // git status if it's a git repo
}

// Model is the bubbletea model
type Model struct {
	items              []Item
	sessionItems       []SessionItem
	cursor             int
	inSessionsSection  bool // true when cursor is in additional sessions section
	projectRoot        string
	width              int
	height             int
	err                error
	waitingSessions    map[string]bool // paths with sessions waiting for input
	options            Options
	config             config.Config
	serverWorktreePath string // path of worktree where server is running
	errorMessage       string // error message to show in popup
	showHelp           bool   // show help popup
	showConfig         bool   // show config popup
	prompts            PromptManager
	newWorktreeDir     string
	showConfirm        bool
	confirmMessage     string
	confirmHandler     func() (func() tea.Msg, error)
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
		m.loadSessions,
		m.loadServerState,
		tea.Tick(2*time.Second, func(t time.Time) tea.Msg { return tickMsg(t) }),
	)
}

func (m Model) loadServerState() tea.Msg {
	state, err := waiting.LoadServerState(m.projectRoot)
	if err != nil || state == nil {
		return nil
	}

	// Verify the pane still exists
	if !tmux.PaneExists(state.PaneID) {
		waiting.ClearServerState(m.projectRoot)
		return nil
	}

	// Restore state
	tmux.ServerPaneID = state.PaneID
	return serverStateMsg{
		worktreePath: state.WorktreePath,
		paneID:       state.PaneID,
	}
}

func (m Model) loadWorktrees() tea.Msg {
	worktrees, err := worktree.List(m.projectRoot)
	if err != nil {
		return errMsg{err}
	}

	var items []Item
	for _, wt := range worktrees {
		item := Item{Worktree: wt}

		// Load git status
		status, _ := worktree.GetStatus(wt.Path)
		item.Worktree.GitStatus = status

		// Check for context file
		item.HasContext = context.Exists(m.projectRoot, wt.Name)
		if item.HasContext {
			ctx, _ := context.Load(m.projectRoot, wt.Name)
			item.Context = ctx
		}

		items = append(items, item)
	}

	return itemsMsg{items}
}

func (m Model) loadSessions() tea.Msg {
	sessionList, err := sessions.List(m.projectRoot)
	if err != nil {
		return nil // Ignore errors, just don't show sessions
	}

	var items []SessionItem
	for _, s := range sessionList {
		item := SessionItem{Session: s}

		// Check if tmux window is open
		_, item.TmuxOpen = tmux.FindWindowByName(s.Name)

		// Get git status if it's a git repo
		if _, err := os.Stat(filepath.Join(s.Path, ".git")); err == nil {
			status, _ := worktree.GetStatus(s.Path)
			item.GitStatus = &status
		}

		items = append(items, item)
	}

	return sessionsMsg{items}
}

type itemsMsg struct {
	items []Item
}

type sessionsMsg struct {
	items []SessionItem
}

type errMsg struct {
	err error
}

type tickMsg time.Time

type serverStateMsg struct {
	worktreePath string
	paneID       string
}

type sessionAddedMsg struct{}
type sessionRemovedMsg struct{}

// getCurrentPath returns the path of the currently selected item
func (m Model) getCurrentPath() string {
	if m.inSessionsSection {
		if m.cursor < len(m.sessionItems) {
			return m.sessionItems[m.cursor].Session.Path
		}
	} else {
		if m.cursor < len(m.items) {
			return m.items[m.cursor].Worktree.Path
		}
	}
	return ""
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle prompt input mode
	if m.prompts.hasActive() {
		return m.updatePrompt(msg)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case itemsMsg:
		m.items = msg.items

	case sessionsMsg:
		m.sessionItems = msg.items

	case sessionAddedMsg, sessionRemovedMsg:
		// Reload sessions after add/remove
		return m, m.loadSessions

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
	// Poll for waiting sessions
	paths, _ := waiting.ListWaiting(m.projectRoot)
	m.waitingSessions = make(map[string]bool)
	for _, p := range paths {
		m.waitingSessions[p] = true
	}

	// Check if server pane was killed externally
	if m.serverWorktreePath != "" && !tmux.IsServerRunning() {
		m.serverWorktreePath = ""
		waiting.ClearServerState(m.projectRoot)
	}

	// Update tmux window status for sessions
	for i := range m.sessionItems {
		_, m.sessionItems[i].TmuxOpen = tmux.FindWindowByName(m.sessionItems[i].Session.Name)
	}

	// Continue polling
	return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
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
		m.confirmMessage = ""

		if key.Matches(msg, keys.Yes) {
			handler, err := m.confirmHandler()
			m.confirmHandler = nil

			if err != nil {
				m.errorMessage = err.Error()
				return m, nil
			}

			return m, handler
		}

		return m, tea.ClearScreen
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

	case key.Matches(msg, keys.AddSession):
		m.prompts.activate("session")
		return m, textinput.Blink

	case key.Matches(msg, keys.Delete):
		// Only allow deletion of additional sessions
		if m.inSessionsSection && m.cursor < len(m.sessionItems) {
			session := m.sessionItems[m.cursor]
			// Kill tmux window if open
			if session.TmuxOpen {
				tmux.KillWindow(session.Session.Name)
			}
			// Remove from sessions file
			sessions.Remove(m.projectRoot, session.Session.Path)
			// Adjust cursor if needed
			if m.cursor >= len(m.sessionItems)-1 && m.cursor > 0 {
				m.cursor--
			}
			// If no more sessions, move back to worktrees section
			if len(m.sessionItems) <= 1 {
				m.inSessionsSection = false
				m.cursor = len(m.items) - 1
				if m.cursor < 0 {
					m.cursor = 0
				}
			}
			return m, m.loadSessions
		} else {
			wt := m.items[m.cursor]

			m.showConfirm = true
			m.confirmMessage = "Remove " + confirmTargetStyle.Render(wt.Worktree.Branch) + "?"
			m.confirmHandler = m.removeSelectedWorktree

			return m, nil
		}

	case key.Matches(msg, keys.AddWorktree):
		m.prompts.activate("worktreeDir")
		return m, textinput.Blink

	case key.Matches(msg, keys.Expand):
		if m.inSessionsSection {
			if m.cursor < len(m.sessionItems) {
				m.sessionItems[m.cursor].Expanded = true
			}
		} else if m.cursor < len(m.items) {
			m.items[m.cursor].Expanded = true
		}

	case key.Matches(msg, keys.Collapse):
		if m.inSessionsSection {
			if m.cursor < len(m.sessionItems) {
				m.sessionItems[m.cursor].Expanded = false
			}
		} else if m.cursor < len(m.items) {
			m.items[m.cursor].Expanded = false
		}

	case key.Matches(msg, keys.Up):
		if m.inSessionsSection {
			if m.cursor > 0 {
				m.cursor--
			} else {
				// Move to worktrees section
				m.inSessionsSection = false
				m.cursor = len(m.items) - 1
				if m.cursor < 0 {
					m.cursor = 0
				}
			}
		} else if m.cursor > 0 {
			m.cursor--
		}

	case key.Matches(msg, keys.Down):
		if m.inSessionsSection {
			if m.cursor < len(m.sessionItems)-1 {
				m.cursor++
			}
		} else {
			if m.cursor < len(m.items)-1 {
				m.cursor++
			} else if len(m.sessionItems) > 0 {
				// Move to sessions section
				m.inSessionsSection = true
				m.cursor = 0
			}
		}

	case key.Matches(msg, keys.Enter):
		if m.inSessionsSection && m.cursor < len(m.sessionItems) {
			session := m.sessionItems[m.cursor]
			// Open or switch to tmux window
			if session.TmuxOpen {
				tmux.SelectWindow(session.Session.Name)
			} else {
				tmux.NewWindow(session.Session.Name, session.Session.Path)
				m.config.SendSetupAndClaudeToTmux("claude")
			}
		} else if m.cursor < len(m.items) {
			item := m.items[m.cursor]
			isNew, _ := tmux.OpenWorktree(item.Worktree.Name, item.Worktree.Path)
			if isNew {
				// Build claude command with optional restore-context
				claudeCmd := "claude \"Hi\""
				if item.HasContext {
					claudeCmd = "claude \"/restore-context " + item.Worktree.Name + "\""
				}
				m.config.SendSetupAndClaudeToTmux(claudeCmd)
			}
		}

	case key.Matches(msg, keys.OpenCode):
		path := m.getCurrentPath()
		if path != "" {
			return m, openEditor(path, m.options.Editor)
		}

	case key.Matches(msg, keys.Server):
		if !m.inSessionsSection && m.cursor < len(m.items) {
			item := m.items[m.cursor]

			// Check if server is running in this worktree - toggle off
			if m.serverWorktreePath == item.Worktree.Path {
				tmux.StopServerSplit()
				m.serverWorktreePath = ""
				waiting.ClearServerState(m.projectRoot)
				return m, nil
			}

			// Check if server command is configured
			if m.options.ServerCommand == "" {
				m.errorMessage = "Error: No server command configured.\nUse -s or --server-command flag, or set WD_SERVER_COMMAND"
				return m, nil
			}

			// Start server (will auto-stop existing one)
			paneID, err := tmux.StartServerSplit(item.Worktree.Path, m.options.ServerCommand, m.options.ServerEnv)
			if err != nil {
				m.errorMessage = fmt.Sprintf("Error starting server: %v", err)
				return m, nil
			}
			m.serverWorktreePath = item.Worktree.Path
			waiting.SaveServerState(m.projectRoot, paneID, item.Worktree.Path)
		}

	case key.Matches(msg, keys.Shell):
		path := m.getCurrentPath()
		if path != "" {
			tmux.OpenShellSplit(path)
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

func (m Model) removeSelectedWorktree() (handler func() tea.Msg, err error) {
	wt := m.items[m.cursor]

	if err := worktree.Remove(wt.Worktree); err != nil {
		return nil, err
	}

	return m.loadWorktrees, nil
}
