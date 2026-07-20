package model

import "fmt"

// ProjectConfigFile is the per-project config filename looked up in the working
// directory (and its parents).
const ProjectConfigFile = ".openport.yml"

// PortRange is an inclusive lo-hi span, optionally named.
type PortRange struct {
	Name string `json:"name,omitempty" yaml:"name,omitempty"`
	From int    `json:"from" yaml:"from"`
	To   int    `json:"to" yaml:"to"`
}

// Contains reports whether port falls inside the range.
func (r PortRange) Contains(port int) bool {
	return port >= r.From && port <= r.To
}

// Validate reports whether the range is well-formed.
func (r PortRange) Validate() error {
	if err := ValidatePort(r.From); err != nil {
		return err
	}
	if err := ValidatePort(r.To); err != nil {
		return err
	}
	if r.From > r.To {
		return fmt.Errorf("range %d-%d is inverted", r.From, r.To)
	}
	return nil
}

// ProjectConfig is the decoded .openport.yml for a working directory.
type ProjectConfig struct {
	Name          string         `yaml:"name" json:"name"`
	ExpectedPorts []int          `yaml:"expected_ports,omitempty" json:"expected_ports,omitempty"`
	Aliases       map[string]int `yaml:"aliases,omitempty" json:"aliases,omitempty"`
	Ranges        []PortRange    `yaml:"ranges,omitempty" json:"ranges,omitempty"`
	Protected     []int          `yaml:"protected,omitempty" json:"protected,omitempty"`
	Blueprint     string         `yaml:"blueprint,omitempty" json:"blueprint,omitempty"`

	// Dir is where the config was loaded from. Not serialized.
	Dir string `yaml:"-" json:"-"`
}

// AliasList flattens the alias map into sorted-by-port Alias records so the
// rest of the app deals in one alias type.
func (c ProjectConfig) AliasList() []Alias {
	out := make([]Alias, 0, len(c.Aliases))
	for name, port := range c.Aliases {
		out = append(out, Alias{Name: name, Port: port, Note: "from " + ProjectConfigFile})
	}
	return out
}

// Validate reports whether the config is well-formed enough to save.
func (c ProjectConfig) Validate() error {
	for _, p := range c.ExpectedPorts {
		if err := ValidatePort(p); err != nil {
			return fmt.Errorf("expected_ports: %w", err)
		}
	}
	for name, p := range c.Aliases {
		if err := ValidatePort(p); err != nil {
			return fmt.Errorf("alias %q: %w", name, err)
		}
	}
	for _, p := range c.Protected {
		if err := ValidatePort(p); err != nil {
			return fmt.Errorf("protected: %w", err)
		}
	}
	for _, r := range c.Ranges {
		if err := r.Validate(); err != nil {
			return fmt.Errorf("ranges: %w", err)
		}
	}
	return nil
}
