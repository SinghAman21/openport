// Package store owns persistence: JSON state under the XDG config dir plus the
// per-project .openport.yml. It is the only component that touches the disk.
package store

import (
	"github.com/SinghAman21/openport/internal/model"
	"github.com/SinghAman21/openport/internal/sync"
)

// Store is the persistence contract. The service and the UI both depend on
// this interface, never on FileStore directly, which is what lets tests run
// entirely in memory.
type Store interface {
	Aliases() ([]model.Alias, error)
	SaveAlias(model.Alias) error
	DeleteAlias(name string) error

	Groups() ([]model.Group, error)
	SaveGroup(model.Group) error
	DeleteGroup(name string) error

	Blueprints() ([]model.Blueprint, error)
	SaveBlueprint(model.Blueprint) error
	DeleteBlueprint(name string) error

	ProtectedPorts() (model.ProtectedPorts, error)
	SaveProtectedPorts(model.ProtectedPorts) error

	// ProjectConfig reads .openport.yml from dir, walking up to the filesystem
	// root. The bool reports whether one was found.
	ProjectConfig(dir string) (model.ProjectConfig, bool, error)
	SaveProjectConfig(dir string, cfg model.ProjectConfig) error

	AppendHistory(model.HistoryEntry) error
	History(limit int) ([]model.HistoryEntry, error)
	AppendLog(model.LogEntry) error
	Logs(limit int) ([]model.LogEntry, error)

	// Snapshot and ApplySnapshot are the serialization boundary for Pro
	// push/pull. They are implemented on the Free tier too — the sync provider
	// is what decides whether anything leaves the machine.
	Snapshot() (sync.Snapshot, error)
	ApplySnapshot(sync.Snapshot) error
}
