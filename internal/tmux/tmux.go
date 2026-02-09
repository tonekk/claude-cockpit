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

// ProjectRoot is set by the TUI to track which project owns all sessions
var ProjectRoot string

// NewWindow creates a new tmux window with the given name and working directory
func NewWindow(windowName, workDir string) error {
	args := []string{"new-window", "-n", windowName, "-c", workDir}
	// Set environment variable so hooks know which project to notify
	if ProjectRoot != "" {
		args = append(args, "-e", "WORKTREE_DASHBOARD_PROJECT="+ProjectRoot)
	}
	cmd := exec.Command("tmux", args...)
	return cmd.Run()
}

// SendKeys sends keystrokes to the current tmux pane
func SendKeys(keys string) error {
	cmd := exec.Command("tmux", "send-keys", keys, "Enter")
	return cmd.Run()
}

// OpenWorktree opens a worktree in a tmux window
// Returns true if a new window was created (caller should send commands)
func OpenWorktree(name, worktreePath string) (isNew bool, err error) {
	// Check if window already exists (with or without waiting prefix)
	if actualName, found := FindWindowByName(name); found {
		return false, SelectWindow(actualName)
	}

	// Create new window
	if err := NewWindow(name, worktreePath); err != nil {
		return false, err
	}

	return true, nil
}

// FindWindowByName finds a tmux window by name, checking both with and without waiting suffix
// Returns the actual window name and whether it was found
func FindWindowByName(name string) (string, bool) {
	cmd := exec.Command("tmux", "list-windows", "-F", "#{window_name}")
	output, err := cmd.Output()
	if err != nil {
		return "", false
	}

	windows := strings.Split(strings.TrimSpace(string(output)), "\n")
	waitingName := name + WaitingSuffix

	for _, w := range windows {
		if w == name || w == waitingName {
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

// KillWindow kills a tmux window by name
func KillWindow(windowName string) error {
	cmd := exec.Command("tmux", "kill-window", "-t", windowName)
	return cmd.Run()
}

// MarkWindowWaiting adds the waiting suffix to a window's name
func MarkWindowWaiting(name string) error {
	currentName, found := FindWindowByName(name)
	if !found {
		return nil // Window doesn't exist, nothing to do
	}

	// Already has waiting suffix
	if strings.HasSuffix(currentName, WaitingSuffix) {
		return nil
	}

	return RenameWindow(currentName, name+WaitingSuffix)
}

// ClearWindowWaiting removes the waiting suffix from a window's name
func ClearWindowWaiting(name string) error {
	currentName, found := FindWindowByName(name)
	if !found {
		return nil // Window doesn't exist, nothing to do
	}

	// Doesn't have waiting suffix
	if !strings.HasSuffix(currentName, WaitingSuffix) {
		return nil
	}

	return RenameWindow(currentName, name)
}

// ServerPaneID stores the pane ID of the running server split
var ServerPaneID string

// StartServerSplit creates a horizontal split and runs the server command
// Returns the pane ID of the new split
func StartServerSplit(workDir, serverCommand string, envVars map[string]string) (string, error) {
	// First stop any existing server
	if ServerPaneID != "" {
		StopServerSplit()
	}

	// Build the final command with env vars prefix
	finalCommand := serverCommand

	// Prepend env vars if any
	if len(envVars) > 0 {
		var envParts []string
		for k, v := range envVars {
			envParts = append(envParts, k+"="+v)
		}
		finalCommand = "env " + strings.Join(envParts, " ") + " " + serverCommand
	}

	// Check if .mise.toml exists in workDir and mise is available - if so, wrap with mise exec
	if _, err := os.Stat(filepath.Join(workDir, ".mise.toml")); err == nil {
		if _, err := exec.LookPath("mise"); err == nil {
			finalCommand = "mise exec -- " + finalCommand
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

// RenameCurrentWindow renames the current tmux window
func RenameCurrentWindow(newName string) error {
	cmd := exec.Command("tmux", "rename-window", newName)
	return cmd.Run()
}

// EnsureCockpitSession checks tmux state and sets up the cockpit window/session
// Returns true if we need to exec into a new session (caller should exec)
func EnsureCockpitSession(projectDir string) (needsExec bool, sessionName string) {
	basename := projectDir
	if idx := strings.LastIndex(projectDir, string(os.PathSeparator)); idx >= 0 {
		basename = projectDir[idx+1:]
	}
	sessionName = "claude-" + basename

	if IsInsideTmux() {
		// Already in tmux, just rename the current window
		RenameCurrentWindow("cockpit 🎛️")
		return false, ""
	}

	// Not in tmux - need to create/attach to session
	return true, sessionName
}

// ExecIntoSession creates a new tmux session (or attaches if exists) and execs into it
// This replaces the current process
func ExecIntoSession(sessionName, workDir string) error {
	tmuxPath, err := exec.LookPath("tmux")
	if err != nil {
		return err
	}

	// Get our own executable path to re-run in the new session
	selfPath, err := os.Executable()
	if err != nil {
		selfPath = "claude-cockpit" // fallback to hoping it's in PATH
	}

	// Try to create a new session with the cockpit window, running ourselves
	cmd := exec.Command("tmux", "new-session", "-d", "-s", sessionName, "-c", workDir, "-n", "cockpit 🎛️", selfPath)
	if err := cmd.Run(); err != nil {
		// Session might already exist - select the cockpit window if it exists, or create it
		if actualName, found := FindWindowByName("cockpit 🎛️"); found {
			SelectWindow(actualName)
		} else {
			// Create a new window in the existing session
			exec.Command("tmux", "new-window", "-t", sessionName, "-n", "cockpit 🎛️", "-c", workDir, selfPath).Run()
		}
	}

	// Exec into tmux attach
	return execSyscall(tmuxPath, []string{"tmux", "attach-session", "-t", sessionName}, os.Environ())
}
