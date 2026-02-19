package session

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Session represents a stored session with window references.
type Session struct {
	Name      string       `json:"name"`
	Backend   string       `json:"backend"`
	Count     int          `json:"count"`
	Dir       string       `json:"dir"`
	CreatedAt time.Time    `json:"created_at"`
	Windows   []WindowRef  `json:"windows"`
	Worktrees []WorktreeRef `json:"worktrees,omitempty"`
	Status    string       `json:"status,omitempty"`
	RepoPath  string       `json:"repo_path,omitempty"`
}

// WindowRef represents a reference to a spawned window.
type WindowRef struct {
	ID    string `json:"id"`
	Index int    `json:"index"`
}

// WorktreeRef represents a reference to a git worktree.
type WorktreeRef struct {
	Path   string `json:"path"`
	Branch string `json:"branch"`
}

// Store manages session persistence to JSON files.
type Store struct {
	baseDir string
}

// NewStore creates a new Store with the given base directory.
// If baseDir is empty, defaults to ~/.claude-grid/sessions/
func NewStore(baseDir string) *Store {
	if baseDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			baseDir = "~/.claude-grid/sessions"
		} else {
			baseDir = filepath.Join(home, ".claude-grid", "sessions")
		}
	} else {
		baseDir = filepath.Join(baseDir, "sessions")
	}
	return &Store{baseDir: baseDir}
}

// GenerateSessionName generates a unique session name in format "grid-XXXX"
// where XXXX is 4 random hex characters.
func (s *Store) GenerateSessionName() string {
	for {
		b := make([]byte, 2)
		rand.Read(b)
		name := "grid-" + hex.EncodeToString(b)
		
		// Check for collision
		path := filepath.Join(s.baseDir, name+".json")
		if _, err := os.Stat(path); err != nil {
			// File doesn't exist, name is unique
			return name
		}
	}
}

// SaveSession saves a session to disk as JSON.
// Auto-creates the sessions directory if it doesn't exist.
func (s *Store) SaveSession(session Session) error {
	if err := os.MkdirAll(s.baseDir, 0755); err != nil {
		return fmt.Errorf("failed to create sessions directory: %w", err)
	}
	
	path := filepath.Join(s.baseDir, session.Name+".json")
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}
	
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
	}
	
	return nil
}

// UpdateSession overwrites an existing session file with new data.
// Identical to SaveSession but with semantic distinction for updates.
func (s *Store) UpdateSession(session Session) error {
	if err := os.MkdirAll(s.baseDir, 0755); err != nil {
		return fmt.Errorf("failed to create sessions directory: %w", err)
	}
	
	path := filepath.Join(s.baseDir, session.Name+".json")
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}
	
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
	}
	
	return nil
}

// LoadSession loads a session from disk by name.
func (s *Store) LoadSession(name string) (Session, error) {
	path := filepath.Join(s.baseDir, name+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return Session{}, fmt.Errorf("failed to read session file: %w", err)
	}
	
	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return Session{}, fmt.Errorf("failed to unmarshal session: %w", err)
	}
	
	return session, nil
}

// ListSessions returns all sessions from the sessions directory.
func (s *Store) ListSessions() ([]Session, error) {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Session{}, nil
		}
		return nil, fmt.Errorf("failed to read sessions directory: %w", err)
	}
	
	var sessions []Session
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		
		name := entry.Name()[:len(entry.Name())-5]
		session, err := s.LoadSession(name)
		if err != nil {
			continue
		}
		
		sessions = append(sessions, session)
	}
	
	return sessions, nil
}

// DeleteSession removes a session file from disk.
func (s *Store) DeleteSession(name string) error {
	path := filepath.Join(s.baseDir, name+".json")
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to delete session file: %w", err)
	}
	return nil
}
