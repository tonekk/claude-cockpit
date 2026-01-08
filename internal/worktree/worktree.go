package worktree

import (
	"bufio"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// Worktree represents a git worktree
type Worktree struct {
	Path      string
	Branch    string
	IssueID   string // e.g., "SH-429"
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
	Clean        bool
	Files        []FileStatus
	StagedCount  int
	ModifiedCount int // unstaged modifications
	UntrackedCount int
	Ahead        int
	Behind       int
}

var issueIDRegex = regexp.MustCompile(`SH-\d+`)

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
			current = Worktree{Path: strings.TrimPrefix(line, "worktree ")}
		} else if strings.HasPrefix(line, "branch ") {
			branch := strings.TrimPrefix(line, "branch refs/heads/")
			current.Branch = branch
			// Extract issue ID from branch name
			if match := issueIDRegex.FindString(branch); match != "" {
				current.IssueID = match
			}
		}
	}

	// Don't forget the last worktree
	if current.Path != "" {
		worktrees = append(worktrees, current)
	}

	// Also try to extract issue ID from path (e.g., .worktrees/SH-429)
	for i := range worktrees {
		if worktrees[i].IssueID == "" {
			dir := filepath.Base(worktrees[i].Path)
			if match := issueIDRegex.FindString(dir); match != "" {
				worktrees[i].IssueID = match
			}
		}
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
