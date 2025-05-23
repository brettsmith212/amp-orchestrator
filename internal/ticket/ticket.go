package ticket

import (
	"errors"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Ticket represents a feature request or task to be completed by an agent
type Ticket struct {
	ID          string    `yaml:"id" json:"id"`
	Title       string    `yaml:"title" json:"title"`
	Description string    `yaml:"description" json:"description"`
	Priority    int       `yaml:"priority" json:"priority"`
	Locks       []string  `yaml:"locks,omitempty" json:"locks,omitempty"`
	Dependencies []string `yaml:"dependencies,omitempty" json:"dependencies,omitempty"`
	EstimateMin int       `yaml:"estimate_min,omitempty" json:"estimate_min,omitempty"`
	Tags        []string  `yaml:"tags,omitempty" json:"tags,omitempty"`
	CreatedAt   time.Time `yaml:"created_at,omitempty" json:"created_at,omitempty"`
	UpdatedAt   time.Time `yaml:"updated_at,omitempty" json:"updated_at,omitempty"`
}

// Load loads a ticket from a YAML file
func Load(filepath string) (*Ticket, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read ticket file %s: %w", filepath, err)
	}

	var ticket Ticket
	if err := yaml.Unmarshal(data, &ticket); err != nil {
		return nil, fmt.Errorf("failed to parse YAML in %s: %w", filepath, err)
	}

	// Set timestamps if not provided
	now := time.Now()
	if ticket.CreatedAt.IsZero() {
		ticket.CreatedAt = now
	}
	if ticket.UpdatedAt.IsZero() {
		ticket.UpdatedAt = now
	}

	// Validate required fields
	if err := ticket.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed for ticket in %s: %w", filepath, err)
	}

	return &ticket, nil
}

// LoadFromBytes loads a ticket from YAML bytes
func LoadFromBytes(data []byte) (*Ticket, error) {
	var ticket Ticket
	if err := yaml.Unmarshal(data, &ticket); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Set timestamps if not provided
	now := time.Now()
	if ticket.CreatedAt.IsZero() {
		ticket.CreatedAt = now
	}
	if ticket.UpdatedAt.IsZero() {
		ticket.UpdatedAt = now
	}

	// Validate required fields
	if err := ticket.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return &ticket, nil
}

// Validate checks that all required fields are present and valid
func (t *Ticket) Validate() error {
	if t.ID == "" {
		return errors.New("ticket ID is required")
	}
	
	if t.Title == "" {
		return errors.New("ticket title is required")
	}
	
	if t.Description == "" {
		return errors.New("ticket description is required")
	}
	
	if t.Priority < 1 || t.Priority > 5 {
		return errors.New("ticket priority must be between 1 and 5")
	}
	
	return nil
}

// ToYAML returns the ticket as YAML bytes
func (t *Ticket) ToYAML() ([]byte, error) {
	return yaml.Marshal(t)
}