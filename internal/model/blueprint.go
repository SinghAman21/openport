package model

import (
	"fmt"
	"strings"
	"time"
)

// HealthSpec names the probe used to verify a blueprint service is up.
type HealthSpec string

const (
	HealthNone HealthSpec = ""
	HealthTCP  HealthSpec = "tcp"
	HealthHTTP HealthSpec = "http"
)

// BlueprintService is one service entry inside a Blueprint.
type BlueprintService struct {
	Name       string     `json:"name" yaml:"name"`
	Port       int        `json:"port" yaml:"port"`
	Command    string     `json:"command,omitempty" yaml:"command,omitempty"`
	Health     HealthSpec `json:"health,omitempty" yaml:"health,omitempty"`
	HealthPath string     `json:"health_path,omitempty" yaml:"health_path,omitempty"`
	Optional   bool       `json:"optional,omitempty" yaml:"optional,omitempty"`
}

// Blueprint is a reusable template describing an environment's services and
// the ports they are expected to occupy.
type Blueprint struct {
	Name        string             `json:"name" yaml:"name"`
	Description string             `json:"description,omitempty" yaml:"description,omitempty"`
	Services    []BlueprintService `json:"services" yaml:"services"`
	CreatedAt   time.Time          `json:"created_at" yaml:"-"`
	UpdatedAt   time.Time          `json:"updated_at" yaml:"-"`
}

// Ports returns every port the blueprint expects, in declaration order.
func (b Blueprint) Ports() []int {
	out := make([]int, 0, len(b.Services))
	for _, s := range b.Services {
		out = append(out, s.Port)
	}
	return out
}

// Validate reports whether the blueprint is well-formed enough to persist.
func (b Blueprint) Validate() error {
	if strings.TrimSpace(b.Name) == "" {
		return fmt.Errorf("blueprint name is required")
	}
	if len(b.Services) == 0 {
		return fmt.Errorf("blueprint %q needs at least one service", b.Name)
	}
	seen := make(map[int]string, len(b.Services))
	for _, s := range b.Services {
		if strings.TrimSpace(s.Name) == "" {
			return fmt.Errorf("blueprint %q has a service with no name", b.Name)
		}
		if err := ValidatePort(s.Port); err != nil {
			return fmt.Errorf("service %q: %w", s.Name, err)
		}
		if prev, dup := seen[s.Port]; dup {
			return fmt.Errorf("port %d claimed by both %q and %q", s.Port, prev, s.Name)
		}
		seen[s.Port] = s.Name
	}
	return nil
}
