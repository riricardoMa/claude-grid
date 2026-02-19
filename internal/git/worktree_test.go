package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestNewManagerInGitRepo(t *testing.T) {
	repoPath := initGitRepo(t)
	nestedPath := filepath.Join(repoPath, "nested", "dir")
	if err := os.MkdirAll(nestedPath, 0755); err != nil {
		t.Fatalf("failed to create nested path: %v", err)
	}

	manager, err := NewManager(nestedPath)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	repoInfo, err := os.Stat(repoPath)
	if err != nil {
		t.Fatalf("failed to stat repo path: %v", err)
	}
	managerInfo, err := os.Stat(manager.RepoPath())
	if err != nil {
		t.Fatalf("failed to stat manager repo path: %v", err)
	}
	if !os.SameFile(repoInfo, managerInfo) {
		t.Fatalf("RepoPath() = %q, want path to same directory as %q", manager.RepoPath(), repoPath)
	}
}

func TestNewManagerInNonGitRepo(t *testing.T) {
	nonGitPath := t.TempDir()

	_, err := NewManager(nonGitPath)
	if err == nil {
		t.Fatalf("NewManager() error = nil, want error")
	}

	if !strings.Contains(err.Error(), "not a git repository") {
		t.Fatalf("NewManager() error = %q, want contains %q", err.Error(), "not a git repository")
	}
}

func TestCreateWorktree(t *testing.T) {
	repoPath := initGitRepo(t)

	manager, err := NewManager(repoPath)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	manager.worktreeBase = filepath.Join(t.TempDir(), "worktrees")

	worktreePath, err := manager.CreateWorktree("feature-create-worktree")
	if err != nil {
		t.Fatalf("CreateWorktree() error = %v", err)
	}

	if _, err := os.Stat(worktreePath); err != nil {
		t.Fatalf("worktree path does not exist: %v", err)
	}

	branchOutput := runGit(t, repoPath, "branch", "--list", "feature-create-worktree")
	if !strings.Contains(branchOutput, "feature-create-worktree") {
		t.Fatalf("branch list output = %q, want contains %q", branchOutput, "feature-create-worktree")
	}

	currentBranch := strings.TrimSpace(runGit(t, worktreePath, "branch", "--show-current"))
	if currentBranch != "feature-create-worktree" {
		t.Fatalf("worktree branch = %q, want %q", currentBranch, "feature-create-worktree")
	}

	readmePath := filepath.Join(worktreePath, "README.md")
	if _, err := os.Stat(readmePath); err != nil {
		t.Fatalf("README.md missing from worktree: %v", err)
	}

	pathPattern := regexp.MustCompile(`feature-create-worktree_[0-9a-f]+$`)
	if !pathPattern.MatchString(worktreePath) {
		t.Fatalf("worktree path = %q, want suffix matching %q", worktreePath, pathPattern.String())
	}
}

func TestCreateWorktreeBranchAlreadyExists(t *testing.T) {
	repoPath := initGitRepo(t)

	manager, err := NewManager(repoPath)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	manager.worktreeBase = filepath.Join(t.TempDir(), "worktrees")

	runGit(t, repoPath, "branch", "already-exists")

	_, err = manager.CreateWorktree("already-exists")
	if err == nil {
		t.Fatalf("CreateWorktree() error = nil, want error")
	}

	if !strings.Contains(err.Error(), "branch already exists") {
		t.Fatalf("CreateWorktree() error = %q, want contains %q", err.Error(), "branch already exists")
	}
}

func TestCreateWorktreeBranchAlreadyCheckedOut(t *testing.T) {
	repoPath := initGitRepo(t)

	manager, err := NewManager(repoPath)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	manager.worktreeBase = filepath.Join(t.TempDir(), "worktrees")

	firstPath, err := manager.CreateWorktree("already-checked-out")
	if err != nil {
		t.Fatalf("first CreateWorktree() error = %v", err)
	}
	t.Cleanup(func() {
		_ = manager.RemoveWorktree(firstPath)
	})

	_, err = manager.CreateWorktree("already-checked-out")
	if err == nil {
		t.Fatalf("second CreateWorktree() error = nil, want error")
	}

	if !strings.Contains(err.Error(), "already checked out") {
		t.Fatalf("CreateWorktree() error = %q, want contains %q", err.Error(), "already checked out")
	}

	if !strings.Contains(err.Error(), "claude-grid clean") {
		t.Fatalf("CreateWorktree() error = %q, want contains cleanup hint", err.Error())
	}
}

func TestRemoveWorktree(t *testing.T) {
	repoPath := initGitRepo(t)

	manager, err := NewManager(repoPath)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	manager.worktreeBase = filepath.Join(t.TempDir(), "worktrees")

	worktreePath, err := manager.CreateWorktree("remove-me")
	if err != nil {
		t.Fatalf("CreateWorktree() error = %v", err)
	}

	if err := manager.RemoveWorktree(worktreePath); err != nil {
		t.Fatalf("RemoveWorktree() error = %v", err)
	}

	if _, err := os.Stat(worktreePath); !os.IsNotExist(err) {
		t.Fatalf("worktree path still exists after removal, stat error = %v", err)
	}

	worktreeList := runGit(t, repoPath, "worktree", "list", "--porcelain")
	if strings.Contains(worktreeList, worktreePath) {
		t.Fatalf("worktree list still contains removed path %q", worktreePath)
	}
}

func TestRemoveWorktreeMissingDirectory(t *testing.T) {
	repoPath := initGitRepo(t)

	manager, err := NewManager(repoPath)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	manager.worktreeBase = filepath.Join(t.TempDir(), "worktrees")

	worktreePath, err := manager.CreateWorktree("deleted-before-remove")
	if err != nil {
		t.Fatalf("CreateWorktree() error = %v", err)
	}

	if err := os.RemoveAll(worktreePath); err != nil {
		t.Fatalf("failed to delete worktree path before RemoveWorktree(): %v", err)
	}

	if err := manager.RemoveWorktree(worktreePath); err != nil {
		t.Fatalf("RemoveWorktree() with missing dir error = %v", err)
	}

	worktreeList := runGit(t, repoPath, "worktree", "list", "--porcelain")
	if strings.Contains(worktreeList, worktreePath) {
		t.Fatalf("worktree list still contains removed path %q", worktreePath)
	}
}

func TestDetectSubmodulesFalseWhenNone(t *testing.T) {
	repoPath := initGitRepo(t)

	manager, err := NewManager(repoPath)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	if manager.DetectSubmodules() {
		t.Fatalf("DetectSubmodules() = true, want false")
	}
}

func initGitRepo(t *testing.T) string {
	t.Helper()

	repoPath := t.TempDir()
	runGit(t, repoPath, "init")
	runGit(t, repoPath, "config", "user.email", "test@example.com")
	runGit(t, repoPath, "config", "user.name", "Test User")

	readmePath := filepath.Join(repoPath, "README.md")
	if err := os.WriteFile(readmePath, []byte("hello\n"), 0644); err != nil {
		t.Fatalf("failed to write README.md: %v", err)
	}

	runGit(t, repoPath, "add", "README.md")
	runGit(t, repoPath, "commit", "-m", "init")

	return repoPath
}

func runGit(t *testing.T, repoPath string, args ...string) string {
	t.Helper()

	commandArgs := append([]string{"-C", repoPath}, args...)
	cmd := exec.Command("git", commandArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v (output: %s)", args, err, strings.TrimSpace(string(output)))
	}

	return strings.TrimSpace(string(output))
}
