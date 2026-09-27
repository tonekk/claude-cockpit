package tui

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/tonekk/claude-cockpit/internal/worktree"
)

// View renders the UI
func (m Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress q to quit.", m.err)
	}

	if len(m.worktreeItems) == 0 {
		return "Loading worktrees..."
	}

	var b strings.Builder

	b.WriteString("\n\n")
	b.WriteString(titleStyle.Render("Claude Cockpit"))
	b.WriteString("\n\n")

	// Worktrees section with border
	var worktreeContent strings.Builder
	worktreeContent.WriteString(sectionTitleStyle.Render("Worktrees"))
	worktreeContent.WriteString("\n\n")
	for i, item := range m.worktreeItems {
		isSelected := i == m.cursor
		worktreeContent.WriteString(m.renderItem(item, isSelected, m.width))
	}
	b.WriteString(worktreeBoxStyle.Render(worktreeContent.String()))
	b.WriteString("\n")

	// Minimal help hint
	b.WriteString(helpStyle.Render("[?] help  [q] quit"))

	// Prompt popup
	if promptPopup := m.renderPromptPopup(); promptPopup != "" {
		b.WriteString("\n\n")
		b.WriteString(promptPopup)
	}

	// Confirm popup
	if m.showConfirm {
		var cb strings.Builder
		cb.WriteString(confirmTitleStyle.Render(m.confirmHeader))
		cb.WriteString("\n\n")
		cb.WriteString(m.confirmMessage)
		cb.WriteString("\n\n")
		cb.WriteString(confirmHintStyle.Render("Press "))
		cb.WriteString(helpKeyStyle.Render("y"))
		cb.WriteString(confirmHintStyle.Render(" to confirm, any other key to cancel"))
		b.WriteString("\n\n")
		b.WriteString(confirmPopupStyle.Render(cb.String()))
	}

	// Help popup
	if m.showHelp {
		b.WriteString("\n\n")
		b.WriteString(m.renderHelpPopup())
	}

	// Config popup
	if m.showConfig {
		b.WriteString("\n\n")
		b.WriteString(m.renderConfigPopup())
	}

	// Error popup overlay
	if m.errorMessage != "" {
		b.WriteString("\n\n")
		b.WriteString(errorPopupStyle.Render(m.errorMessage + "\n\nPress any key to dismiss"))
	}

	return b.String()
}

func (m Model) renderPromptPopup() string {
	prompt, hasActive := m.prompts.active()
	if !hasActive {
		return ""
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render(prompt.title))
	b.WriteString("\n\n")
	b.WriteString("  Path: ")
	b.WriteString(prompt.input.View())
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("  Enter to confirm, Esc to cancel"))

	return inputPopupStyle.Render(b.String())
}

// renderHelpPopup renders the help popup content
func (m Model) renderHelpPopup() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Keyboard Shortcuts"))
	b.WriteString("\n\n")

	for _, item := range keys.AllBindings() {
		b.WriteString(fmt.Sprintf("  %s  %s\n",
			helpKeyStyle.Render(fmt.Sprintf("%-7s", item.Help().Key)),
			helpDescStyle.Render(item.Help().Desc),
		))
	}

	b.WriteString(helpStyle.Render("\nPress any key to close"))

	return helpPopupStyle.Render(b.String())
}

// renderConfigPopup renders the config popup content
func (m Model) renderConfigPopup() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Config"))
	b.WriteString("\n\n")

	// Server command
	b.WriteString(fmt.Sprintf("  %s  ", helpKeyStyle.Render("server:")))
	if m.options.ServerCommand != "" {
		b.WriteString(configValueStyle.Render(m.options.ServerCommand))
	} else {
		b.WriteString(configNotSetStyle.Render("not set"))
	}
	b.WriteString("\n")

	// Server env vars
	if len(m.options.ServerEnv) > 0 {
		b.WriteString(fmt.Sprintf("  %s  ", helpKeyStyle.Render("env:")))
		var envPairs []string
		for k, v := range m.options.ServerEnv {
			envPairs = append(envPairs, k+"="+v)
		}
		b.WriteString(configValueStyle.Render(strings.Join(envPairs, ", ")))
		b.WriteString("\n")
	}

	// Editor
	b.WriteString(fmt.Sprintf("  %s  ", helpKeyStyle.Render("editor:")))
	b.WriteString(configValueStyle.Render(m.options.Editor))
	b.WriteString("\n")

	b.WriteString(helpStyle.Render("\nPress any key to close"))

	return helpPopupStyle.Render(b.String())
}

func (m Model) renderItem(item WorktreeItem, isSelected bool, width int) string {
	var b strings.Builder
	status := item.Worktree.GitStatus

	// Expand indicator (selection shown by highlight)
	indicator := "▶"
	if item.Expanded {
		indicator = "▼"
	}

	// Name (directory basename)
	name := item.Worktree.Name
	hasWindow := m.tmuxWindows[item.Worktree.Path]

	// Color the indicator when selected
	styledIndicator := indicator
	if isSelected {
		styledIndicator = selectedStyle.Render(indicator)
	}

	// First line: indicator, issue ID, badges, branch
	nameStyle := issueStyle
	bStyle := branchStyle
	if !hasWindow {
		nameStyle = dimStyle
		bStyle = dimStyle
	}

	line1 := fmt.Sprintf("%s %s  %s",
		styledIndicator,
		nameStyle.Render(name),
		bStyle.Render(item.Worktree.Branch),
	)

	b.WriteString(line1)
	b.WriteString("\n")

	// Second line: status summary
	var statusParts []string

	if status.Clean {
		s := cleanStyle
		if !hasWindow {
			s = dimStyle
		}
		statusParts = append(statusParts, s.Render("✓ clean"))
	} else {
		totalFiles := len(status.Files)
		if hasWindow {
			statusParts = append(statusParts, fmt.Sprintf("● %d files", totalFiles))
		} else {
			statusParts = append(statusParts, dimStyle.Render(fmt.Sprintf("● %d files", totalFiles)))
		}

		if status.StagedCount > 0 {
			s := stagedStyle
			if !hasWindow {
				s = dimStyle
			}
			statusParts = append(statusParts, s.Render(fmt.Sprintf("+%d staged", status.StagedCount)))
		}
		if status.ModifiedCount > 0 {
			s := modifiedStyle
			if !hasWindow {
				s = dimStyle
			}
			statusParts = append(statusParts, s.Render(fmt.Sprintf("~%d modified", status.ModifiedCount)))
		}
		if status.UntrackedCount > 0 {
			s := untrackedStyle
			if !hasWindow {
				s = dimStyle
			}
			statusParts = append(statusParts, s.Render(fmt.Sprintf("?%d untracked", status.UntrackedCount)))
		}
	}

	if status.Ahead > 0 {
		text := fmt.Sprintf("↑%d", status.Ahead)
		if !hasWindow {
			text = dimStyle.Render(text)
		}
		statusParts = append(statusParts, text)
	}
	if status.Behind > 0 {
		text := fmt.Sprintf("↓%d", status.Behind)
		if !hasWindow {
			text = dimStyle.Render(text)
		}
		statusParts = append(statusParts, text)
	}

	line2 := "    " + strings.Join(statusParts, "  ")
	b.WriteString(line2)
	b.WriteString("\n")

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
