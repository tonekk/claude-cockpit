package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tonekk/claude-cockpit/internal/context"
	"github.com/tonekk/claude-cockpit/internal/tmux"
	"github.com/tonekk/claude-cockpit/internal/tui"
	"github.com/tonekk/claude-cockpit/internal/waiting"
	"github.com/tonekk/claude-cockpit/internal/worktree"
)

// envFlag is a custom flag type for collecting multiple -e KEY=VALUE flags
type envFlag []string

func (e *envFlag) String() string {
	return strings.Join(*e, ", ")
}

func (e *envFlag) Set(value string) error {
	*e = append(*e, value)
	return nil
}

func main() {
	// Handle subcommands for hooks (before flag parsing)
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "notify-waiting":
			handleNotifyWaiting()
			return
		case "clear-waiting":
			handleClearWaiting()
			return
		}
	}

	// CLI flags
	listFlag := flag.Bool("list", false, "List worktrees without TUI")
	serverCmd := flag.String("s", "", "Server command to run in worktrees")
	serverCmdLong := flag.String("server-command", "", "Server command to run in worktrees")
	editor := flag.String("E", "", "Editor command (default: code)")
	editorLong := flag.String("editor", "", "Editor command (default: code)")
	helpFlag := flag.Bool("h", false, "Show help")
	helpFlagLong := flag.Bool("help", false, "Show help")

	// Repeatable -e/--env flags for server environment variables
	var envFlags envFlag
	flag.Var(&envFlags, "e", "Server environment variable (KEY=VALUE), can be repeated")
	flag.Var(&envFlags, "env", "Server environment variable (KEY=VALUE), can be repeated")

	flag.Usage = printUsage
	flag.Parse()

	if *helpFlag || *helpFlagLong {
		printUsage()
		return
	}

	// Resolve server command: flag > env > empty
	finalServerCmd := *serverCmd
	if finalServerCmd == "" {
		finalServerCmd = *serverCmdLong
	}
	if finalServerCmd == "" {
		finalServerCmd = os.Getenv("WD_SERVER_COMMAND")
	}

	// Resolve editor: flag > env > default
	finalEditor := *editor
	if finalEditor == "" {
		finalEditor = *editorLong
	}
	if finalEditor == "" {
		finalEditor = os.Getenv("WD_EDITOR")
	}
	if finalEditor == "" {
		finalEditor = "code"
	}

	// Collect server environment variables
	serverEnv := make(map[string]string)

	// First, collect from WD_ENV_* environment variables
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "WD_ENV_") {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				// Strip "WD_ENV_" prefix from key
				key := strings.TrimPrefix(parts[0], "WD_ENV_")
				serverEnv[key] = parts[1]
			}
		}
	}

	// Then, apply -e/--env flags (override WD_ENV_* if same key)
	for _, e := range envFlags {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			serverEnv[parts[0]] = parts[1]
		}
	}

	// Find the project root (look for .git directory)
	projectRoot, err := findProjectRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Validate .claude directory exists
	claudeDir := filepath.Join(projectRoot, ".claude")
	if _, err := os.Stat(claudeDir); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: .claude directory not found in %s\n", projectRoot)
		fmt.Fprintf(os.Stderr, "Run 'mkdir .claude' to create it, or start from a Claude-enabled project.\n")
		os.Exit(1)
	}

	// Set project root for tmux package so it can set env vars
	tmux.ProjectRoot = projectRoot

	if *listFlag {
		listWorktrees(projectRoot)
		return
	}

	// Ensure we're in a proper tmux session with correct window name
	if needsExec, sessionName := tmux.EnsureCockpitSession(projectRoot); needsExec {
		if err := tmux.ExecIntoSession(sessionName, projectRoot); err != nil {
			fmt.Fprintf(os.Stderr, "Error starting tmux session: %v\n", err)
			os.Exit(1)
		}
		return // ExecIntoSession replaces process, but just in case
	}

	// Create and run the TUI
	config := tui.Config{
		ServerCommand: finalServerCmd,
		ServerEnv:     serverEnv,
		Editor:        finalEditor,
	}
	m := tui.New(projectRoot, config)
	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}

func listWorktrees(projectRoot string) {
	worktrees, err := worktree.List(projectRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing worktrees: %v\n", err)
		os.Exit(1)
	}

	for _, wt := range worktrees {
		status, _ := worktree.GetStatus(wt.Path)

		statusStr := "✓ clean"
		if !status.Clean {
			statusStr = fmt.Sprintf("● %d files", len(status.Files))
		}

		contextStr := "[no context]"
		if context.Exists(projectRoot, wt.Name) {
			contextStr = "[context saved]"
		}

		fmt.Printf("%-10s %-45s %s  %s\n", wt.Name, wt.Branch, statusStr, contextStr)
	}
}

func findProjectRoot() (string, error) {
	return findProjectRootFrom("")
}

func findProjectRootFrom(startDir string) (string, error) {
	// First try git rev-parse
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	if startDir != "" {
		cmd.Dir = startDir
	}
	output, err := cmd.Output()
	if err == nil {
		return strings.TrimSpace(string(output)), nil
	}

	// Fallback: walk up from directory
	var dir string
	if startDir != "" {
		dir = startDir
	} else {
		dir, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("not in a git repository")
}

// hookInput represents the JSON input from Claude Code hooks
type hookInput struct {
	Cwd string `json:"cwd"`
}

// handleNotifyWaiting is called by the Stop hook to mark a session as waiting
func handleNotifyWaiting() {
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
		os.Exit(1)
	}

	var data hookInput
	if err := json.Unmarshal(input, &data); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing JSON: %v\n", err)
		os.Exit(1)
	}

	if data.Cwd == "" {
		fmt.Fprintf(os.Stderr, "No cwd in input\n")
		os.Exit(1)
	}

	// Use WORKTREE_DASHBOARD_PROJECT if set, otherwise find from cwd
	projectRoot := os.Getenv("WORKTREE_DASHBOARD_PROJECT")
	if projectRoot == "" {
		projectRoot, err = findProjectRootFrom(data.Cwd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error finding project root: %v\n", err)
			os.Exit(1)
		}
	}

	// Find the worktree/session root from cwd (Claude may be in a subdirectory)
	sessionRoot, err := findProjectRootFrom(data.Cwd)
	if err != nil {
		sessionRoot = data.Cwd // fallback for non-git directories
	}

	if err := waiting.MarkWaiting(projectRoot, sessionRoot); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking waiting: %v\n", err)
		os.Exit(1)
	}

	// Update tmux window name using session root basename
	name := filepath.Base(sessionRoot)
	_ = tmux.MarkWindowWaiting(name)
}

func printUsage() {
	fmt.Println("claude-cockpit - Your command center for multiple Claude Code sessions")
	fmt.Println()
	fmt.Println("Usage: claude-cockpit [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -s, --server-command <cmd>  Server command to run in worktrees")
	fmt.Println("  -e, --env <KEY=VALUE>       Server env var (can be repeated)")
	fmt.Println("  -E, --editor <cmd>          Editor command (default: code)")
	fmt.Println("  --list                      List worktrees without TUI")
	fmt.Println("  -h, --help                  Show this help message")
	fmt.Println()
	fmt.Println("Environment variables:")
	fmt.Println("  WD_SERVER_COMMAND           Server command (overridden by -s)")
	fmt.Println("  WD_EDITOR                   Editor command (overridden by -E)")
	fmt.Println("  WD_ENV_<KEY>                Server env vars (e.g., WD_ENV_RAILS_ENV=development)")
	fmt.Println()
	fmt.Println("Keys:")
	fmt.Println("  enter    Open worktree in tmux with Claude")
	fmt.Println("  o        Open worktree in editor")
	fmt.Println("  s        Start/stop server in worktree")
	fmt.Println("  l/→      Expand file tree")
	fmt.Println("  h/←      Collapse file tree")
	fmt.Println("  j/k      Navigate up/down")
	fmt.Println("  q        Quit")
}

// handleClearWaiting is called by the UserPromptSubmit hook to clear waiting status
func handleClearWaiting() {
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
		os.Exit(1)
	}

	var data hookInput
	if err := json.Unmarshal(input, &data); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing JSON: %v\n", err)
		os.Exit(1)
	}

	if data.Cwd == "" {
		fmt.Fprintf(os.Stderr, "No cwd in input\n")
		os.Exit(1)
	}

	// Use WORKTREE_DASHBOARD_PROJECT if set, otherwise find from cwd
	projectRoot := os.Getenv("WORKTREE_DASHBOARD_PROJECT")
	if projectRoot == "" {
		var err error
		projectRoot, err = findProjectRootFrom(data.Cwd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error finding project root: %v\n", err)
			os.Exit(1)
		}
	}

	// Find the worktree/session root from cwd (Claude may be in a subdirectory)
	sessionRoot, err := findProjectRootFrom(data.Cwd)
	if err != nil {
		sessionRoot = data.Cwd // fallback for non-git directories
	}

	if err := waiting.ClearWaiting(projectRoot, sessionRoot); err != nil {
		fmt.Fprintf(os.Stderr, "Error clearing waiting: %v\n", err)
		os.Exit(1)
	}

	// Update tmux window name using session root basename
	name := filepath.Base(sessionRoot)
	_ = tmux.ClearWindowWaiting(name)
}
