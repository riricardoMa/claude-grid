package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Manager struct {
	repoPath     string
	worktreeBase string
}

func NewManager(dir string) (*Manager, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve absolute path for %q: %w", dir, err)
	}

	cmd := exec.Command("git", "-C", absDir, "rev-parse", "--show-toplevel")
	output, err := cmd.CombinedOutput()
	outputStr := strings.TrimSpace(string(output))
	if err != nil {
		return nil, fmt.Errorf("not a git repository: %w (output: %s)", err, outputStr)
	}

	repoPath := strings.TrimSpace(outputStr)
	if repoPath == "" {
		return nil, fmt.Errorf("failed to resolve git repository root (output: %s)", outputStr)
	}

	absRepoPath, err := filepath.Abs(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve absolute repository root for %q: %w", repoPath, err)
	}

	worktreeBase := defaultWorktreeBase()

	return &Manager{
		repoPath:     absRepoPath,
		worktreeBase: worktreeBase,
	}, nil
}

func (m *Manager) CreateWorktree(branchName string) (string, error) {
	if err := os.MkdirAll(m.worktreeBase, 0755); err != nil {
		return "", fmt.Errorf("failed to create worktree base directory %q: %w", m.worktreeBase, err)
	}

	cmdList := exec.Command("git", "-C", m.repoPath, "worktree", "list", "--porcelain")
	listOutput, err := cmdList.CombinedOutput()
	listOutputStr := strings.TrimSpace(string(listOutput))
	if err != nil {
		return "", fmt.Errorf("failed to list worktrees: %w (output: %s)", err, listOutputStr)
	}

	if strings.Contains(listOutputStr, "branch refs/heads/"+branchName) {
		return "", fmt.Errorf("branch %q is already checked out in another worktree. Run `claude-grid clean` to remove stale worktrees (output: %s)", branchName, listOutputStr)
	}

	cmdHead := exec.Command("git", "-C", m.repoPath, "rev-parse", "HEAD")
	headOutput, err := cmdHead.CombinedOutput()
	headOutputStr := strings.TrimSpace(string(headOutput))
	if err != nil {
		return "", fmt.Errorf("failed to resolve HEAD SHA: %w (output: %s)", err, headOutputStr)
	}

	suffix := fmt.Sprintf("%x", time.Now().UnixNano())
	worktreePath := filepath.Join(m.worktreeBase, fmt.Sprintf("%s_%s", branchName, suffix))

	cmdAdd := exec.Command("git", "-C", m.repoPath, "worktree", "add", "-b", branchName, worktreePath, headOutputStr)
	addOutput, err := cmdAdd.CombinedOutput()
	addOutputStr := strings.TrimSpace(string(addOutput))
	if err != nil {
		lowerOutput := strings.ToLower(addOutputStr)
		switch {
		case strings.Contains(lowerOutput, "already checked out"):
			return "", fmt.Errorf("branch %q is already checked out in another worktree. Run `claude-grid clean` to remove stale worktrees: %w (output: %s)", branchName, err, addOutputStr)
		case strings.Contains(lowerOutput, "already exists"):
			return "", fmt.Errorf("branch already exists: %q: %w (output: %s)", branchName, err, addOutputStr)
		default:
			return "", fmt.Errorf("failed to create worktree for branch %q at %q: %w (output: %s)", branchName, worktreePath, err, addOutputStr)
		}
	}

	return worktreePath, nil
}

func (m *Manager) RemoveWorktree(worktreePath string) error {
	if _, err := os.Stat(worktreePath); err == nil {
		cmd := exec.Command("git", "-C", m.repoPath, "worktree", "remove", "--force", worktreePath)
		output, removeErr := cmd.CombinedOutput()
		outputStr := strings.TrimSpace(string(output))
		if removeErr != nil {
			return fmt.Errorf("failed to remove worktree %q: %w (output: %s)", worktreePath, removeErr, outputStr)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to stat worktree path %q: %w", worktreePath, err)
	}

	if err := m.Prune(); err != nil {
		return err
	}

	return nil
}

func (m *Manager) Prune() error {
	cmd := exec.Command("git", "-C", m.repoPath, "worktree", "prune")
	output, err := cmd.CombinedOutput()
	outputStr := strings.TrimSpace(string(output))
	if err != nil {
		return fmt.Errorf("failed to prune worktrees: %w (output: %s)", err, outputStr)
	}
	return nil
}

func (m *Manager) DetectSubmodules() bool {
	cmd := exec.Command("git", "-C", m.repoPath, "submodule", "status")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) != ""
}

func (m *Manager) RepoPath() string {
	return m.repoPath
}

func defaultWorktreeBase() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "~/.claude-grid/worktrees"
	}
	return filepath.Join(home, ".claude-grid", "worktrees")
}
