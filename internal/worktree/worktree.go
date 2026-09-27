package worktree

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/tonekk/claude-cockpit/internal/tmux"
)

// Worktree represents a git worktree
type Worktree struct {
	Path      string
	Branch    string
	Name      string // directory basename, e.g., "my-feature" or "SH-429"
	GitStatus Status
}

// FileStatus represents the status of a single file
type FileStatus struct {
	Path     string
	Staged   byte // ' ', 'M', 'A', 'D', 'R', 'C', 'U'
	Unstaged byte // ' ', 'M', 'D', '?'
}

// Status represents the git status of a worktree
type Status struct {
	Clean          bool
	Files          []FileStatus
	StagedCount    int
	ModifiedCount  int // unstaged modifications
	UntrackedCount int
	Ahead          int
	Behind         int
}

// Create creates a new worktree with a new branch in .worktrees/
func Create(repoPath, dirName, branchName string) error {
	worktreesDir := filepath.Join(repoPath, ".worktrees")
	if _, err := os.Stat(worktreesDir); os.IsNotExist(err) {
		if err := os.MkdirAll(worktreesDir, 0755); err != nil {
			return err
		}
	}

	// Always branch from origin/main, regardless of what the main worktree has checked out
	fetch := exec.Command("git", "fetch", "origin", "main")
	fetch.Dir = repoPath
	if err := fetch.Run(); err != nil {
		return fmt.Errorf("failed to fetch origin/main: %w", err)
	}

	worktreePath := filepath.Join(worktreesDir, dirName)
	cmd := exec.Command("git", "worktree", "add", "-b", branchName, worktreePath, "origin/main")
	cmd.Dir = repoPath

	return cmd.Run()
}

// Remove discards all changes and removes the worktree
func Remove(wt Worktree) error {
	if wt.Branch == "main" {
		return errors.New("can't remove main worktree")
	}

	if tmux.WindowExists(wt.Name) {
		tmux.KillWindow(wt.Name)
	}

	// Discard all tracked changes
	restore := exec.Command("git", "restore", ".")
	restore.Dir = wt.Path
	if err := restore.Run(); err != nil {
		return fmt.Errorf("git restore failed: %w", err)
	}

	// Remove untracked files and directories
	clean := exec.Command("git", "clean", "-fd")
	clean.Dir = wt.Path
	if err := clean.Run(); err != nil {
		return fmt.Errorf("git clean failed: %w", err)
	}

	cmd := exec.Command("git", "worktree", "remove", wt.Path)
	return cmd.Run()
}

// List returns all worktrees for the given git repository
func List(repoPath string) ([]Worktree, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return parseWorktreeList(string(output))
}

func parseWorktreeList(output string) ([]Worktree, error) {
	var worktrees []Worktree
	var current Worktree

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "worktree ") {
			if current.Path != "" {
				worktrees = append(worktrees, current)
			}
			path := strings.TrimPrefix(line, "worktree ")
			current = Worktree{
				Path: path,
				Name: filepath.Base(path),
			}
		} else if strings.HasPrefix(line, "branch ") {
			current.Branch = strings.TrimPrefix(line, "branch refs/heads/")
		}
	}

	// Don't forget the last worktree
	if current.Path != "" {
		worktrees = append(worktrees, current)
	}

	return worktrees, scanner.Err()
}

// GetStatus fetches the git status for a worktree
func GetStatus(worktreePath string) (Status, error) {
	status := Status{Clean: true}

	// Get modified files with details
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = worktreePath
	output, err := cmd.Output()
	if err != nil {
		return status, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if len(line) < 3 {
			continue
		}

		// Format: "XY PATH" where XY is 2-char status, space, then path
		// X = staging area, Y = working tree
		x := line[0]
		y := line[1]
		// Path starts after "XY " - use position 2 and trim space
		filename := strings.TrimPrefix(line[2:], " ")

		// Handle renames: "R  old -> new"
		if idx := strings.Index(filename, " -> "); idx != -1 {
			filename = filename[idx+4:]
		}

		fs := FileStatus{
			Path:     filename,
			Staged:   x,
			Unstaged: y,
		}
		status.Files = append(status.Files, fs)

		// Count by category
		if x == '?' && y == '?' {
			status.UntrackedCount++
		} else {
			if x != ' ' && x != '?' {
				status.StagedCount++
			}
			if y != ' ' && y != '?' {
				status.ModifiedCount++
			}
		}
	}

	if len(status.Files) > 0 {
		status.Clean = false
	}

	// Get ahead/behind info
	cmd = exec.Command("git", "status", "--branch", "--porcelain=v2")
	cmd.Dir = worktreePath
	output, err = cmd.Output()
	if err != nil {
		return status, nil // Don't fail if this doesn't work
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "# branch.ab ") {
			// Format: # branch.ab +1 -0
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				// Parse ahead
				if strings.HasPrefix(parts[2], "+") {
					var ahead int
					parseIntFromString(parts[2][1:], &ahead)
					status.Ahead = ahead
				}
				// Parse behind
				if strings.HasPrefix(parts[3], "-") {
					var behind int
					parseIntFromString(parts[3][1:], &behind)
					status.Behind = behind
				}
			}
		}
	}

	return status, nil
}

func parseIntFromString(s string, result *int) {
	for _, c := range s {
		if c >= '0' && c <= '9' {
			*result = *result*10 + int(c-'0')
		} else {
			break
		}
	}
}
