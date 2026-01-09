package tui

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/foodstarter/worktree-dashboard/internal/context"
	"github.com/foodstarter/worktree-dashboard/internal/tmux"
	"github.com/foodstarter/worktree-dashboard/internal/waiting"
	"github.com/foodstarter/worktree-dashboard/internal/worktree"
)

// Config holds the TUI configuration
type Config struct {
	ServerCommand string
	Editor        string
}

// Item represents a worktree with its context info
type Item struct {
	Worktree   worktree.Worktree
	HasContext bool
	Context    *context.Context
	Expanded   bool
}

// Model is the bubbletea model
type Model struct {
	items              []Item
	cursor             int
	projectRoot        string
	width              int
	height             int
	err                error
	waitingSessions    map[string]bool // paths with sessions waiting for input
	config             Config
	serverWorktreePath string // path of worktree where server is running
	errorMessage       string // error message to show in popup
	showHelp           bool   // show help popup
}

// KeyMap defines keyboard shortcuts
type KeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Enter    key.Binding
	OpenCode key.Binding
	Server   key.Binding
	Shell    key.Binding
	Expand   key.Binding
	Collapse key.Binding
	Help     key.Binding
	Quit     key.Binding
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
	OpenCode: key.NewBinding(
		key.WithKeys("o"),
		key.WithHelp("o", "open in editor"),
	),
	Server: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "start/stop server"),
	),
	Shell: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "open shell"),
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
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("212")).
			MarginBottom(1)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("212")).
			Bold(true)

	issueStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39"))

	branchStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243"))

	cleanStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42"))

	contextStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			Italic(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginTop(1)

	treeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("237"))

	// Status colors
	stagedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")) // green

	modifiedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")) // orange

	untrackedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")) // gray

	deletedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("203")) // red

	waitingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("220")). // yellow/amber
			Bold(true)

	serverRunningStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("42")). // green
				Bold(true)

	stepStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			Italic(true)

	stepTimestampStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("237"))

	configKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243"))

	configValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("39"))

	configNotSetStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("241")).
				Italic(true)

	errorPopupStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("203")).
			Padding(1, 2).
			Foreground(lipgloss.Color("203"))

	helpPopupStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("212")).
			Padding(1, 2)

	helpKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("212")).
			Bold(true)

	helpDescStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))
)

// New creates a new TUI model
func New(projectRoot string, config Config) Model {
	return Model{
		projectRoot: projectRoot,
		config:      config,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.loadWorktrees,
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
		if wt.IssueID != "" {
			item.HasContext = context.Exists(m.projectRoot, wt.IssueID)
			if item.HasContext {
				ctx, _ := context.Load(m.projectRoot, wt.IssueID)
				item.Context = ctx
			}
		}

		items = append(items, item)
	}

	return itemsMsg{items}
}

type itemsMsg struct {
	items []Item
}

type errMsg struct {
	err error
}

type tickMsg time.Time

type serverStateMsg struct {
	worktreePath string
	paneID       string
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case itemsMsg:
		m.items = msg.items

	case errMsg:
		m.err = msg.err

	case serverStateMsg:
		m.serverWorktreePath = msg.worktreePath

	case tickMsg:
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

		// Continue polling
		return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })

	case tea.KeyMsg:
		// If error popup is showing, dismiss on any key
		if m.errorMessage != "" {
			m.errorMessage = ""
			return m, nil
		}

		// If help popup is showing, close it but continue processing the key
		if m.showHelp {
			m.showHelp = false
			// Don't return - let the key also perform its action
		}

		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, keys.Help):
			m.showHelp = !m.showHelp // Toggle
			return m, nil

		case key.Matches(msg, keys.Expand):
			if len(m.items) > 0 {
				m.items[m.cursor].Expanded = true
			}

		case key.Matches(msg, keys.Collapse):
			if len(m.items) > 0 {
				m.items[m.cursor].Expanded = false
			}

		case key.Matches(msg, keys.Up):
			if m.cursor > 0 {
				m.cursor--
			}

		case key.Matches(msg, keys.Down):
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}

		case key.Matches(msg, keys.Enter):
			if len(m.items) > 0 {
				item := m.items[m.cursor]
				if item.Worktree.IssueID != "" {
					_ = tmux.OpenWorktree(item.Worktree.IssueID, item.Worktree.Path)
				}
			}

		case key.Matches(msg, keys.OpenCode):
			if len(m.items) > 0 {
				item := m.items[m.cursor]
				return m, openEditor(item.Worktree.Path, m.config.Editor)
			}

		case key.Matches(msg, keys.Server):
			if len(m.items) > 0 {
				item := m.items[m.cursor]

				// Check if server is running in this worktree - toggle off
				if m.serverWorktreePath == item.Worktree.Path {
					tmux.StopServerSplit()
					m.serverWorktreePath = ""
					waiting.ClearServerState(m.projectRoot)
					return m, nil
				}

				// Check if server command is configured
				if m.config.ServerCommand == "" {
					m.errorMessage = "Error: No server command configured.\nUse -s or --server-command flag, or set WORKTREE_DASHBOARD_SERVER_COMMAND"
					return m, nil
				}

				// Start server (will auto-stop existing one)
				paneID, err := tmux.StartServerSplit(item.Worktree.Path, m.config.ServerCommand)
				if err != nil {
					m.errorMessage = fmt.Sprintf("Error starting server: %v", err)
					return m, nil
				}
				m.serverWorktreePath = item.Worktree.Path
				waiting.SaveServerState(m.projectRoot, paneID, item.Worktree.Path)
			}

		case key.Matches(msg, keys.Shell):
			if len(m.items) > 0 {
				item := m.items[m.cursor]
				tmux.OpenShellSplit(item.Worktree.Path)
			}
		}
	}

	return m, nil
}

// openEditor opens the configured editor in the given path
func openEditor(path, editor string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command(editor, path)
		_ = cmd.Start()
		return nil
	}
}

// View renders the UI
func (m Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress q to quit.", m.err)
	}

	if len(m.items) == 0 {
		return "Loading worktrees..."
	}

	var b strings.Builder

	b.WriteString(titleStyle.Render("Claude Worktrees"))
	b.WriteString("\n")

	// Options tree view
	b.WriteString(m.renderOptionsTree())
	b.WriteString("\n")

	for i, item := range m.items {
		isSelected := i == m.cursor
		b.WriteString(m.renderItem(item, isSelected, m.width))
	}

	// Minimal help hint
	b.WriteString(helpStyle.Render("[?] help  [q] quit"))

	// Help popup
	if m.showHelp {
		b.WriteString("\n\n")
		b.WriteString(m.renderHelpPopup())
	}

	// Error popup overlay
	if m.errorMessage != "" {
		b.WriteString("\n\n")
		b.WriteString(errorPopupStyle.Render(m.errorMessage + "\n\nPress any key to dismiss"))
	}

	return b.String()
}

// renderHelpPopup renders the help popup content
func (m Model) renderHelpPopup() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Keyboard Shortcuts"))
	b.WriteString("\n\n")

	helpItems := []struct {
		key  string
		desc string
	}{
		{"enter", "Open worktree in tmux with Claude"},
		{"o", "Open worktree in editor"},
		{"s", "Start/stop server"},
		{"c", "Open shell in vertical split"},
		{"l / →", "Expand file tree"},
		{"h / ←", "Collapse file tree"},
		{"j / k", "Navigate down/up"},
		{"?", "Show this help"},
		{"q", "Quit"},
	}

	for _, item := range helpItems {
		b.WriteString(fmt.Sprintf("  %s  %s\n",
			helpKeyStyle.Render(fmt.Sprintf("%-7s", item.key)),
			helpDescStyle.Render(item.desc),
		))
	}

	b.WriteString(helpStyle.Render("\nPress any key to close"))

	return helpPopupStyle.Render(b.String())
}

// renderOptionsTree renders the config options as a tree view
func (m Model) renderOptionsTree() string {
	var b strings.Builder

	// Server command
	b.WriteString(treeStyle.Render("├── "))
	b.WriteString(configKeyStyle.Render("server: "))
	if m.config.ServerCommand != "" {
		b.WriteString(configValueStyle.Render(m.config.ServerCommand))
	} else {
		b.WriteString(configNotSetStyle.Render("not set"))
	}
	b.WriteString("\n")

	// Editor
	b.WriteString(treeStyle.Render("└── "))
	b.WriteString(configKeyStyle.Render("editor: "))
	b.WriteString(configValueStyle.Render(m.config.Editor))
	b.WriteString("\n")

	return b.String()
}

func (m Model) renderItem(item Item, isSelected bool, width int) string {
	var b strings.Builder
	status := item.Worktree.GitStatus

	// Expand indicator (selection shown by highlight)
	indicator := "▸"
	if item.Expanded {
		indicator = "▾"
	}

	// Issue ID
	issueID := item.Worktree.IssueID
	if issueID == "" {
		issueID = "main"
	}

	// Check if session is waiting for input
	waitingBadge := ""
	if m.waitingSessions[item.Worktree.Path] {
		waitingBadge = " " + waitingStyle.Render("⏳")
	}

	// Check if server is running in this worktree
	serverBadge := ""
	if m.serverWorktreePath == item.Worktree.Path {
		serverBadge = " " + serverRunningStyle.Render("▶")
	}

	// Color the indicator when selected
	styledIndicator := indicator
	if isSelected {
		styledIndicator = selectedStyle.Render(indicator)
	}

	// First line: indicator, issue ID, badges, branch
	line1 := fmt.Sprintf("%s %s%s%s  %s",
		styledIndicator,
		issueStyle.Render(issueID),
		serverBadge,
		waitingBadge,
		branchStyle.Render(item.Worktree.Branch),
	)

	b.WriteString(line1)
	b.WriteString("\n")

	// Second line: status summary
	var statusParts []string

	if status.Clean {
		statusParts = append(statusParts, cleanStyle.Render("✓ clean"))
	} else {
		totalFiles := len(status.Files)
		statusParts = append(statusParts, fmt.Sprintf("● %d files", totalFiles))

		if status.StagedCount > 0 {
			statusParts = append(statusParts, stagedStyle.Render(fmt.Sprintf("+%d staged", status.StagedCount)))
		}
		if status.ModifiedCount > 0 {
			statusParts = append(statusParts, modifiedStyle.Render(fmt.Sprintf("~%d modified", status.ModifiedCount)))
		}
		if status.UntrackedCount > 0 {
			statusParts = append(statusParts, untrackedStyle.Render(fmt.Sprintf("?%d untracked", status.UntrackedCount)))
		}
	}

	if status.Ahead > 0 {
		statusParts = append(statusParts, fmt.Sprintf("↑%d", status.Ahead))
	}
	if status.Behind > 0 {
		statusParts = append(statusParts, fmt.Sprintf("↓%d", status.Behind))
	}

	// Only show [context] if there are no steps (steps imply context exists)
	if item.HasContext && (item.Context == nil || len(item.Context.Steps) == 0) {
		statusParts = append(statusParts, contextStyle.Render("[context]"))
	}

	line2 := "    " + strings.Join(statusParts, "  ")
	b.WriteString(line2)
	b.WriteString("\n")

	// Show steps from context (if any)
	if item.Context != nil && len(item.Context.Steps) > 0 {
		steps := item.Context.Steps
		if item.Expanded {
			// Show last 5 steps in chronological order with timestamps
			start := 0
			if len(steps) > 5 {
				start = len(steps) - 5
			}
			for i := start; i < len(steps); i++ {
				step := steps[i]
				// Truncate message if needed (indent=4, timestamp~20, buffer=3)
				msg := truncateString(step.Message, width-27)
				stepLine := fmt.Sprintf("    %s %s",
					stepTimestampStyle.Render("["+step.Timestamp+"]"),
					stepStyle.Render(msg),
				)
				b.WriteString(stepLine)
				b.WriteString("\n")
			}
			// Add blank line before file tree if there are files
			if !status.Clean {
				b.WriteString("\n")
			}
		} else {
			// Collapsed: show only the last step message (no timestamp)
			lastStep := steps[len(steps)-1]
			// Truncate message if needed (indent=4, buffer=3)
			msg := truncateString(lastStep.Message, width-7)
			stepLine := fmt.Sprintf("    %s", stepStyle.Render(msg))
			b.WriteString(stepLine)
			b.WriteString("\n")
		}
	}

	// Expanded file tree
	if item.Expanded && !status.Clean {
		treeLines := renderFileTree(status.Files)
		for _, line := range treeLines {
			b.WriteString("    " + line + "\n")
		}
	}

	b.WriteString("\n")
	return b.String()
}

// truncateString truncates a string to maxLen, adding "..." if truncated
func truncateString(s string, maxLen int) string {
	if maxLen <= 0 || len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return "..."
	}
	return s[:maxLen-3] + "..."
}

// renderFileTree renders all files as a unified tree with status indicators
func renderFileTree(files []worktree.FileStatus) []string {
	if len(files) == 0 {
		return nil
	}

	// Sort files by path
	sorted := make([]worktree.FileStatus, len(files))
	copy(sorted, files)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Path < sorted[j].Path
	})

	// Group by top-level directory
	type dirGroup struct {
		files []worktree.FileStatus
	}
	dirs := make(map[string]*dirGroup)
	var rootFiles []worktree.FileStatus
	var dirOrder []string

	for _, f := range sorted {
		parts := strings.SplitN(f.Path, string(filepath.Separator), 2)
		if len(parts) == 1 {
			// Root-level file
			rootFiles = append(rootFiles, f)
		} else {
			topDir := parts[0]
			if dirs[topDir] == nil {
				dirs[topDir] = &dirGroup{}
				dirOrder = append(dirOrder, topDir)
			}
			dirs[topDir].files = append(dirs[topDir].files, f)
		}
	}

	var lines []string
	totalItems := len(dirOrder) + len(rootFiles)
	itemIdx := 0

	// Render directories
	for _, dirName := range dirOrder {
		itemIdx++
		isLast := itemIdx == totalItems

		prefix := "├── "
		childPrefix := "│   "
		if isLast {
			prefix = "└── "
			childPrefix = "    "
		}

		lines = append(lines, treeStyle.Render(prefix)+dirName+"/")

		// Render files in this directory
		dirFiles := dirs[dirName].files
		for j, f := range dirFiles {
			fileIsLast := j == len(dirFiles)-1
			filePrefix := childPrefix + "├── "
			if fileIsLast {
				filePrefix = childPrefix + "└── "
			}

			// Get just the filename part after the directory
			name := strings.TrimPrefix(f.Path, dirName+string(filepath.Separator))
			styledFile := renderFileWithStatus(name, f)
			lines = append(lines, treeStyle.Render(filePrefix)+styledFile)
		}
	}

	// Render root-level files
	for _, f := range rootFiles {
		itemIdx++
		isLast := itemIdx == totalItems

		prefix := "├── "
		if isLast {
			prefix = "└── "
		}

		styledFile := renderFileWithStatus(f.Path, f)
		lines = append(lines, treeStyle.Render(prefix)+styledFile)
	}

	return lines
}

// renderFileWithStatus renders a filename with its status indicator
func renderFileWithStatus(name string, f worktree.FileStatus) string {
	// Determine the status indicator and style
	var indicator string
	var style lipgloss.Style

	if f.Staged == '?' && f.Unstaged == '?' {
		// Untracked
		indicator = " ?"
		style = untrackedStyle
	} else if f.Staged != ' ' && f.Unstaged != ' ' && f.Unstaged != '?' {
		// Both staged and unstaged changes
		indicator = fmt.Sprintf(" %c%c", f.Staged, f.Unstaged)
		style = modifiedStyle
	} else if f.Staged != ' ' && f.Staged != '?' {
		// Staged only
		indicator = fmt.Sprintf(" %c", f.Staged)
		style = stagedStyle
	} else if f.Unstaged != ' ' && f.Unstaged != '?' {
		// Unstaged only
		indicator = fmt.Sprintf(" %c", f.Unstaged)
		style = modifiedStyle
	}

	// Use deleted style for deletions
	if f.Staged == 'D' || f.Unstaged == 'D' {
		style = deletedStyle
	}

	return style.Render(name + indicator)
}
