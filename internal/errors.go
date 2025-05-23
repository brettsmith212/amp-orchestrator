package internal

import (
	"errors"
	"fmt"
)

// Common error types for the orchestrator
var (
	ErrGitOperationFailed = errors.New("git operation failed")
	ErrInvalidRepository  = errors.New("invalid repository")
	ErrWorktreeExists     = errors.New("worktree already exists")
	ErrWorktreeNotFound   = errors.New("worktree not found")
	ErrCommitFailed       = errors.New("commit operation failed")
	ErrPushFailed         = errors.New("push operation failed")
	ErrBranchExists       = errors.New("branch already exists")
	ErrBranchNotFound     = errors.New("branch not found")
)

// GitError wraps git-related errors with additional context
type GitError struct {
	Operation string
	Path      string
	Err       error
}

func (e *GitError) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("git %s failed at %s: %v", e.Operation, e.Path, e.Err)
	}
	return fmt.Sprintf("git %s failed: %v", e.Operation, e.Err)
}

func (e *GitError) Unwrap() error {
	return e.Err
}

// NewGitError creates a new GitError
func NewGitError(operation, path string, err error) *GitError {
	return &GitError{
		Operation: operation,
		Path:      path,
		Err:       err,
	}
}

// IsGitError checks if an error is a GitError
func IsGitError(err error) bool {
	var gitErr *GitError
	return errors.As(err, &gitErr)
}