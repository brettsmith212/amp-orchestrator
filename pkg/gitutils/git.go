package gitutils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/brettsmith212/amp-orchestrator/internal"
)

// GitRepo represents a git repository
type GitRepo struct {
	Path string // Path to the bare repository
}

// NewRepo creates a new GitRepo instance
func NewRepo(repoPath string) *GitRepo {
	return &GitRepo{
		Path: repoPath,
	}
}

// AddWorktree creates a new git worktree for the given branch
// Returns the path to the created worktree
func (r *GitRepo) AddWorktree(worktreePath, branchName string) (string, error) {
	// Ensure the worktree directory doesn't already exist
	if _, err := os.Stat(worktreePath); err == nil {
		return "", internal.NewGitError("add-worktree", worktreePath, internal.ErrWorktreeExists)
	}

	// Create parent directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(worktreePath), 0755); err != nil {
		return "", internal.NewGitError("mkdir", worktreePath, err)
	}

	// Check if branch already exists in the repository
	branchExists, err := r.branchExists(branchName)
	if err != nil {
		return "", err
	}

	var cmd *exec.Cmd
	if branchExists {
		// Checkout existing branch
		cmd = exec.Command("git", "--git-dir", r.Path, "worktree", "add", worktreePath, branchName)
	} else {
		// Create new branch from main/master
		mainBranch, err := r.getMainBranch()
		if err != nil {
			return "", err
		}
		cmd = exec.Command("git", "--git-dir", r.Path, "worktree", "add", "-b", branchName, worktreePath, mainBranch)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", internal.NewGitError("add-worktree", worktreePath, 
			fmt.Errorf("%s: %s", err, strings.TrimSpace(string(output))))
	}

	return worktreePath, nil
}

// RemoveWorktree removes a git worktree
func (r *GitRepo) RemoveWorktree(worktreePath string) error {
	cmd := exec.Command("git", "--git-dir", r.Path, "worktree", "remove", worktreePath, "--force")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return internal.NewGitError("remove-worktree", worktreePath, 
			fmt.Errorf("%s: %s", err, strings.TrimSpace(string(output))))
	}
	return nil
}

// CommitFile adds, commits, and pushes a file to the repository
// Returns the commit hash
func (r *GitRepo) CommitFile(worktreePath, filePath, commitMessage string) (string, error) {
	// Get absolute path to repository before changing directories
	absRepoPath, err := filepath.Abs(r.Path)
	if err != nil {
		return "", internal.NewGitError("abs-path", r.Path, err)
	}

	// Change to worktree directory for git operations
	originalDir, err := os.Getwd()
	if err != nil {
		return "", internal.NewGitError("getwd", "", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(worktreePath); err != nil {
		return "", internal.NewGitError("chdir", worktreePath, err)
	}

	// Add the file
	addCmd := exec.Command("git", "add", filePath)
	if output, err := addCmd.CombinedOutput(); err != nil {
		return "", internal.NewGitError("add", filePath, 
			fmt.Errorf("%s: %s", err, strings.TrimSpace(string(output))))
	}

	// Check if there are changes to commit
	statusCmd := exec.Command("git", "status", "--porcelain")
	statusOutput, err := statusCmd.CombinedOutput()
	if err != nil {
		return "", internal.NewGitError("status", worktreePath, err)
	}
	
	if len(strings.TrimSpace(string(statusOutput))) == 0 {
		return "", internal.NewGitError("commit", worktreePath, fmt.Errorf("no changes to commit"))
	}

	// Commit the file
	commitCmd := exec.Command("git", "commit", "-m", commitMessage)
	if output, err := commitCmd.CombinedOutput(); err != nil {
		return "", internal.NewGitError("commit", worktreePath, 
			fmt.Errorf("%s: %s", err, strings.TrimSpace(string(output))))
	}

	// Get the commit hash
	hashCmd := exec.Command("git", "rev-parse", "HEAD")
	hashOutput, err := hashCmd.CombinedOutput()
	if err != nil {
		return "", internal.NewGitError("rev-parse", worktreePath, err)
	}

	commitHash := strings.TrimSpace(string(hashOutput))

	// Push the commit
	getCurrentBranchCmd := exec.Command("git", "branch", "--show-current")
	branchOutput, err := getCurrentBranchCmd.CombinedOutput()
	if err != nil {
		return "", internal.NewGitError("branch", worktreePath, err)
	}
	
	currentBranch := strings.TrimSpace(string(branchOutput))
	
	// Configure the remote to point to the bare repository
	remoteCmd := exec.Command("git", "remote", "add", "origin", absRepoPath)
	if _, err := remoteCmd.CombinedOutput(); err != nil {
		// Remote might already exist, try to set the URL instead
		remoteCmd = exec.Command("git", "remote", "set-url", "origin", absRepoPath)
		if output, err := remoteCmd.CombinedOutput(); err != nil {
			return "", internal.NewGitError("remote", worktreePath, 
				fmt.Errorf("%s: %s", err, strings.TrimSpace(string(output))))
		}
	}
	
	pushCmd := exec.Command("git", "push", "origin", currentBranch)
	if output, err := pushCmd.CombinedOutput(); err != nil {
		return "", internal.NewGitError("push", worktreePath, 
			fmt.Errorf("%s: %s", err, strings.TrimSpace(string(output))))
	}

	return commitHash, nil
}

// GetCommitCount returns the number of commits on the given branch
func (r *GitRepo) GetCommitCount(branchName string) (int, error) {
	cmd := exec.Command("git", "--git-dir", r.Path, "rev-list", "--count", branchName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, internal.NewGitError("rev-list", r.Path, 
			fmt.Errorf("%s: %s", err, strings.TrimSpace(string(output))))
	}

	var count int
	if _, err := fmt.Sscanf(strings.TrimSpace(string(output)), "%d", &count); err != nil {
		return 0, internal.NewGitError("parse-count", r.Path, err)
	}

	return count, nil
}

// ListBranches returns a list of all branches in the repository
func (r *GitRepo) ListBranches() ([]string, error) {
	cmd := exec.Command("git", "--git-dir", r.Path, "branch", "-a")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, internal.NewGitError("branch", r.Path, err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var branches []string
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Remove the current branch marker (*)
		if strings.HasPrefix(line, "* ") {
			line = line[2:]
		}
		// Skip remote tracking info
		if !strings.Contains(line, "->") {
			branches = append(branches, line)
		}
	}

	return branches, nil
}

// branchExists checks if a branch exists in the repository
func (r *GitRepo) branchExists(branchName string) (bool, error) {
	cmd := exec.Command("git", "--git-dir", r.Path, "show-ref", "--verify", "--quiet", "refs/heads/"+branchName)
	err := cmd.Run()
	if err != nil {
		// Exit code 1 means branch doesn't exist, which is not an error
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return false, nil
		}
		return false, internal.NewGitError("show-ref", r.Path, err)
	}
	return true, nil
}

// getMainBranch determines the main branch (main or master)
func (r *GitRepo) getMainBranch() (string, error) {
	// Try 'main' first (modern default)
	if exists, err := r.branchExists("main"); err != nil {
		return "", err
	} else if exists {
		return "main", nil
	}

	// Fall back to 'master'
	if exists, err := r.branchExists("master"); err != nil {
		return "", err
	} else if exists {
		return "master", nil
	}

	return "", internal.NewGitError("find-main-branch", r.Path, 
		fmt.Errorf("neither 'main' nor 'master' branch found"))
}

// InitBareRepo creates a new bare git repository
func InitBareRepo(repoPath string) error {
	if err := os.MkdirAll(repoPath, 0755); err != nil {
		return internal.NewGitError("mkdir", repoPath, err)
	}

	cmd := exec.Command("git", "init", "--bare", repoPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return internal.NewGitError("init", repoPath, 
			fmt.Errorf("%s: %s", err, strings.TrimSpace(string(output))))
	}

	return nil
}

// CloneRepo clones a repository to create an initial commit
func (r *GitRepo) CreateInitialCommit() error {
	tmpDir, err := os.MkdirTemp("", "git-init-*")
	if err != nil {
		return internal.NewGitError("mktemp", "", err)
	}
	defer os.RemoveAll(tmpDir)

	// Clone the bare repo
	cloneCmd := exec.Command("git", "clone", r.Path, tmpDir+"/repo")
	if output, err := cloneCmd.CombinedOutput(); err != nil {
		return internal.NewGitError("clone", r.Path, 
			fmt.Errorf("%s: %s", err, strings.TrimSpace(string(output))))
	}

	// Change to cloned directory
	repoDir := filepath.Join(tmpDir, "repo")
	originalDir, err := os.Getwd()
	if err != nil {
		return internal.NewGitError("getwd", "", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(repoDir); err != nil {
		return internal.NewGitError("chdir", repoDir, err)
	}

	// Create initial README
	readmeContent := "# Amp Orchestrator Repository\n\nThis repository is managed by the Amp Orchestrator.\n"
	if err := os.WriteFile("README.md", []byte(readmeContent), 0644); err != nil {
		return internal.NewGitError("write-file", "README.md", err)
	}

	// Configure git user (required for commits)
	exec.Command("git", "config", "user.name", "Amp Orchestrator").Run()
	exec.Command("git", "config", "user.email", "orchestrator@localhost").Run()

	// Add, commit, and push
	if output, err := exec.Command("git", "add", "README.md").CombinedOutput(); err != nil {
		return internal.NewGitError("add", "README.md", 
			fmt.Errorf("%s: %s", err, strings.TrimSpace(string(output))))
	}

	if output, err := exec.Command("git", "commit", "-m", "Initial commit").CombinedOutput(); err != nil {
		return internal.NewGitError("commit", repoDir, 
			fmt.Errorf("%s: %s", err, strings.TrimSpace(string(output))))
	}

	if _, err := exec.Command("git", "push", "origin", "main").CombinedOutput(); err != nil {
		// Try master if main fails
		if masterOutput, masterErr := exec.Command("git", "push", "origin", "master").CombinedOutput(); masterErr != nil {
			return internal.NewGitError("push", repoDir, 
				fmt.Errorf("%s: %s", masterErr, strings.TrimSpace(string(masterOutput))))
		}
	}

	return nil
}