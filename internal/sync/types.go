package sync

import (
	"time"

	"github.com/SinghAman21/openport/internal/model"
)

// SnapshotVersion is bumped when the serialized shape changes.
const SnapshotVersion = 1

// Snapshot is the serialization boundary between the local store and any
// remote workspace. store.Snapshot()/ApplySnapshot() are the only producers
// and consumers.
type Snapshot struct {
	Version    int                  `json:"version"`
	CapturedAt time.Time            `json:"captured_at"`
	Device     string               `json:"device,omitempty"`
	Aliases    []model.Alias        `json:"aliases"`
	Groups     []model.Group        `json:"groups"`
	Blueprints []model.Blueprint    `json:"blueprints"`
	Protected  model.ProtectedPorts `json:"protected"`
}

// Workspace is a team namespace a Pro user can sync against.
type Workspace struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Role    string `json:"role"` // owner, admin, member
	Members int    `json:"members"`
}

// MemberService is one service a teammate currently runs.
type MemberService struct {
	Name   string `json:"name"`
	Port   int    `json:"port"`
	Status string `json:"status"` // up, down, unknown
}

// Member is a teammate in the shared registry — who runs what, where.
type Member struct {
	ID       string          `json:"id"`
	Name     string          `json:"name"`
	Email    string          `json:"email,omitempty"`
	Device   string          `json:"device,omitempty"`
	Services []MemberService `json:"services,omitempty"`
	LastSeen time.Time       `json:"last_seen"`
}

// DriftKind classifies how live state diverges from the expected blueprint.
type DriftKind string

const (
	// DriftMissing: the blueprint expects a service that is not listening.
	DriftMissing DriftKind = "missing"
	// DriftUnexpected: something listens on a port the blueprint does not claim.
	DriftUnexpected DriftKind = "unexpected"
	// DriftPortMismatch: the service runs, but on a different port.
	DriftPortMismatch DriftKind = "port_mismatch"
	// DriftConflict: a teammate claims the same port for a different service.
	DriftConflict DriftKind = "conflict"
)

// Drift is one divergence between local reality and the expected environment.
type Drift struct {
	Kind     DriftKind `json:"kind"`
	Service  string    `json:"service"`
	Expected int       `json:"expected,omitempty"`
	Actual   int       `json:"actual,omitempty"`
	Owner    string    `json:"owner,omitempty"`
	Detail   string    `json:"detail,omitempty"`
}

// OAuthProvider is the identity provider used for Pro login.
type OAuthProvider string

const (
	OAuthGitHub OAuthProvider = "github"
	OAuthGoogle OAuthProvider = "google"
)

// Session is the result of a successful login.
type Session struct {
	Token     string    `json:"token"`
	UserID    string    `json:"user_id"`
	Email     string    `json:"email,omitempty"`
	Name      string    `json:"name,omitempty"`
	ExpiresAt time.Time `json:"expires_at"`
}
