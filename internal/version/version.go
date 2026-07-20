// Package version carries build metadata, injected at link time.
package version

import "fmt"

// Overridden via -ldflags at build time; see the Makefile.
var (
	Version = "0.1.0"
	Commit  = "dev"
	Date    = "unknown"
)

// String renders a one-line version banner.
func String() string {
	return fmt.Sprintf("openport %s (%s, built %s)", Version, Commit, Date)
}
