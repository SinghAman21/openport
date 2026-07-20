package model

import (
	"fmt"
	"strings"
)

// ProtectedPorts is the never-kill list. It matches on explicit ports, ranges,
// and process names so users can protect "whatever postgres is on today".
type ProtectedPorts struct {
	Ports     []int       `json:"ports" yaml:"ports"`
	Ranges    []PortRange `json:"ranges,omitempty" yaml:"ranges,omitempty"`
	Processes []string    `json:"processes,omitempty" yaml:"processes,omitempty"`
}

// DefaultProtectedPorts is what a fresh install starts with: the ports where
// killing the listener tends to lock you out of the machine or break DNS.
func DefaultProtectedPorts() ProtectedPorts {
	return ProtectedPorts{
		Ports: []int{22, 53, 68, 123, 631, 5353},
		Processes: []string{
			"sshd", "systemd", "systemd-resolve", "systemd-resolved",
			"launchd", "init", "dnsmasq", "NetworkManager", "cupsd",
		},
	}
}

// IsProtected reports whether a port/process pair is on the never-kill list,
// along with a human-readable reason for the UI to surface.
func (p ProtectedPorts) IsProtected(port int, process string) (bool, string) {
	for _, x := range p.Ports {
		if x == port {
			return true, fmt.Sprintf("port %d is protected", port)
		}
	}
	for _, r := range p.Ranges {
		if r.Contains(port) {
			label := r.Name
			if label == "" {
				label = fmt.Sprintf("%d-%d", r.From, r.To)
			}
			return true, fmt.Sprintf("port %d is in protected range %s", port, label)
		}
	}
	if process != "" {
		for _, name := range p.Processes {
			if strings.EqualFold(name, process) {
				return true, fmt.Sprintf("process %q is protected", process)
			}
		}
	}
	return false, ""
}

// WithPort returns a copy with port added (idempotent).
func (p ProtectedPorts) WithPort(port int) ProtectedPorts {
	for _, x := range p.Ports {
		if x == port {
			return p
		}
	}
	out := p
	out.Ports = append(append([]int(nil), p.Ports...), port)
	return out
}

// WithoutPort returns a copy with port removed.
func (p ProtectedPorts) WithoutPort(port int) ProtectedPorts {
	out := p
	out.Ports = make([]int, 0, len(p.Ports))
	for _, x := range p.Ports {
		if x != port {
			out.Ports = append(out.Ports, x)
		}
	}
	return out
}
