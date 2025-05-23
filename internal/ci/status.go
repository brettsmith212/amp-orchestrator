package ci

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// Status represents the CI status for a commit
type Status struct {
	Ref       string    `json:"ref"`
	Commit    string    `json:"commit"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Output    string    `json:"output"`
}

// StatusReader provides methods to read CI status files
type StatusReader struct {
	statusDir string
}

// NewStatusReader creates a new StatusReader for the given status directory
func NewStatusReader(statusDir string) *StatusReader {
	return &StatusReader{
		statusDir: statusDir,
	}
}

// GetStatus reads the CI status for a specific commit hash
func (sr *StatusReader) GetStatus(commitHash string) (*Status, error) {
	filePath := filepath.Join(sr.statusDir, commitHash+".json")
	
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("CI status not found for commit %s", commitHash)
		}
		return nil, fmt.Errorf("failed to read CI status file: %w", err)
	}
	
	var status Status
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, fmt.Errorf("failed to parse CI status JSON: %w", err)
	}
	
	return &status, nil
}

// ListStatuses returns all CI statuses in the status directory
func (sr *StatusReader) ListStatuses() ([]*Status, error) {
	var statuses []*Status
	
	err := filepath.WalkDir(sr.statusDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		
		if d.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}
		
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read status file %s: %w", path, err)
		}
		
		var status Status
		if err := json.Unmarshal(data, &status); err != nil {
			return fmt.Errorf("failed to parse status JSON in %s: %w", path, err)
		}
		
		statuses = append(statuses, &status)
		return nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to list CI statuses: %w", err)
	}
	
	return statuses, nil
}

// HasStatus checks if a CI status exists for the given commit
func (sr *StatusReader) HasStatus(commitHash string) bool {
	filePath := filepath.Join(sr.statusDir, commitHash+".json")
	_, err := os.Stat(filePath)
	return err == nil
}

// IsPassing returns true if the CI status for the given commit is "PASS"
func (sr *StatusReader) IsPassing(commitHash string) (bool, error) {
	status, err := sr.GetStatus(commitHash)
	if err != nil {
		return false, err
	}
	
	return status.Status == "PASS", nil
}