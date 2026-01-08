package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/foodstarter/worktree-dashboard/internal/context"
	"github.com/foodstarter/worktree-dashboard/internal/tmux"
	"github.com/foodstarter/worktree-dashboard/internal/worktree"
)

// Item represents a worktree with its context info
type Item struct {
	Worktree   worktree.Worktree
	HasContext bool
	Context    *context.Context
}

// Model is the bubbletea model
type Model struct {
	items       []Item
	cursor      int
	projectRoot string
	width       int
	height      int
	err         error
}

// KeyMap defines keyboard shortcuts
type KeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Enter  key.Binding
	Delete key.Binding
	Quit   key.Binding
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
	Delete: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "delete"),
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
			Bold(true).
			Foreground(lipgloss.Color("212")).
			Background(lipgloss.Color("236")).
			Padding(0, 1)

	normalStyle = lipgloss.NewStyle().
			Padding(0, 1)

	issueStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39"))

	branchStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243"))

	cleanStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42"))

	dirtyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214"))

	contextStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			Italic(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginTop(1)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2)
)

// New creates a new TUI model
func New(projectRoot string) Model {
	return Model{
		projectRoot: projectRoot,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return m.loadWorktrees
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

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit

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
		}
	}

	return m, nil
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
	b.WriteString("\n\n")

	for i, item := range m.items {
		cursor := "  "
		style := normalStyle
		if i == m.cursor {
			cursor = "▸ "
			style = selectedStyle
		}

		// First line: issue ID and branch
		issueID := item.Worktree.IssueID
		if issueID == "" {
			issueID = "main"
		}

		line1 := fmt.Sprintf("%s%s  %s",
			cursor,
			issueStyle.Render(issueID),
			branchStyle.Render(item.Worktree.Branch),
		)

		// Second line: status info
		var statusParts []string

		if item.Worktree.GitStatus.Clean {
			statusParts = append(statusParts, cleanStyle.Render("✓ clean"))
		} else {
			statusParts = append(statusParts, dirtyStyle.Render(fmt.Sprintf("● %d modified", item.Worktree.GitStatus.ModifiedFiles)))
		}

		if item.Worktree.GitStatus.Ahead > 0 {
			statusParts = append(statusParts, fmt.Sprintf("↑%d", item.Worktree.GitStatus.Ahead))
		}
		if item.Worktree.GitStatus.Behind > 0 {
			statusParts = append(statusParts, fmt.Sprintf("↓%d", item.Worktree.GitStatus.Behind))
		}

		if item.HasContext {
			statusParts = append(statusParts, contextStyle.Render("[context saved]"))
		} else {
			statusParts = append(statusParts, contextStyle.Render("[no context]"))
		}

		line2 := "     " + strings.Join(statusParts, "   ")

		if i == m.cursor {
			b.WriteString(style.Render(line1))
		} else {
			b.WriteString(line1)
		}
		b.WriteString("\n")
		b.WriteString(line2)
		b.WriteString("\n\n")
	}

	// Help
	help := "[enter] open in tmux   [j/k] navigate   [q] quit"
	b.WriteString(helpStyle.Render(help))

	return b.String()
}
