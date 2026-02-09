package tmux

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const WaitingSuffix = " 🔴"
const ServerPrefix = "🔌 "

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

	// Add 🔌 prefix if server is running for this worktree
	if serverWorktreeName == name {
		RenameWindow(name, ServerPrefix+name)
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

	// Match all prefix/suffix combinations
	candidates := []string{
		name,
		ServerPrefix + name,
		name + WaitingSuffix,
		ServerPrefix + name + WaitingSuffix,
	}

	for _, w := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		for _, c := range candidates {
			if w == c {
				return w, true
			}
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
	// Find window index by name to avoid tmux target parsing issues with special characters
	cmd := exec.Command("tmux", "list-windows", "-F", "#{window_index} #{window_name}")
	output, err := cmd.Output()
	if err != nil {
		return err
	}
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		idx, name, found := strings.Cut(line, " ")
		if found && name == windowName {
			return exec.Command("tmux", "kill-window", "-t", idx).Run()
		}
	}
	return nil
}

// MarkWindowWaiting adds the waiting suffix to a window's name
func MarkWindowWaiting(name string) error {
	currentName, found := FindWindowByName(name)
	if !found {
		return nil
	}
	if strings.Contains(currentName, WaitingSuffix) {
		return nil
	}
	return RenameWindow(currentName, currentName+WaitingSuffix)
}

// ClearWindowWaiting removes the waiting suffix from a window's name
func ClearWindowWaiting(name string) error {
	currentName, found := FindWindowByName(name)
	if !found {
		return nil
	}
	if !strings.Contains(currentName, WaitingSuffix) {
		return nil
	}
	return RenameWindow(currentName, strings.Replace(currentName, WaitingSuffix, "", 1))
}

// IsClaudeRunningInWindow checks if claude is the current command in any pane of the given window
func IsClaudeRunningInWindow(name string) bool {
	actualName, found := FindWindowByName(name)
	if !found {
		return false
	}

	cmd := exec.Command("tmux", "list-panes", "-t", actualName, "-F", "#{pane_current_command}")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line == "claude" {
			return true
		}
	}
	return false
}

// ServerWindowName stores the window name of the running server
var ServerWindowName string

// serverWorktreeName stores the worktree name whose window gets the 🔌 suffix
var serverWorktreeName string

// StartServerWindow creates a new tmux window and runs the server command
// Returns the window name
func StartServerWindow(worktreeName, workDir, serverCommand string, envVars map[string]string) (string, error) {
	// First stop any existing server
	if ServerWindowName != "" {
		StopServerWindow()
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

	windowName := "🖥️ server: " + worktreeName

	// Create new window right after the cockpit window
	cmd := exec.Command("tmux", "new-window", "-a", "-t", "🎛️ cockpit", "-n", windowName, "-c", workDir, finalCommand)
	if err := cmd.Run(); err != nil {
		return "", err
	}

	ServerWindowName = windowName
	serverWorktreeName = worktreeName

	// Mark the worktree window with 🔌 prefix (preserving other suffixes)
	if actualName, found := FindWindowByName(worktreeName); found && !strings.HasPrefix(actualName, ServerPrefix) {
		RenameWindow(actualName, ServerPrefix+actualName)
	}

	return windowName, nil
}

const serverWindowPrefix = "🖥️ server: "

// FindServerWindow scans tmux windows for one matching the server prefix
// Returns the window name and the pane's working directory if found
func FindServerWindow() (windowName string, workDir string, found bool) {
	cmd := exec.Command("tmux", "list-windows", "-F", "#{window_name}\t#{pane_current_path}")
	output, err := cmd.Output()
	if err != nil {
		return "", "", false
	}
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		name, path, ok := strings.Cut(line, "\t")
		if ok && strings.HasPrefix(name, serverWindowPrefix) {
			// Restore worktree name tracking
			serverWorktreeName = strings.TrimPrefix(name, serverWindowPrefix)
			// Ensure the worktree window has the 🔌 prefix
			if actualName, found := FindWindowByName(serverWorktreeName); found && !strings.HasPrefix(actualName, ServerPrefix) {
				RenameWindow(actualName, ServerPrefix+actualName)
			}
			return name, path, true
		}
	}
	return "", "", false
}

// clearServerPrefix removes the 🔌 prefix from the worktree window, preserving other suffixes
func clearServerPrefix() {
	if serverWorktreeName == "" {
		return
	}
	if actualName, found := FindWindowByName(serverWorktreeName); found && strings.HasPrefix(actualName, ServerPrefix) {
		RenameWindow(actualName, strings.TrimPrefix(actualName, ServerPrefix))
	}
}

// StopServerWindow kills the server window
func StopServerWindow() error {
	if ServerWindowName == "" {
		return nil
	}

	clearServerPrefix()

	err := KillWindow(ServerWindowName)
	ServerWindowName = ""
	serverWorktreeName = ""
	return err
}

// IsServerRunning returns true if a server window is currently running
func IsServerRunning() bool {
	if ServerWindowName == "" {
		return false
	}

	if !WindowExists(ServerWindowName) {
		clearServerPrefix()
		if serverWorktreeName != "" {
			serverWorktreeName = ""
		}
		ServerWindowName = ""
		return false
	}

	return true
}

// OpenDiff opens git diff in a horizontal split at the given directory
func OpenDiff(workDir string) error {
	cmd := exec.Command("tmux", "split-window", "-h", "-c", workDir, "git", "diff")
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
		RenameCurrentWindow("🎛️ cockpit")
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
	cmd := exec.Command("tmux", "new-session", "-d", "-s", sessionName, "-c", workDir, "-n", "🎛️ cockpit", selfPath)
	if err := cmd.Run(); err != nil {
		// Session might already exist - select the cockpit window if it exists, or create it
		if actualName, found := FindWindowByName("🎛️ cockpit"); found {
			SelectWindow(actualName)
		} else {
			// Create a new window in the existing session
			exec.Command("tmux", "new-window", "-t", sessionName, "-n", "🎛️ cockpit", "-c", workDir, selfPath).Run()
		}
	}

	// Exec into tmux attach
	return execSyscall(tmuxPath, []string{"tmux", "attach-session", "-t", sessionName}, os.Environ())
}
