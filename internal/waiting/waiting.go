package waiting

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// WaitingSession represents a Claude session waiting for input
type WaitingSession struct {
	Path      string    `json:"path"`
	IssueID   string    `json:"issue_id,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// waitingDir returns the directory for waiting session files
func waitingDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude", "waiting"), nil
}

// pathHash creates a short hash of the path for use as filename
func pathHash(path string) string {
	h := sha256.Sum256([]byte(path))
	return hex.EncodeToString(h[:8]) // First 8 bytes = 16 chars
}

// MarkWaiting records that a Claude session at the given path is waiting for input
func MarkWaiting(cwd string) error {
	dir, err := waitingDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	session := WaitingSession{
		Path:      cwd,
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(session)
	if err != nil {
		return err
	}

	filename := pathHash(cwd) + ".json"
	return os.WriteFile(filepath.Join(dir, filename), data, 0644)
}

// ClearWaiting removes the waiting marker for a Claude session
func ClearWaiting(cwd string) error {
	dir, err := waitingDir()
	if err != nil {
		return err
	}

	filename := pathHash(cwd) + ".json"
	path := filepath.Join(dir, filename)

	err = os.Remove(path)
	if os.IsNotExist(err) {
		return nil // Already cleared
	}
	return err
}

// ListWaiting returns all paths that have sessions waiting for input
func ListWaiting() ([]string, error) {
	dir, err := waitingDir()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil // No waiting sessions
	}
	if err != nil {
		return nil, err
	}

	var paths []string
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}

		var session WaitingSession
		if err := json.Unmarshal(data, &session); err != nil {
			continue
		}

		paths = append(paths, session.Path)
	}

	return paths, nil
}

// IsWaiting checks if a specific path has a waiting session
func IsWaiting(worktreePath string) bool {
	paths, err := ListWaiting()
	if err != nil {
		return false
	}

	for _, p := range paths {
		if p == worktreePath {
			return true
		}
	}
	return false
}
