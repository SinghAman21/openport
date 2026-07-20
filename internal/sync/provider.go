// Package sync is the Pro/cloud seam. Core features depend on the
// SyncProvider interface and never on a concrete remote client; the default
// implementation is NoopProvider, so the entire offline path works with zero
// network access.
package sync

import (
	"context"
	"errors"
	"strings"
)

// SyncProvider is everything the Pro tier can do. NoopProvider satisfies it
// offline; RemoteProvider satisfies it once a license is present.
type SyncProvider interface {
	IsPro() bool
	Login(ctx context.Context, provider OAuthProvider) (Session, error)
	Workspaces(ctx context.Context) ([]Workspace, error)
	SelectWorkspace(ctx context.Context, id string) error
	Push(ctx context.Context, snap Snapshot) error
	Pull(ctx context.Context) (Snapshot, error)
	Registry(ctx context.Context) ([]Member, error)
	DetectDrift(ctx context.Context, local Snapshot) ([]Drift, error)
}

// ErrProRequired is returned by every NoopProvider method.
var ErrProRequired = errors.New("this feature requires OpenPort Pro")

// NoopProvider is the zero-network default. Every Pro call fails cleanly with
// ErrProRequired so the UI can show a "Pro" affordance rather than break.
type NoopProvider struct{}

var _ SyncProvider = NoopProvider{}

func (NoopProvider) IsPro() bool { return false }

func (NoopProvider) Login(context.Context, OAuthProvider) (Session, error) {
	return Session{}, ErrProRequired
}
func (NoopProvider) Workspaces(context.Context) ([]Workspace, error) { return nil, ErrProRequired }
func (NoopProvider) SelectWorkspace(context.Context, string) error   { return ErrProRequired }
func (NoopProvider) Push(context.Context, Snapshot) error            { return ErrProRequired }
func (NoopProvider) Pull(context.Context) (Snapshot, error)          { return Snapshot{}, ErrProRequired }
func (NoopProvider) Registry(context.Context) ([]Member, error)      { return nil, ErrProRequired }
func (NoopProvider) DetectDrift(context.Context, Snapshot) ([]Drift, error) {
	return nil, ErrProRequired
}

// Detect returns the provider for the given license token. Absent a license,
// it returns NoopProvider — enabling Pro is a one-line swap here, and nothing
// upstream changes.
func Detect(license string) SyncProvider {
	license = strings.TrimSpace(license)
	if license == "" {
		return NoopProvider{}
	}
	return NewRemoteProvider(license)
}
