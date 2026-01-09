package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/foodstarter/worktree-dashboard/internal/context"
	"github.com/foodstarter/worktree-dashboard/internal/tmux"
	"github.com/foodstarter/worktree-dashboard/internal/tui"
	"github.com/foodstarter/worktree-dashboard/internal/waiting"
	"github.com/foodstarter/worktree-dashboard/internal/worktree"
)

var issueIDRegex = regexp.MustCompile(`SH-\d+`)

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
	editor := flag.String("e", "", "Editor command (default: code)")
	editorLong := flag.String("editor", "", "Editor command (default: code)")
	helpFlag := flag.Bool("h", false, "Show help")
	helpFlagLong := flag.Bool("help", false, "Show help")

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
		finalServerCmd = os.Getenv("WORKTREE_DASHBOARD_SERVER_COMMAND")
	}

	// Resolve editor: flag > env > default
	finalEditor := *editor
	if finalEditor == "" {
		finalEditor = *editorLong
	}
	if finalEditor == "" {
		finalEditor = os.Getenv("WORKTREE_DASHBOARD_EDITOR")
	}
	if finalEditor == "" {
		finalEditor = "code"
	}

	// Find the project root (look for .git directory)
	projectRoot, err := findProjectRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if *listFlag {
		listWorktrees(projectRoot)
		return
	}

	// Create and run the TUI
	config := tui.Config{
		ServerCommand: finalServerCmd,
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

		issueID := wt.IssueID
		if issueID == "" {
			issueID = "(main)"
		}

		statusStr := "✓ clean"
		if !status.Clean {
			statusStr = fmt.Sprintf("● %d files", len(status.Files))
		}

		contextStr := "[no context]"
		if wt.IssueID != "" && context.Exists(projectRoot, wt.IssueID) {
			contextStr = "[context saved]"
		}

		fmt.Printf("%-10s %-45s %s  %s\n", issueID, wt.Branch, statusStr, contextStr)
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

	projectRoot, err := findProjectRootFrom(data.Cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding project root: %v\n", err)
		os.Exit(1)
	}

	if err := waiting.MarkWaiting(projectRoot, data.Cwd); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking waiting: %v\n", err)
		os.Exit(1)
	}

	// Update tmux window name if we can find an issue ID
	if issueID := issueIDRegex.FindString(data.Cwd); issueID != "" {
		_ = tmux.MarkWindowWaiting(issueID)
	}
}

func printUsage() {
	fmt.Println("worktree-dashboard - TUI for managing git worktrees with Claude Code integration")
	fmt.Println()
	fmt.Println("Usage: worktree-dashboard [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -s, --server-command <cmd>  Server command to run in worktrees")
	fmt.Println("  -e, --editor <cmd>          Editor command (default: code)")
	fmt.Println("  --list                      List worktrees without TUI")
	fmt.Println("  -h, --help                  Show this help message")
	fmt.Println()
	fmt.Println("Environment variables:")
	fmt.Println("  WORKTREE_DASHBOARD_SERVER_COMMAND  Server command (overridden by -s)")
	fmt.Println("  WORKTREE_DASHBOARD_EDITOR          Editor command (overridden by -e)")
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

	projectRoot, err := findProjectRootFrom(data.Cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding project root: %v\n", err)
		os.Exit(1)
	}

	if err := waiting.ClearWaiting(projectRoot, data.Cwd); err != nil {
		fmt.Fprintf(os.Stderr, "Error clearing waiting: %v\n", err)
		os.Exit(1)
	}

	// Update tmux window name if we can find an issue ID
	if issueID := issueIDRegex.FindString(data.Cwd); issueID != "" {
		_ = tmux.ClearWindowWaiting(issueID)
	}
}
