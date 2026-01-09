package tmux

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const WaitingSuffix = " 🔴"

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
	// Check if window already exists (with or without waiting prefix)
	if actualName, found := FindWindowByIssueID(issueID); found {
		return SelectWindow(actualName)
	}

	// Create new window
	if err := NewWindow(issueID, worktreePath); err != nil {
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

// FindWindowByIssueID finds a tmux window by issue ID, checking both with and without waiting suffix
// Returns the actual window name and whether it was found
func FindWindowByIssueID(issueID string) (string, bool) {
	cmd := exec.Command("tmux", "list-windows", "-F", "#{window_name}")
	output, err := cmd.Output()
	if err != nil {
		return "", false
	}

	windows := strings.Split(strings.TrimSpace(string(output)), "\n")
	waitingName := issueID + WaitingSuffix

	for _, w := range windows {
		if w == issueID || w == waitingName {
			return w, true
		}
	}
	return "", false
}

// RenameWindow renames a tmux window
func RenameWindow(oldName, newName string) error {
	cmd := exec.Command("tmux", "rename-window", "-t", oldName, newName)
	return cmd.Run()
}

// MarkWindowWaiting adds the waiting suffix to a window's name
func MarkWindowWaiting(issueID string) error {
	currentName, found := FindWindowByIssueID(issueID)
	if !found {
		return nil // Window doesn't exist, nothing to do
	}

	// Already has waiting suffix
	if strings.HasSuffix(currentName, WaitingSuffix) {
		return nil
	}

	return RenameWindow(currentName, issueID+WaitingSuffix)
}

// ClearWindowWaiting removes the waiting suffix from a window's name
func ClearWindowWaiting(issueID string) error {
	currentName, found := FindWindowByIssueID(issueID)
	if !found {
		return nil // Window doesn't exist, nothing to do
	}

	// Doesn't have waiting suffix
	if !strings.HasSuffix(currentName, WaitingSuffix) {
		return nil
	}

	return RenameWindow(currentName, issueID)
}

// ServerPaneID stores the pane ID of the running server split
var ServerPaneID string

// StartServerSplit creates a horizontal split and runs the server command
// Returns the pane ID of the new split
func StartServerSplit(workDir, serverCommand string) (string, error) {
	// First stop any existing server
	if ServerPaneID != "" {
		StopServerSplit()
	}

	// Check if .mise.toml exists in workDir and mise is available - if so, wrap with mise exec
	finalCommand := serverCommand
	if _, err := os.Stat(filepath.Join(workDir, ".mise.toml")); err == nil {
		if _, err := exec.LookPath("mise"); err == nil {
			finalCommand = "mise exec -- " + serverCommand
		}
	}

	// Create horizontal split (left/right) with the server command
	cmd := exec.Command("tmux", "split-window", "-h", "-c", workDir, "-P", "-F", "#{pane_id}", finalCommand)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	paneID := strings.TrimSpace(string(output))
	ServerPaneID = paneID

	// Focus back on the original pane (the dashboard)
	exec.Command("tmux", "select-pane", "-L").Run()

	return paneID, nil
}

// StopServerSplit kills the server split pane
func StopServerSplit() error {
	if ServerPaneID == "" {
		return nil
	}

	cmd := exec.Command("tmux", "kill-pane", "-t", ServerPaneID)
	err := cmd.Run()
	ServerPaneID = ""
	return err
}

// PaneExists checks if a tmux pane with the given ID exists
func PaneExists(paneID string) bool {
	if paneID == "" {
		return false
	}

	cmd := exec.Command("tmux", "list-panes", "-a", "-F", "#{pane_id}")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	panes := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, p := range panes {
		if p == paneID {
			return true
		}
	}

	return false
}

// IsServerRunning returns true if a server split is currently running
func IsServerRunning() bool {
	if ServerPaneID == "" {
		return false
	}

	if !PaneExists(ServerPaneID) {
		ServerPaneID = ""
		return false
	}

	return true
}

// OpenShellSplit opens a shell in a vertical split at the given directory
func OpenShellSplit(workDir string) error {
	cmd := exec.Command("tmux", "split-window", "-v", "-c", workDir)
	return cmd.Run()
}
