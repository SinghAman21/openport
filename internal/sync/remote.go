package sync

import (
	"context"
	"time"
)

// DefaultEndpoint is where a real deployment would talk to. No Free code path
// ever reaches it.
const DefaultEndpoint = "https://api.openport.dev"

// RemoteProvider is the Pro implementation. The network calls are deliberately
// stubbed for this build: the interface, types, gating and view wiring are
// complete, so lighting Pro up is a matter of filling these method bodies in.
type RemoteProvider struct {
	License   string
	Endpoint  string
	workspace string
	session   Session
}

var _ SyncProvider = (*RemoteProvider)(nil)

// NewRemoteProvider builds a Pro provider for a license token.
func NewRemoteProvider(license string) *RemoteProvider {
	return &RemoteProvider{License: license, Endpoint: DefaultEndpoint}
}

// IsPro reports true whenever a license token is present. Pro UI is gated on
// this and nothing else.
func (r *RemoteProvider) IsPro() bool { return r != nil && r.License != "" }

// Workspace returns the currently selected workspace ID, if any.
func (r *RemoteProvider) Workspace() string { return r.workspace }

// Session returns the active session, zero-valued when not logged in.
func (r *RemoteProvider) Session() Session { return r.session }

// TODO(pro): exchange the OAuth device code with the endpoint.
func (r *RemoteProvider) Login(ctx context.Context, provider OAuthProvider) (Session, error) {
	return Session{}, ErrProRequired
}

// TODO(pro): GET {endpoint}/v1/workspaces
func (r *RemoteProvider) Workspaces(ctx context.Context) ([]Workspace, error) {
	return nil, ErrProRequired
}

// SelectWorkspace records the active workspace locally; the remote call that
// validates membership is still to come.
func (r *RemoteProvider) SelectWorkspace(ctx context.Context, id string) error {
	if !r.IsPro() {
		return ErrProRequired
	}
	r.workspace = id
	return nil
}

// TODO(pro): PUT {endpoint}/v1/workspaces/{id}/snapshot
func (r *RemoteProvider) Push(ctx context.Context, snap Snapshot) error {
	return ErrProRequired
}

// TODO(pro): GET {endpoint}/v1/workspaces/{id}/snapshot
func (r *RemoteProvider) Pull(ctx context.Context) (Snapshot, error) {
	return Snapshot{}, ErrProRequired
}

// TODO(pro): GET {endpoint}/v1/workspaces/{id}/registry
func (r *RemoteProvider) Registry(ctx context.Context) ([]Member, error) {
	return nil, ErrProRequired
}

// TODO(pro): compare the local snapshot against the workspace blueprint
// server-side; DiffSnapshots below is the local half of that computation.
func (r *RemoteProvider) DetectDrift(ctx context.Context, local Snapshot) ([]Drift, error) {
	return nil, ErrProRequired
}

// DiffSnapshots compares a local snapshot against a reference one and reports
// blueprint-level divergence. It is pure and network-free, so the Free tier
// reuses it for local drift badges against locally-defined blueprints.
func DiffSnapshots(local, reference Snapshot) []Drift {
	refPorts := map[string]int{}
	for _, bp := range reference.Blueprints {
		for _, s := range bp.Services {
			refPorts[s.Name] = s.Port
		}
	}
	localPorts := map[string]int{}
	for _, bp := range local.Blueprints {
		for _, s := range bp.Services {
			localPorts[s.Name] = s.Port
		}
	}

	var drifts []Drift
	for name, want := range refPorts {
		got, ok := localPorts[name]
		switch {
		case !ok:
			drifts = append(drifts, Drift{Kind: DriftMissing, Service: name, Expected: want})
		case got != want:
			drifts = append(drifts, Drift{
				Kind: DriftPortMismatch, Service: name, Expected: want, Actual: got,
			})
		}
	}
	for name, got := range localPorts {
		if _, ok := refPorts[name]; !ok {
			drifts = append(drifts, Drift{Kind: DriftUnexpected, Service: name, Actual: got})
		}
	}
	return drifts
}

// NewSnapshot stamps a snapshot with the current version and time.
func NewSnapshot(device string, at time.Time) Snapshot {
	return Snapshot{Version: SnapshotVersion, CapturedAt: at, Device: device}
}
