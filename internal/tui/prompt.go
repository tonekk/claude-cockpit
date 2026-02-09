package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/tonekk/claude-cockpit/internal/sessions"
	"github.com/tonekk/claude-cockpit/internal/tmux"
	"github.com/tonekk/claude-cockpit/internal/worktree"
)

type Prompt struct {
	id       string
	title    string
	input    textinput.Model
	onSubmit func(m Model, value string) (tea.Model, tea.Cmd)
}

type PromptManager struct {
	prompts  []Prompt
	activeID string
}

func NewPromptManager() PromptManager {
	prompts := []Prompt{
		NewPrompt("session", "Add Additional Session", "~/path/to/directory", acceptSession),
		NewPrompt("worktreeDir", "Add Worktree - Choose directory name", "directory_name", acceptWorktreeDir),
		NewPrompt("worktreeBranch", "Add Worktree - Choose branch name", "feature/whatever-comes-to-your-mind", acceptWorktreeBranch),
	}

	return PromptManager{
		prompts: prompts,
	}
}

func NewPrompt(id, title, placeholder string, onSubmit func(m Model, value string) (tea.Model, tea.Cmd)) Prompt {
	input := textinput.New()
	input.Placeholder = placeholder
	input.CharLimit = 256
	input.Width = 50

	return Prompt{
		id:       id,
		title:    title,
		input:    input,
		onSubmit: onSubmit,
	}
}

func (pm *PromptManager) deactivate() {
	pm.activeID = ""
}

func (pm *PromptManager) byID(id string) (*Prompt, error) {
	for i := range pm.prompts {
		if pm.prompts[i].id == id {
			return &pm.prompts[i], nil
		}
	}
	return nil, fmt.Errorf("prompt with id %s not found", id)
}

func (pm *PromptManager) active() (*Prompt, bool) {
	prompt, err := pm.byID(pm.activeID)
	if err != nil {
		return nil, false
	}
	return prompt, true
}

func (pm *PromptManager) hasActive() bool {
	return pm.activeID != ""
}

func (pm *PromptManager) activate(id string) error {
	if pm.activeID != "" {
		return fmt.Errorf("%s is already activated", pm.activeID)
	}

	prompt, err := pm.byID(id)
	if err != nil {
		return err
	}

	pm.activeID = id
	prompt.input.SetValue("")
	prompt.input.Focus()

	return nil
}

func acceptSession(m Model, path string) (tea.Model, tea.Cmd) {
	// Expand ~ to home directory
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, path[2:])
	}
	// Validate path exists
	if _, err := os.Stat(path); err != nil {
		m.errorMessage = fmt.Sprintf("Path does not exist: %s", path)
		m.prompts.deactivate()
		return m, nil
	}
	// Add to sessions
	if err := sessions.Add(m.projectRoot, path); err != nil {
		m.errorMessage = fmt.Sprintf("Error adding additional session: %v", err)
		m.prompts.deactivate()
		return m, nil
	}
	// Open tmux window with claude immediately
	sessionName := filepath.Base(path)
	tmux.NewWindow(sessionName, path)
	tmux.SendKeys("claude")
	m.prompts.deactivate()
	return m, m.loadSessions
}

func acceptWorktreeDir(m Model, dirName string) (tea.Model, tea.Cmd) {
	m.newWorktreeDir = dirName
	m.prompts.deactivate()
	m.prompts.activate("worktreeBranch")
	return m, textinput.Blink
}

func acceptWorktreeBranch(m Model, branchName string) (tea.Model, tea.Cmd) {
	dirName := m.newWorktreeDir
	m.newWorktreeDir = ""
	m.prompts.deactivate()

	if err := worktree.Create(m.projectRoot, dirName, branchName); err != nil {
		m.errorMessage = fmt.Sprintf("Error creating worktree: %v", err)
		return m, nil
	}

	// Open the new worktree in tmux with setup + claude
	worktreePath := filepath.Join(m.projectRoot, ".worktrees", dirName)
	tmux.NewWindow(dirName, worktreePath)
	m.config.SendSetupAndClaudeToTmux("claude")

	return m, m.loadWorktrees
}
