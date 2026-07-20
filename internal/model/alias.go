package model

import (
	"fmt"
	"strings"
	"time"
)

// Alias binds a human-friendly name to a port, e.g. "frontend" -> 3000.
type Alias struct {
	Name      string    `json:"name" yaml:"name"`
	Port      int       `json:"port" yaml:"port"`
	Note      string    `json:"note,omitempty" yaml:"note,omitempty"`
	CreatedAt time.Time `json:"created_at" yaml:"-"`
}

// Validate reports whether the alias is well-formed enough to persist.
func (a Alias) Validate() error {
	if strings.TrimSpace(a.Name) == "" {
		return fmt.Errorf("alias name is required")
	}
	if strings.ContainsAny(a.Name, " \t\n") {
		return fmt.Errorf("alias name %q must not contain whitespace", a.Name)
	}
	return ValidatePort(a.Port)
}

// ValidatePort is the single place port bounds are enforced.
func ValidatePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("port %d out of range (1-65535)", port)
	}
	return nil
}
