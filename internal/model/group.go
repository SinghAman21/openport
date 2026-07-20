package model

import (
	"fmt"
	"strings"
	"time"
)

// Group is a named set of ports acted on together, e.g. "dev_stack" ->
// {3000, 5432, 6379}.
type Group struct {
	Name      string    `json:"name" yaml:"name"`
	Ports     []int     `json:"ports" yaml:"ports"`
	Note      string    `json:"note,omitempty" yaml:"note,omitempty"`
	CreatedAt time.Time `json:"created_at" yaml:"-"`
}

// Contains reports whether port is a member of the group.
func (g Group) Contains(port int) bool {
	for _, p := range g.Ports {
		if p == port {
			return true
		}
	}
	return false
}

// Validate reports whether the group is well-formed enough to persist.
func (g Group) Validate() error {
	if strings.TrimSpace(g.Name) == "" {
		return fmt.Errorf("group name is required")
	}
	if len(g.Ports) == 0 {
		return fmt.Errorf("group %q needs at least one port", g.Name)
	}
	for _, p := range g.Ports {
		if err := ValidatePort(p); err != nil {
			return err
		}
	}
	return nil
}
