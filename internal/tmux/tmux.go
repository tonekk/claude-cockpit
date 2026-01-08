package tmux

import (
	"os"
	"os/exec"
	"strings"
)

// IsInsideTmux returns true if we're running inside a tmux session
func IsInsideTmux() bool {
	return os.Getenv("TMUX") != ""
}

// WindowExists checks if a tmux window with the given name exists
func WindowExists(windowName string) bool {
	cmd := exec.Command("tmux", "list-windows", "-F", "#{window_name}")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	windows := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, w := range windows {
		if w == windowName {
			return true
		}
	}
	return false
}

// SelectWindow switches to an existing tmux window
func SelectWindow(windowName string) error {
	cmd := exec.Command("tmux", "select-window", "-t", windowName)
	return cmd.Run()
}

// NewWindow creates a new tmux window with the given name and working directory
func NewWindow(windowName, workDir string) error {
	cmd := exec.Command("tmux", "new-window", "-n", windowName, "-c", workDir)
	return cmd.Run()
}

// SendKeys sends keystrokes to the current tmux pane
func SendKeys(keys string) error {
	cmd := exec.Command("tmux", "send-keys", keys, "Enter")
	return cmd.Run()
}

// OpenWorktree opens a worktree in a tmux window, running claude with restore-context
func OpenWorktree(issueID, worktreePath string) error {
	windowName := issueID

	// Check if window already exists
	if WindowExists(windowName) {
		return SelectWindow(windowName)
	}

	// Create new window
	if err := NewWindow(windowName, worktreePath); err != nil {
		return err
	}

	// Start claude with restore-context as the initial prompt
	return SendKeys("claude \"/restore-context " + issueID + "\"")
}

// StartSession starts a new tmux session if not already in one
func StartSession(sessionName string) error {
	if IsInsideTmux() {
		return nil
	}

	cmd := exec.Command("tmux", "new-session", "-d", "-s", sessionName)
	if err := cmd.Run(); err != nil {
		// Session might already exist, try to attach
		cmd = exec.Command("tmux", "attach-session", "-t", sessionName)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	// Attach to the new session
	cmd = exec.Command("tmux", "attach-session", "-t", sessionName)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
