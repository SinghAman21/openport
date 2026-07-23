package sync

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/SinghAman21/openport/internal/version"
)

// DefaultEndpoint is where a real deployment talks to. No Free code path ever
// reaches it, and it can be overridden with OPENPORT_ENDPOINT (a self-hosted
// deployment, or a test server).
const DefaultEndpoint = "https://api.openport.dev"

// maxErrorBody caps how much of an error response body we read before giving
// up, so a misbehaving endpoint cannot make us buffer without bound.
const maxErrorBody = 8 << 10

// RemoteProvider is the Pro implementation: a thin REST client for the
// OpenPort cloud. It is constructed only when a license is present (see
// Detect), so every method may assume Pro is intended — IsPro still guards the
// wire calls so a blank license fails closed rather than opening a socket.
type RemoteProvider struct {
	License   string
	Endpoint  string
	hc        *http.Client
	workspace string
	session   Session
}

var _ SyncProvider = (*RemoteProvider)(nil)

// NewRemoteProvider builds a Pro provider for a license token. The endpoint is
// DefaultEndpoint unless OPENPORT_ENDPOINT is set.
func NewRemoteProvider(license string) *RemoteProvider {
	endpoint := DefaultEndpoint
	if override := strings.TrimSpace(os.Getenv("OPENPORT_ENDPOINT")); override != "" {
		endpoint = override
	}
	return &RemoteProvider{
		License:  license,
		Endpoint: endpoint,
		hc:       &http.Client{},
	}
}

// IsPro reports true whenever a license token is present. Pro UI is gated on
// this and nothing else.
func (r *RemoteProvider) IsPro() bool { return r != nil && r.License != "" }

// Workspace returns the currently selected workspace ID, if any.
func (r *RemoteProvider) Workspace() string { return r.workspace }

// Session returns the active session, zero-valued when not logged in.
func (r *RemoteProvider) Session() Session { return r.session }

// errNoWorkspace is returned by the workspace-scoped calls before one is
// picked, so the UI can prompt for a selection rather than 404.
var errNoWorkspace = errors.New("select a workspace first")

// Login exchanges the license and chosen identity provider for a session at
// the endpoint. The license is the credential the user obtained by signing in
// on the web; this call validates it and returns the session the rest of the
// API is authorized with.
func (r *RemoteProvider) Login(ctx context.Context, provider OAuthProvider) (Session, error) {
	if !r.IsPro() {
		return Session{}, ErrProRequired
	}
	body := struct {
		Provider OAuthProvider `json:"provider"`
		License  string        `json:"license"`
	}{Provider: provider, License: r.License}

	var session Session
	if err := r.do(ctx, http.MethodPost, "/oauth/token", body, &session); err != nil {
		return Session{}, err
	}
	r.session = session
	return session, nil
}

// Workspaces lists the workspaces the session can sync against.
func (r *RemoteProvider) Workspaces(ctx context.Context) ([]Workspace, error) {
	var out []Workspace
	if err := r.do(ctx, http.MethodGet, "/v1/workspaces", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// SelectWorkspace validates membership of id against the endpoint and, on
// success, records it as the active workspace for later push/pull/registry
// calls.
func (r *RemoteProvider) SelectWorkspace(ctx context.Context, id string) error {
	if !r.IsPro() {
		return ErrProRequired
	}
	if strings.TrimSpace(id) == "" {
		return errNoWorkspace
	}
	var ws Workspace
	if err := r.do(ctx, http.MethodGet, "/v1/workspaces/"+url.PathEscape(id), nil, &ws); err != nil {
		return err
	}
	r.workspace = id
	return nil
}

// Push uploads the local snapshot to the active workspace.
func (r *RemoteProvider) Push(ctx context.Context, snap Snapshot) error {
	if r.workspace == "" {
		return errNoWorkspace
	}
	return r.do(ctx, http.MethodPut, r.workspacePath("/snapshot"), snap, nil)
}

// Pull downloads the active workspace's snapshot.
func (r *RemoteProvider) Pull(ctx context.Context) (Snapshot, error) {
	if r.workspace == "" {
		return Snapshot{}, errNoWorkspace
	}
	var snap Snapshot
	if err := r.do(ctx, http.MethodGet, r.workspacePath("/snapshot"), nil, &snap); err != nil {
		return Snapshot{}, err
	}
	return snap, nil
}

// Registry returns who on the team runs which service, and where.
func (r *RemoteProvider) Registry(ctx context.Context) ([]Member, error) {
	if r.workspace == "" {
		return nil, errNoWorkspace
	}
	var members []Member
	if err := r.do(ctx, http.MethodGet, r.workspacePath("/registry"), nil, &members); err != nil {
		return nil, err
	}
	return members, nil
}

// DetectDrift asks the endpoint to compare the local snapshot against the
// workspace blueprint and report divergence. The server owns the comparison so
// it can also flag conflicts against teammates' claimed ports, which the local
// DiffSnapshots cannot see.
func (r *RemoteProvider) DetectDrift(ctx context.Context, local Snapshot) ([]Drift, error) {
	if r.workspace == "" {
		return nil, errNoWorkspace
	}
	var drifts []Drift
	if err := r.do(ctx, http.MethodPost, r.workspacePath("/drift"), local, &drifts); err != nil {
		return nil, err
	}
	return drifts, nil
}

// workspacePath builds a path under the active workspace.
func (r *RemoteProvider) workspacePath(suffix string) string {
	return "/v1/workspaces/" + url.PathEscape(r.workspace) + suffix
}

// APIError is a non-2xx response from the endpoint, decoded from the standard
// {error, message} body when the server sends one.
type APIError struct {
	Status  int    `json:"-"`
	Code    string `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
}

func (e *APIError) Error() string {
	switch {
	case e.Message != "":
		return fmt.Sprintf("openport pro: %s (%d)", e.Message, e.Status)
	case e.Code != "":
		return fmt.Sprintf("openport pro: %s (%d)", e.Code, e.Status)
	default:
		return fmt.Sprintf("openport pro: unexpected status %d", e.Status)
	}
}

// do performs one JSON request/response round-trip. body is encoded when
// non-nil; out is decoded when non-nil. Non-2xx responses become an *APIError.
func (r *RemoteProvider) do(ctx context.Context, method, path string, body, out any) error {
	if !r.IsPro() {
		return ErrProRequired
	}

	var reader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(buf)
	}

	req, err := http.NewRequestWithContext(ctx, method, r.baseURL()+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+r.authToken())
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "openport/"+version.Version)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := r.client().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return parseAPIError(resp)
	}
	if out == nil {
		// Drain so the connection can be reused.
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("openport pro: decoding %s %s: %w", method, path, err)
	}
	return nil
}

// authToken is the bearer credential: the session token once logged in, and
// the license itself beforehand so a stored session is not required for every
// call.
func (r *RemoteProvider) authToken() string {
	if r.session.Token != "" {
		return r.session.Token
	}
	return r.License
}

func (r *RemoteProvider) baseURL() string {
	if r.Endpoint != "" {
		return strings.TrimRight(r.Endpoint, "/")
	}
	return DefaultEndpoint
}

func (r *RemoteProvider) client() *http.Client {
	if r.hc != nil {
		return r.hc
	}
	return http.DefaultClient
}

// parseAPIError turns a non-2xx response into an *APIError, decoding the
// {error, message} body when present.
func parseAPIError(resp *http.Response) error {
	apiErr := &APIError{Status: resp.StatusCode}
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, apiErr) // best-effort; keeps the bare status on failure
	}
	return apiErr
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
