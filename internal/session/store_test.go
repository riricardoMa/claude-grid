package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"
)

func TestGenerateSessionName(t *testing.T) {
	store := NewStore("")
	
	// Test format: grid-XXXX (4 random hex chars)
	name := store.GenerateSessionName()
	
	if !regexp.MustCompile(`^grid-[0-9a-f]{4}$`).MatchString(name) {
		t.Errorf("GenerateSessionName() = %q, want format grid-[0-9a-f]{4}", name)
	}
}

func TestGenerateSessionNameUnique(t *testing.T) {
	tempDir := t.TempDir()
	store := NewStore(tempDir)
	
	// Generate 100 names and check for uniqueness
	names := make(map[string]bool)
	for i := 0; i < 100; i++ {
		name := store.GenerateSessionName()
		if names[name] {
			t.Errorf("GenerateSessionName() produced duplicate: %q", name)
		}
		names[name] = true
	}
	
	if len(names) != 100 {
		t.Errorf("Expected 100 unique names, got %d", len(names))
	}
}

func TestGenerateSessionNameCollisionCheck(t *testing.T) {
	tempDir := t.TempDir()
	store := NewStore(tempDir)
	
	// Create a session file manually
	sessionDir := filepath.Join(tempDir, "sessions")
	os.MkdirAll(sessionDir, 0755)
	
	// Create a session file with a specific name
	existingName := "grid-abcd"
	existingPath := filepath.Join(sessionDir, existingName+".json")
	os.WriteFile(existingPath, []byte("{}"), 0644)
	
	// Mock the random generation to return the existing name first, then a different one
	// We'll test this by ensuring GenerateSessionName avoids collisions
	// For now, just verify the function doesn't crash when files exist
	name := store.GenerateSessionName()
	if name == "" {
		t.Error("GenerateSessionName() returned empty string")
	}
}

func TestSaveSession(t *testing.T) {
	tempDir := t.TempDir()
	store := NewStore(tempDir)
	
	session := Session{
		Name:      "grid-test",
		Backend:   "terminal",
		Count:     2,
		Dir:       "/tmp",
		CreatedAt: time.Now(),
		Windows: []WindowRef{
			{ID: "window-1", Index: 0},
			{ID: "window-2", Index: 1},
		},
	}
	
	err := store.SaveSession(session)
	if err != nil {
		t.Fatalf("SaveSession() error = %v", err)
	}
	
	// Verify file was created
	sessionPath := filepath.Join(tempDir, "sessions", "grid-test.json")
	if _, err := os.Stat(sessionPath); err != nil {
		t.Errorf("Session file not created at %s: %v", sessionPath, err)
	}
	
	// Verify file contents
	data, err := os.ReadFile(sessionPath)
	if err != nil {
		t.Fatalf("Failed to read session file: %v", err)
	}
	
	var loaded Session
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Failed to unmarshal session: %v", err)
	}
	
	if loaded.Name != session.Name {
		t.Errorf("Session.Name = %q, want %q", loaded.Name, session.Name)
	}
	if loaded.Backend != session.Backend {
		t.Errorf("Session.Backend = %q, want %q", loaded.Backend, session.Backend)
	}
	if loaded.Count != session.Count {
		t.Errorf("Session.Count = %d, want %d", loaded.Count, session.Count)
	}
}

func TestLoadSession(t *testing.T) {
	tempDir := t.TempDir()
	store := NewStore(tempDir)
	
	// Create a session file
	sessionDir := filepath.Join(tempDir, "sessions")
	os.MkdirAll(sessionDir, 0755)
	
	session := Session{
		Name:      "grid-load",
		Backend:   "warp",
		Count:     4,
		Dir:       "/home/user",
		CreatedAt: time.Now(),
		Windows: []WindowRef{
			{ID: "1", Index: 0},
			{ID: "2", Index: 1},
		},
	}
	
	data, _ := json.Marshal(session)
	sessionPath := filepath.Join(sessionDir, "grid-load.json")
	os.WriteFile(sessionPath, data, 0644)
	
	// Load the session
	loaded, err := store.LoadSession("grid-load")
	if err != nil {
		t.Fatalf("LoadSession() error = %v", err)
	}
	
	if loaded.Name != session.Name {
		t.Errorf("Loaded session Name = %q, want %q", loaded.Name, session.Name)
	}
	if loaded.Backend != session.Backend {
		t.Errorf("Loaded session Backend = %q, want %q", loaded.Backend, session.Backend)
	}
	if loaded.Count != session.Count {
		t.Errorf("Loaded session Count = %d, want %d", loaded.Count, session.Count)
	}
	if len(loaded.Windows) != len(session.Windows) {
		t.Errorf("Loaded session Windows count = %d, want %d", len(loaded.Windows), len(session.Windows))
	}
}

func TestLoadSessionNotFound(t *testing.T) {
	tempDir := t.TempDir()
	store := NewStore(tempDir)
	
	_, err := store.LoadSession("nonexistent")
	if err == nil {
		t.Error("LoadSession() expected error for nonexistent session, got nil")
	}
}

func TestListSessions(t *testing.T) {
	tempDir := t.TempDir()
	store := NewStore(tempDir)
	
	// Create multiple session files
	sessionDir := filepath.Join(tempDir, "sessions")
	os.MkdirAll(sessionDir, 0755)
	
	sessions := []Session{
		{Name: "grid-one", Backend: "terminal", Count: 1, Dir: "/tmp", CreatedAt: time.Now()},
		{Name: "grid-two", Backend: "warp", Count: 2, Dir: "/home", CreatedAt: time.Now()},
		{Name: "grid-three", Backend: "terminal", Count: 3, Dir: "/var", CreatedAt: time.Now()},
	}
	
	for _, s := range sessions {
		data, _ := json.Marshal(s)
		path := filepath.Join(sessionDir, s.Name+".json")
		os.WriteFile(path, data, 0644)
	}
	
	// List sessions
	listed, err := store.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	
	if len(listed) != 3 {
		t.Errorf("ListSessions() returned %d sessions, want 3", len(listed))
	}
	
	// Verify all sessions are present
	names := make(map[string]bool)
	for _, s := range listed {
		names[s.Name] = true
	}
	
	for _, s := range sessions {
		if !names[s.Name] {
			t.Errorf("Session %q not found in list", s.Name)
		}
	}
}

func TestListSessionsEmpty(t *testing.T) {
	tempDir := t.TempDir()
	store := NewStore(tempDir)
	
	// List sessions from empty directory
	listed, err := store.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	
	if len(listed) != 0 {
		t.Errorf("ListSessions() returned %d sessions, want 0", len(listed))
	}
}

func TestDeleteSession(t *testing.T) {
	tempDir := t.TempDir()
	store := NewStore(tempDir)
	
	// Create a session file
	sessionDir := filepath.Join(tempDir, "sessions")
	os.MkdirAll(sessionDir, 0755)
	
	session := Session{
		Name:      "grid-delete",
		Backend:   "terminal",
		Count:     1,
		Dir:       "/tmp",
		CreatedAt: time.Now(),
	}
	
	data, _ := json.Marshal(session)
	sessionPath := filepath.Join(sessionDir, "grid-delete.json")
	os.WriteFile(sessionPath, data, 0644)
	
	// Verify file exists
	if _, err := os.Stat(sessionPath); err != nil {
		t.Fatalf("Session file not created: %v", err)
	}
	
	// Delete the session
	err := store.DeleteSession("grid-delete")
	if err != nil {
		t.Fatalf("DeleteSession() error = %v", err)
	}
	
	// Verify file is deleted
	if _, err := os.Stat(sessionPath); err == nil {
		t.Error("Session file still exists after DeleteSession()")
	} else if !os.IsNotExist(err) {
		t.Errorf("Unexpected error checking deleted file: %v", err)
	}
}

func TestDeleteSessionNotFound(t *testing.T) {
	tempDir := t.TempDir()
	store := NewStore(tempDir)
	
	err := store.DeleteSession("nonexistent")
	if err == nil {
		t.Error("DeleteSession() expected error for nonexistent session, got nil")
	}
}

func TestAutoCreateDirectory(t *testing.T) {
	tempDir := t.TempDir()
	// Use a non-existent subdirectory
	nonExistentDir := filepath.Join(tempDir, "subdir", "sessions")
	store := NewStore(filepath.Join(tempDir, "subdir"))
	
	session := Session{
		Name:      "grid-mkdir",
		Backend:   "terminal",
		Count:     1,
		Dir:       "/tmp",
		CreatedAt: time.Now(),
	}
	
	err := store.SaveSession(session)
	if err != nil {
		t.Fatalf("SaveSession() error = %v", err)
	}
	
	// Verify directory was created
	if _, err := os.Stat(nonExistentDir); err != nil {
		t.Errorf("Directory not auto-created at %s: %v", nonExistentDir, err)
	}
	
	// Verify file exists
	sessionPath := filepath.Join(nonExistentDir, "grid-mkdir.json")
	if _, err := os.Stat(sessionPath); err != nil {
		t.Errorf("Session file not created: %v", err)
	}
}

func TestSessionCRUDLifecycle(t *testing.T) {
	tempDir := t.TempDir()
	store := NewStore(tempDir)
	
	// CREATE
	session := Session{
		Name:      "grid-lifecycle",
		Backend:   "terminal",
		Count:     2,
		Dir:       "/tmp",
		CreatedAt: time.Now(),
		Windows: []WindowRef{
			{ID: "w1", Index: 0},
			{ID: "w2", Index: 1},
		},
	}
	
	if err := store.SaveSession(session); err != nil {
		t.Fatalf("SaveSession() error = %v", err)
	}
	
	// READ
	loaded, err := store.LoadSession("grid-lifecycle")
	if err != nil {
		t.Fatalf("LoadSession() error = %v", err)
	}
	
	if loaded.Name != session.Name || loaded.Count != session.Count {
		t.Error("Loaded session doesn't match saved session")
	}
	
	// LIST
	sessions, err := store.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	
	found := false
	for _, s := range sessions {
		if s.Name == "grid-lifecycle" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Session not found in list after save")
	}
	
	// DELETE
	if err := store.DeleteSession("grid-lifecycle"); err != nil {
		t.Fatalf("DeleteSession() error = %v", err)
	}
	
	// Verify deletion
	_, err = store.LoadSession("grid-lifecycle")
	if err == nil {
		t.Error("Session still exists after delete")
	}
}
