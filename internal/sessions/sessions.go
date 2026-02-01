package sessions

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/tonekk/claude-cockpit/internal/tmux"
)

// sessionsFile returns the path to the sessions file
func sessionsFile(projectRoot string) string {
	return filepath.Join(projectRoot, ".claude", "sessions")
}

// Session represents an additional session (non-worktree path)
type Session struct {
	Path string
	Name string // derived from path basename
}

// List returns all additional sessions from the sessions file
func List(projectRoot string) ([]Session, error) {
	file := sessionsFile(projectRoot)
	f, err := os.Open(file)
	if os.IsNotExist(err) {
		return nil, nil // No sessions file yet
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var sessions []Session
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue // Skip empty lines and comments
		}
		sessions = append(sessions, Session{
			Path: line,
			Name: filepath.Base(line),
		})
	}

	return sessions, scanner.Err()
}

// Add adds a new session path to the sessions file
func Add(projectRoot, path string) error {
	// Ensure .claude directory exists
	claudeDir := filepath.Join(projectRoot, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		return err
	}

	// Check if already exists
	sessions, err := List(projectRoot)
	if err != nil {
		return err
	}
	for _, s := range sessions {
		if s.Path == path {
			return nil // Already exists
		}
	}

	// Append to file
	file := sessionsFile(projectRoot)
	f, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(path + "\n")
	return err
}

// Remove removes a session path from the sessions file
func Remove(projectRoot string, session Session) error {
	sessions, err := List(projectRoot)
	if err != nil {
		return err
	}

	// Filter out the path to remove
	var remaining []string
	for _, s := range sessions {
		if s.Path != session.Path {
			remaining = append(remaining, s.Path)
		}
	}

	// Rewrite the file
	file := sessionsFile(projectRoot)
	f, err := os.Create(file)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, p := range remaining {
		if _, err := f.WriteString(p + "\n"); err != nil {
			return err
		}
	}

	// Kill tmux window if open
	if tmux.WindowExists(session.Name) {
		tmux.KillWindow(session.Name)
	}

	return nil
}
