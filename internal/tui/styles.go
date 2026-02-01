package tui

import "github.com/charmbracelet/lipgloss"

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

	confirmPopupStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("203")).
				Padding(1, 2)

	confirmTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("203"))

	confirmTargetStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("212"))

	confirmHintStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("241"))

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

	sessionNameStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("141")). // purple-ish
				Bold(true)

	sessionPathStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("243"))

	inputPopupStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("212")).
			Padding(1, 2)

	// Section box styles
	worktreeBoxStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("39")). // cyan like issue IDs
				Padding(0, 1).
				MarginBottom(1)

	sessionBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("141")). // purple like session names
			Padding(0, 1).
			MarginBottom(1)

	sectionTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("252"))
)
