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

// waitingDir returns the directory for waiting session files within a project
func waitingDir(projectRoot string) string {
	return filepath.Join(projectRoot, ".claude", "waiting")
}

// pathHash creates a short hash of the path for use as filename
func pathHash(path string) string {
	h := sha256.Sum256([]byte(path))
	return hex.EncodeToString(h[:8]) // First 8 bytes = 16 chars
}

// MarkWaiting records that a Claude session at the given path is waiting for input
func MarkWaiting(projectRoot, cwd string) error {
	dir := waitingDir(projectRoot)

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
func ClearWaiting(projectRoot, cwd string) error {
	dir := waitingDir(projectRoot)

	filename := pathHash(cwd) + ".json"
	path := filepath.Join(dir, filename)

	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil // Already cleared
	}
	return err
}

// ListWaiting returns all paths that have sessions waiting for input
func ListWaiting(projectRoot string) ([]string, error) {
	dir := waitingDir(projectRoot)

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
func IsWaiting(projectRoot, worktreePath string) bool {
	paths, err := ListWaiting(projectRoot)
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

// ServerState represents the running server state
type ServerState struct {
	PaneID       string `json:"pane_id"`
	WorktreePath string `json:"worktree_path"`
}

// serverStatePath returns the path to the server state file
func serverStatePath(projectRoot string) string {
	return filepath.Join(projectRoot, ".claude", "server.json")
}

// SaveServerState persists the server state to disk
func SaveServerState(projectRoot, paneID, worktreePath string) error {
	dir := filepath.Join(projectRoot, ".claude")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	state := ServerState{
		PaneID:       paneID,
		WorktreePath: worktreePath,
	}

	data, err := json.Marshal(state)
	if err != nil {
		return err
	}

	return os.WriteFile(serverStatePath(projectRoot), data, 0644)
}

// LoadServerState loads the server state from disk
func LoadServerState(projectRoot string) (*ServerState, error) {
	data, err := os.ReadFile(serverStatePath(projectRoot))
	if os.IsNotExist(err) {
		return nil, nil // No state file
	}
	if err != nil {
		return nil, err
	}

	var state ServerState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	return &state, nil
}

// ClearServerState removes the server state file
func ClearServerState(projectRoot string) error {
	err := os.Remove(serverStatePath(projectRoot))
	if os.IsNotExist(err) {
		return nil // Already cleared
	}
	return err
}
