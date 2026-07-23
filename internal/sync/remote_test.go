package sync

// These tests exercise the Pro HTTP client against an httptest server bound to
// loopback. They never reach a real endpoint or the OS — the CLAUDE.md rule
// that "tests never touch the real network" is about external calls, and a
// 127.0.0.1 test server is the standard, offline way to test an HTTP client.

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SinghAman21/openport/internal/model"
)

// newTestProvider returns a provider pointed at srv, already "logged in" so the
// workspace-scoped calls can run, with workspace pre-selected when id != "".
func newTestProvider(srv *httptest.Server, workspace string) *RemoteProvider {
	return &RemoteProvider{
		License:   "test-license",
		Endpoint:  srv.URL,
		hc:        srv.Client(),
		workspace: workspace,
	}
}

func TestLoginExchangesLicenseForSession(t *testing.T) {
	var gotAuth, gotPath string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		gotAuth = req.Header.Get("Authorization")
		gotPath = req.URL.Path
		_ = json.NewDecoder(req.Body).Decode(&gotBody)
		writeJSON(t, w, http.StatusOK, Session{Token: "sess-tok", Email: "dev@example.com"})
	}))
	defer srv.Close()

	p := &RemoteProvider{License: "test-license", Endpoint: srv.URL, hc: srv.Client()}
	sess, err := p.Login(context.Background(), OAuthGitHub)
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if sess.Email != "dev@example.com" || sess.Token != "sess-tok" {
		t.Fatalf("unexpected session: %+v", sess)
	}
	if gotPath != "/oauth/token" {
		t.Fatalf("path = %q, want /oauth/token", gotPath)
	}
	// Before login the license is the bearer credential.
	if gotAuth != "Bearer test-license" {
		t.Fatalf("auth = %q, want Bearer test-license", gotAuth)
	}
	if gotBody["provider"] != "github" {
		t.Fatalf("provider in body = %v, want github", gotBody["provider"])
	}
	// After login the session token becomes the credential.
	if p.authToken() != "sess-tok" {
		t.Fatalf("authToken = %q, want sess-tok", p.authToken())
	}
}

func TestWorkspacesDecodesList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/v1/workspaces" {
			t.Errorf("path = %q", req.URL.Path)
		}
		writeJSON(t, w, http.StatusOK, []Workspace{
			{ID: "ws1", Name: "Acme", Role: "owner", Members: 3},
			{ID: "ws2", Name: "Beta"},
		})
	}))
	defer srv.Close()

	got, err := newTestProvider(srv, "").Workspaces(context.Background())
	if err != nil {
		t.Fatalf("Workspaces: %v", err)
	}
	if len(got) != 2 || got[0].ID != "ws1" || got[1].Name != "Beta" {
		t.Fatalf("unexpected workspaces: %+v", got)
	}
}

func TestSelectWorkspaceValidatesThenRecords(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/v1/workspaces/ws1" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		writeJSON(t, w, http.StatusOK, Workspace{ID: "ws1", Name: "Acme"})
	}))
	defer srv.Close()

	p := newTestProvider(srv, "")
	if err := p.SelectWorkspace(context.Background(), "ws1"); err != nil {
		t.Fatalf("SelectWorkspace: %v", err)
	}
	if p.Workspace() != "ws1" {
		t.Fatalf("workspace = %q, want ws1", p.Workspace())
	}

	// A rejected workspace must not be recorded.
	if err := p.SelectWorkspace(context.Background(), "missing"); err == nil {
		t.Fatal("expected error selecting unknown workspace")
	}
	if p.Workspace() != "ws1" {
		t.Fatalf("workspace changed to %q on failed select", p.Workspace())
	}
}

func TestSelectWorkspaceEmptyID(t *testing.T) {
	p := &RemoteProvider{License: "x"}
	if err := p.SelectWorkspace(context.Background(), "  "); !errors.Is(err, errNoWorkspace) {
		t.Fatalf("err = %v, want errNoWorkspace", err)
	}
}

func TestPushSendsSnapshot(t *testing.T) {
	var got Snapshot
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPut || req.URL.Path != "/v1/workspaces/ws1/snapshot" {
			t.Errorf("method/path = %s %s", req.Method, req.URL.Path)
		}
		_ = json.NewDecoder(req.Body).Decode(&got)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	snap := Snapshot{Version: SnapshotVersion, Device: "laptop",
		Aliases: []model.Alias{{Name: "web", Port: 3000}}}
	if err := newTestProvider(srv, "ws1").Push(context.Background(), snap); err != nil {
		t.Fatalf("Push: %v", err)
	}
	if got.Device != "laptop" || len(got.Aliases) != 1 || got.Aliases[0].Port != 3000 {
		t.Fatalf("server received wrong snapshot: %+v", got)
	}
}

func TestPullDecodesSnapshot(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		writeJSON(t, w, http.StatusOK, Snapshot{
			Version: SnapshotVersion,
			Groups:  []model.Group{{Name: "dev", Ports: []int{3000, 5432}}},
		})
	}))
	defer srv.Close()

	snap, err := newTestProvider(srv, "ws1").Pull(context.Background())
	if err != nil {
		t.Fatalf("Pull: %v", err)
	}
	if len(snap.Groups) != 1 || snap.Groups[0].Name != "dev" {
		t.Fatalf("unexpected snapshot: %+v", snap)
	}
}

func TestRegistryDecodesMembers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/v1/workspaces/ws1/registry" {
			t.Errorf("path = %q", req.URL.Path)
		}
		writeJSON(t, w, http.StatusOK, []Member{{
			Name:     "Ada",
			Services: []MemberService{{Name: "api", Port: 8080, Status: "up"}},
		}})
	}))
	defer srv.Close()

	members, err := newTestProvider(srv, "ws1").Registry(context.Background())
	if err != nil {
		t.Fatalf("Registry: %v", err)
	}
	if len(members) != 1 || members[0].Services[0].Port != 8080 {
		t.Fatalf("unexpected registry: %+v", members)
	}
}

func TestDetectDriftPostsSnapshotAndDecodesDrifts(t *testing.T) {
	var got Snapshot
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost || req.URL.Path != "/v1/workspaces/ws1/drift" {
			t.Errorf("method/path = %s %s", req.Method, req.URL.Path)
		}
		_ = json.NewDecoder(req.Body).Decode(&got)
		writeJSON(t, w, http.StatusOK, []Drift{
			{Kind: DriftPortMismatch, Service: "api", Expected: 8080, Actual: 9090},
		})
	}))
	defer srv.Close()

	local := Snapshot{Blueprints: []model.Blueprint{{
		Name: "web", Services: []model.BlueprintService{{Name: "api", Port: 9090}},
	}}}
	drifts, err := newTestProvider(srv, "ws1").DetectDrift(context.Background(), local)
	if err != nil {
		t.Fatalf("DetectDrift: %v", err)
	}
	if len(got.Blueprints) != 1 {
		t.Fatalf("server did not receive local snapshot: %+v", got)
	}
	if len(drifts) != 1 || drifts[0].Kind != DriftPortMismatch || drifts[0].Actual != 9090 {
		t.Fatalf("unexpected drifts: %+v", drifts)
	}
}

func TestWorkspaceScopedCallsNeedAWorkspace(t *testing.T) {
	p := &RemoteProvider{License: "x", hc: http.DefaultClient}
	if err := p.Push(context.Background(), Snapshot{}); !errors.Is(err, errNoWorkspace) {
		t.Errorf("Push err = %v, want errNoWorkspace", err)
	}
	if _, err := p.Pull(context.Background()); !errors.Is(err, errNoWorkspace) {
		t.Errorf("Pull err = %v, want errNoWorkspace", err)
	}
	if _, err := p.Registry(context.Background()); !errors.Is(err, errNoWorkspace) {
		t.Errorf("Registry err = %v, want errNoWorkspace", err)
	}
	if _, err := p.DetectDrift(context.Background(), Snapshot{}); !errors.Is(err, errNoWorkspace) {
		t.Errorf("DetectDrift err = %v, want errNoWorkspace", err)
	}
}

func TestAPIErrorSurfacesServerMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		writeJSON(t, w, http.StatusForbidden, APIError{Code: "forbidden", Message: "not a member"})
	}))
	defer srv.Close()

	_, err := newTestProvider(srv, "").Workspaces(context.Background())
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %T %v, want *APIError", err, err)
	}
	if apiErr.Status != http.StatusForbidden || apiErr.Message != "not a member" {
		t.Fatalf("unexpected APIError: %+v", apiErr)
	}
}

// A blank license must never open a socket: it fails closed with ErrProRequired
// before any request is built. The unreachable endpoint proves no call is made.
func TestNoLicenseNeverCallsOut(t *testing.T) {
	p := &RemoteProvider{License: "", Endpoint: "http://127.0.0.1:0", hc: http.DefaultClient}
	if _, err := p.Workspaces(context.Background()); !errors.Is(err, ErrProRequired) {
		t.Fatalf("err = %v, want ErrProRequired", err)
	}
	if _, err := p.Login(context.Background(), OAuthGitHub); !errors.Is(err, ErrProRequired) {
		t.Fatalf("Login err = %v, want ErrProRequired", err)
	}
}

func TestDetectHonorsEndpointOverride(t *testing.T) {
	t.Setenv("OPENPORT_ENDPOINT", "https://self-hosted.example.com/")
	p, ok := Detect("lic").(*RemoteProvider)
	if !ok {
		t.Fatal("Detect did not return a RemoteProvider for a license")
	}
	if p.baseURL() != "https://self-hosted.example.com" {
		t.Fatalf("baseURL = %q, want trimmed override", p.baseURL())
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, status int, v any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Errorf("encode: %v", err)
	}
}
