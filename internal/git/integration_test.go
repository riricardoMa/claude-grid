//go:build integration

package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestFullLifecycle(t *testing.T) {
	repoPath := integrationInitGitRepo(t)

	manager, err := NewManager(repoPath)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	manager.worktreeBase = filepath.Join(t.TempDir(), "worktrees")

	prefix := GenerateBranchPrefix()
	if err := ValidateBranchPrefix(prefix); err != nil {
		t.Fatalf("GenerateBranchPrefix() produced invalid prefix %q: %v", prefix, err)
	}

	branches := []string{
		fmt.Sprintf("%s-1", prefix),
		fmt.Sprintf("%s-2", prefix),
		fmt.Sprintf("%s-3", prefix),
	}

	worktreePaths := make([]string, 0, len(branches))
	for _, branch := range branches {
		worktreePath, createErr := manager.CreateWorktree(branch)
		if createErr != nil {
			t.Fatalf("CreateWorktree(%q) error = %v", branch, createErr)
		}
		worktreePaths = append(worktreePaths, worktreePath)
	}

	worktreeList := integrationRunGit(t, repoPath, "worktree", "list", "--porcelain")
	for i, branch := range branches {
		if _, statErr := os.Stat(worktreePaths[i]); statErr != nil {
			t.Fatalf("worktree path %q does not exist: %v", worktreePaths[i], statErr)
		}

		branchOutput := integrationRunGit(t, repoPath, "branch", "--list", branch)
		if !strings.Contains(branchOutput, branch) {
			t.Fatalf("git branch --list %q output = %q, want contains %q", branch, branchOutput, branch)
		}

		if !strings.Contains(worktreeList, "branch refs/heads/"+branch) {
			t.Fatalf("git worktree list output missing branch %q", branch)
		}
	}

	isolatedFile := "integration-isolation.txt"
	isolatedPath := filepath.Join(worktreePaths[0], isolatedFile)
	if writeErr := os.WriteFile(isolatedPath, []byte("only in worktree one\n"), 0644); writeErr != nil {
		t.Fatalf("failed to write isolated file: %v", writeErr)
	}

	for i := 1; i < len(worktreePaths); i++ {
		otherPath := filepath.Join(worktreePaths[i], isolatedFile)
		if _, statErr := os.Stat(otherPath); !os.IsNotExist(statErr) {
			t.Fatalf("expected %q to be absent in worktree %d, statErr = %v", isolatedFile, i+1, statErr)
		}
	}

	if _, statErr := os.Stat(filepath.Join(repoPath, isolatedFile)); !os.IsNotExist(statErr) {
		t.Fatalf("expected %q to be absent in main repo, statErr = %v", isolatedFile, statErr)
	}

	for _, worktreePath := range worktreePaths {
		if removeErr := manager.RemoveWorktree(worktreePath); removeErr != nil {
			t.Fatalf("RemoveWorktree(%q) error = %v", worktreePath, removeErr)
		}
	}

	for _, worktreePath := range worktreePaths {
		if _, statErr := os.Stat(worktreePath); !os.IsNotExist(statErr) {
			t.Fatalf("worktree path %q still exists after removal, statErr = %v", worktreePath, statErr)
		}
	}

	postCleanupList := integrationRunGit(t, repoPath, "worktree", "list", "--porcelain")
	paths := parseWorktreePaths(postCleanupList)
	if len(paths) != 1 {
		t.Fatalf("git worktree list has %d entries after cleanup, want 1 (main only)", len(paths))
	}

	repoInfo, repoInfoErr := os.Stat(repoPath)
	if repoInfoErr != nil {
		t.Fatalf("failed to stat repo path: %v", repoInfoErr)
	}
	mainInfo, mainInfoErr := os.Stat(paths[0])
	if mainInfoErr != nil {
		t.Fatalf("failed to stat remaining worktree path %q: %v", paths[0], mainInfoErr)
	}
	if !os.SameFile(repoInfo, mainInfo) {
		t.Fatalf("remaining worktree path = %q, want same directory as repo path %q", paths[0], repoPath)
	}

	for _, worktreePath := range worktreePaths {
		if strings.Contains(postCleanupList, worktreePath) {
			t.Fatalf("git worktree list still contains removed path %q", worktreePath)
		}
	}

	for _, branch := range branches {
		branchOutput := integrationRunGit(t, repoPath, "branch", "--list", branch)
		if !strings.Contains(branchOutput, branch) {
			t.Fatalf("branch %q missing after cleanup, output = %q", branch, branchOutput)
		}
	}
}

func TestErrorPaths(t *testing.T) {
	testCases := []struct {
		name         string
		kind         string
		input        string
		wantContains string
	}{
		{
			name:         "NewManager on non-git directory",
			kind:         "new-manager",
			wantContains: "not a git repository",
		},
		{
			name:         "CreateWorktree with branch already checked out",
			kind:         "duplicate-worktree-branch",
			wantContains: "already checked out",
		},
		{
			name:         "ValidateBranchPrefix empty",
			kind:         "validate-prefix",
			input:        "",
			wantContains: "cannot be empty",
		},
		{
			name:         "ValidateBranchPrefix contains spaces",
			kind:         "validate-prefix",
			input:        "bad name",
			wantContains: "cannot contain spaces",
		},
		{
			name:         "ValidateBranchPrefix forbidden character",
			kind:         "validate-prefix",
			input:        "bad~name",
			wantContains: "forbidden characters",
		},
		{
			name:         "ValidateBranchPrefix double dots",
			kind:         "validate-prefix",
			input:        "bad..name",
			wantContains: "double dots",
		},
		{
			name:         "ValidateBranchPrefix leading slash",
			kind:         "validate-prefix",
			input:        "/badname",
			wantContains: "cannot start with a forward slash",
		},
		{
			name:         "ValidateBranchPrefix consecutive slashes",
			kind:         "validate-prefix",
			input:        "bad//name",
			wantContains: "consecutive forward slashes",
		},
		{
			name:         "ValidateBranchPrefix leading dot",
			kind:         "validate-prefix",
			input:        ".badname",
			wantContains: "cannot start with a dot",
		},
		{
			name:         "ValidateBranchPrefix trailing dot",
			kind:         "validate-prefix",
			input:        "badname.",
			wantContains: "cannot end with a dot",
		},
		{
			name:         "ValidateBranchPrefix leading hyphen",
			kind:         "validate-prefix",
			input:        "-badname",
			wantContains: "cannot start with a hyphen",
		},
		{
			name:         "ValidateBranchPrefix trailing hyphen",
			kind:         "validate-prefix",
			input:        "badname-",
			wantContains: "cannot end with a hyphen",
		},
		{
			name:         "ValidateBranchPrefix non-ASCII printable",
			kind:         "validate-prefix",
			input:        "bad\x01name",
			wantContains: "non-ASCII printable",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var err error

			switch tc.kind {
			case "new-manager":
				_, err = NewManager(t.TempDir())
			case "duplicate-worktree-branch":
				repoPath := integrationInitGitRepo(t)
				manager, newErr := NewManager(repoPath)
				if newErr != nil {
					t.Fatalf("NewManager() setup error = %v", newErr)
				}
				manager.worktreeBase = filepath.Join(t.TempDir(), "worktrees")

				firstPath, firstErr := manager.CreateWorktree("already-checked-out-integration")
				if firstErr != nil {
					t.Fatalf("first CreateWorktree() setup error = %v", firstErr)
				}
				t.Cleanup(func() {
					_ = manager.RemoveWorktree(firstPath)
				})

				_, err = manager.CreateWorktree("already-checked-out-integration")
			case "validate-prefix":
				err = ValidateBranchPrefix(tc.input)
			default:
				t.Fatalf("unknown test kind %q", tc.kind)
			}

			if err == nil {
				t.Fatalf("expected error for test kind %q", tc.kind)
			}

			if !strings.Contains(err.Error(), tc.wantContains) {
				t.Fatalf("error = %q, want contains %q", err.Error(), tc.wantContains)
			}
		})
	}
}

func integrationInitGitRepo(t *testing.T) string {
	t.Helper()

	repoPath := t.TempDir()
	integrationRunGit(t, repoPath, "init")
	integrationRunGit(t, repoPath, "config", "user.email", "test@example.com")
	integrationRunGit(t, repoPath, "config", "user.name", "Test User")

	readmePath := filepath.Join(repoPath, "README.md")
	if err := os.WriteFile(readmePath, []byte("hello\n"), 0644); err != nil {
		t.Fatalf("failed to write README.md: %v", err)
	}

	integrationRunGit(t, repoPath, "add", "README.md")
	integrationRunGit(t, repoPath, "commit", "-m", "init")

	return repoPath
}

func integrationRunGit(t *testing.T, repoPath string, args ...string) string {
	t.Helper()

	commandArgs := append([]string{"-C", repoPath}, args...)
	cmd := exec.Command("git", commandArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v (output: %s)", args, err, strings.TrimSpace(string(output)))
	}

	return strings.TrimSpace(string(output))
}

func parseWorktreePaths(porcelain string) []string {
	lines := strings.Split(porcelain, "\n")
	paths := make([]string, 0)
	for _, line := range lines {
		if strings.HasPrefix(line, "worktree ") {
			paths = append(paths, strings.TrimPrefix(line, "worktree "))
		}
	}
	return paths
}
