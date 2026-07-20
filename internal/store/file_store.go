package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/SinghAman21/openport/internal/model"
	opsync "github.com/SinghAman21/openport/internal/sync"
)

// Retention caps. History and logs are append-only ring buffers on disk;
// without a cap they grow without bound on a long-lived install.
const (
	MaxHistory = 500
	MaxLogs    = 1000
)

// On-disk filenames under the config dir.
const (
	fileAliases    = "aliases.json"
	fileGroups     = "groups.json"
	fileBlueprints = "blueprints.json"
	fileProtected  = "protected.json"
	fileHistory    = "history.json"
	fileLogs       = "logs.json"
)

// FileStore persists state as JSON under the user's config directory. Writes
// are atomic (temp file + rename) so a crash mid-save cannot leave a truncated
// file behind, and every access is mutex-guarded because Bubble Tea runs
// commands on their own goroutines.
type FileStore struct {
	dir string
	mu  sync.RWMutex
	now func() time.Time
}

var _ Store = (*FileStore)(nil)

// ConfigDirEnv overrides where state is stored. `make dev` points it at a
// throwaway directory inside the repo so working on OpenPort cannot disturb
// the aliases and protected ports you actually rely on. It is an explicit
// variable rather than XDG_CONFIG_HOME because that one has no effect on
// macOS, where Go resolves config to ~/Library/Application Support.
const ConfigDirEnv = "OPENPORT_CONFIG_DIR"

// NewFileStore opens the store in the platform's standard config location:
// $XDG_CONFIG_HOME/openport on Linux, ~/Library/Application Support/openport
// on macOS — unless OPENPORT_CONFIG_DIR says otherwise.
func NewFileStore() (Store, error) {
	if dir := strings.TrimSpace(os.Getenv(ConfigDirEnv)); dir != "" {
		return NewFileStoreAt(dir)
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("locate config dir: %w", err)
	}
	return NewFileStoreAt(filepath.Join(base, "openport"))
}

// NewFileStoreAt opens the store at an explicit directory. Tests use this with
// t.TempDir() so they never write to the real config location.
func NewFileStoreAt(dir string) (Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create store dir %s: %w", dir, err)
	}
	return &FileStore{dir: dir, now: time.Now}, nil
}

// Dir reports where the store lives, for display in the config view.
func (s *FileStore) Dir() string { return s.dir }

func (s *FileStore) path(name string) string { return filepath.Join(s.dir, name) }

// readJSON decodes a store file. A missing file is not an error — it means
// "nothing saved yet" and yields the zero value.
func readJSON[T any](s *FileStore, name string, out *T) error {
	b, err := os.ReadFile(s.path(name))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read %s: %w", name, err)
	}
	if len(b) == 0 {
		return nil
	}
	if err := json.Unmarshal(b, out); err != nil {
		return fmt.Errorf("parse %s: %w", name, err)
	}
	return nil
}

// writeJSON atomically replaces a store file.
func writeJSON(s *FileStore, name string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", name, err)
	}
	b = append(b, '\n')

	tmp, err := os.CreateTemp(s.dir, "."+name+".*")
	if err != nil {
		return fmt.Errorf("stage %s: %w", name, err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op once the rename succeeds

	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return fmt.Errorf("write %s: %w", name, err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("sync %s: %w", name, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close %s: %w", name, err)
	}
	if err := os.Rename(tmpName, s.path(name)); err != nil {
		return fmt.Errorf("commit %s: %w", name, err)
	}
	return nil
}

// --- aliases -------------------------------------------------------------

// Aliases returns every alias, sorted by port.
func (s *FileStore) Aliases() ([]model.Alias, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.aliasesLocked()
}

func (s *FileStore) aliasesLocked() ([]model.Alias, error) {
	var out []model.Alias
	if err := readJSON(s, fileAliases, &out); err != nil {
		return nil, err
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Port < out[j].Port })
	return out, nil
}

// SaveAlias inserts or updates an alias by name.
func (s *FileStore) SaveAlias(a model.Alias) error {
	if err := a.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	list, err := s.aliasesLocked()
	if err != nil {
		return err
	}
	if a.CreatedAt.IsZero() {
		a.CreatedAt = s.now()
	}
	replaced := false
	for i := range list {
		if strings.EqualFold(list[i].Name, a.Name) {
			a.CreatedAt = list[i].CreatedAt // preserve original creation time
			list[i] = a
			replaced = true
			break
		}
	}
	if !replaced {
		list = append(list, a)
	}
	return writeJSON(s, fileAliases, list)
}

// DeleteAlias removes an alias by name. Deleting a missing alias is an error
// so the UI can tell the user nothing happened.
func (s *FileStore) DeleteAlias(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	list, err := s.aliasesLocked()
	if err != nil {
		return err
	}
	out := make([]model.Alias, 0, len(list))
	found := false
	for _, a := range list {
		if strings.EqualFold(a.Name, name) {
			found = true
			continue
		}
		out = append(out, a)
	}
	if !found {
		return fmt.Errorf("no alias named %q", name)
	}
	return writeJSON(s, fileAliases, out)
}

// --- groups --------------------------------------------------------------

// Groups returns every group, sorted by name.
func (s *FileStore) Groups() ([]model.Group, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.groupsLocked()
}

func (s *FileStore) groupsLocked() ([]model.Group, error) {
	var out []model.Group
	if err := readJSON(s, fileGroups, &out); err != nil {
		return nil, err
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// SaveGroup inserts or updates a group by name.
func (s *FileStore) SaveGroup(g model.Group) error {
	if err := g.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	list, err := s.groupsLocked()
	if err != nil {
		return err
	}
	if g.CreatedAt.IsZero() {
		g.CreatedAt = s.now()
	}
	sort.Ints(g.Ports)
	replaced := false
	for i := range list {
		if strings.EqualFold(list[i].Name, g.Name) {
			g.CreatedAt = list[i].CreatedAt
			list[i] = g
			replaced = true
			break
		}
	}
	if !replaced {
		list = append(list, g)
	}
	return writeJSON(s, fileGroups, list)
}

// DeleteGroup removes a group by name.
func (s *FileStore) DeleteGroup(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	list, err := s.groupsLocked()
	if err != nil {
		return err
	}
	out := make([]model.Group, 0, len(list))
	found := false
	for _, g := range list {
		if strings.EqualFold(g.Name, name) {
			found = true
			continue
		}
		out = append(out, g)
	}
	if !found {
		return fmt.Errorf("no group named %q", name)
	}
	return writeJSON(s, fileGroups, out)
}

// --- blueprints ----------------------------------------------------------

// Blueprints returns every blueprint, sorted by name.
func (s *FileStore) Blueprints() ([]model.Blueprint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.blueprintsLocked()
}

func (s *FileStore) blueprintsLocked() ([]model.Blueprint, error) {
	var out []model.Blueprint
	if err := readJSON(s, fileBlueprints, &out); err != nil {
		return nil, err
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// SaveBlueprint inserts or updates a blueprint by name.
func (s *FileStore) SaveBlueprint(b model.Blueprint) error {
	if err := b.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	list, err := s.blueprintsLocked()
	if err != nil {
		return err
	}
	now := s.now()
	b.UpdatedAt = now
	if b.CreatedAt.IsZero() {
		b.CreatedAt = now
	}
	replaced := false
	for i := range list {
		if strings.EqualFold(list[i].Name, b.Name) {
			b.CreatedAt = list[i].CreatedAt
			list[i] = b
			replaced = true
			break
		}
	}
	if !replaced {
		list = append(list, b)
	}
	return writeJSON(s, fileBlueprints, list)
}

// DeleteBlueprint removes a blueprint by name.
func (s *FileStore) DeleteBlueprint(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	list, err := s.blueprintsLocked()
	if err != nil {
		return err
	}
	out := make([]model.Blueprint, 0, len(list))
	found := false
	for _, b := range list {
		if strings.EqualFold(b.Name, name) {
			found = true
			continue
		}
		out = append(out, b)
	}
	if !found {
		return fmt.Errorf("no blueprint named %q", name)
	}
	return writeJSON(s, fileBlueprints, out)
}

// --- protected ports -----------------------------------------------------

// ProtectedPorts returns the never-kill list, seeding a sensible default on
// first run so a fresh install cannot immediately kill sshd.
func (s *FileStore) ProtectedPorts() (model.ProtectedPorts, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, err := os.Stat(s.path(fileProtected)); errors.Is(err, os.ErrNotExist) {
		return model.DefaultProtectedPorts(), nil
	}
	var out model.ProtectedPorts
	if err := readJSON(s, fileProtected, &out); err != nil {
		return model.DefaultProtectedPorts(), err
	}
	return out, nil
}

// SaveProtectedPorts replaces the never-kill list.
func (s *FileStore) SaveProtectedPorts(p model.ProtectedPorts) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sort.Ints(p.Ports)
	return writeJSON(s, fileProtected, p)
}

// --- project config ------------------------------------------------------

// ProjectConfig loads .openport.yml, searching dir and then each parent up to
// the filesystem root — so running openport from a subdirectory of a project
// still finds the project's config.
func (s *FileStore) ProjectConfig(dir string) (model.ProjectConfig, bool, error) {
	if dir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return model.ProjectConfig{}, false, err
		}
		dir = wd
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return model.ProjectConfig{}, false, err
	}

	for {
		candidate := filepath.Join(abs, model.ProjectConfigFile)
		b, err := os.ReadFile(candidate)
		switch {
		case err == nil:
			var cfg model.ProjectConfig
			if err := yaml.Unmarshal(b, &cfg); err != nil {
				return model.ProjectConfig{}, true, fmt.Errorf("parse %s: %w", candidate, err)
			}
			cfg.Dir = abs
			return cfg, true, nil
		case !errors.Is(err, os.ErrNotExist):
			return model.ProjectConfig{}, false, fmt.Errorf("read %s: %w", candidate, err)
		}

		parent := filepath.Dir(abs)
		if parent == abs { // reached the root
			return model.ProjectConfig{}, false, nil
		}
		abs = parent
	}
}

// SaveProjectConfig writes .openport.yml into dir.
func (s *FileStore) SaveProjectConfig(dir string, cfg model.ProjectConfig) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	if dir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		dir = wd
	}
	b, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("encode project config: %w", err)
	}
	header := "# OpenPort project config — https://github.com/SinghAman21/openport\n"
	path := filepath.Join(dir, model.ProjectConfigFile)
	if err := os.WriteFile(path, append([]byte(header), b...), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// --- history & logs ------------------------------------------------------

// AppendHistory records a mutating action, trimming to MaxHistory entries.
func (s *FileStore) AppendHistory(e model.HistoryEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var list []model.HistoryEntry
	if err := readJSON(s, fileHistory, &list); err != nil {
		return err
	}
	if e.At.IsZero() {
		e.At = s.now()
	}
	list = append(list, e)
	if len(list) > MaxHistory {
		list = list[len(list)-MaxHistory:]
	}
	return writeJSON(s, fileHistory, list)
}

// History returns the most recent entries, newest first. limit <= 0 means all.
func (s *FileStore) History(limit int) ([]model.HistoryEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var list []model.HistoryEntry
	if err := readJSON(s, fileHistory, &list); err != nil {
		return nil, err
	}
	return reverseTail(list, limit), nil
}

// AppendLog records a diagnostic line, trimming to MaxLogs entries.
func (s *FileStore) AppendLog(e model.LogEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var list []model.LogEntry
	if err := readJSON(s, fileLogs, &list); err != nil {
		return err
	}
	if e.At.IsZero() {
		e.At = s.now()
	}
	list = append(list, e)
	if len(list) > MaxLogs {
		list = list[len(list)-MaxLogs:]
	}
	return writeJSON(s, fileLogs, list)
}

// Logs returns the most recent log lines, newest first. limit <= 0 means all.
func (s *FileStore) Logs(limit int) ([]model.LogEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var list []model.LogEntry
	if err := readJSON(s, fileLogs, &list); err != nil {
		return nil, err
	}
	return reverseTail(list, limit), nil
}

// reverseTail returns the last limit elements, newest first.
func reverseTail[T any](in []T, limit int) []T {
	if limit > 0 && len(in) > limit {
		in = in[len(in)-limit:]
	}
	out := make([]T, len(in))
	for i, v := range in {
		out[len(in)-1-i] = v
	}
	return out
}

// --- snapshots (Pro seam) ------------------------------------------------

// Snapshot captures everything syncable. Nothing here touches the network;
// the sync provider decides whether the result ever leaves the machine.
func (s *FileStore) Snapshot() (opsync.Snapshot, error) {
	aliases, err := s.Aliases()
	if err != nil {
		return opsync.Snapshot{}, err
	}
	groups, err := s.Groups()
	if err != nil {
		return opsync.Snapshot{}, err
	}
	blueprints, err := s.Blueprints()
	if err != nil {
		return opsync.Snapshot{}, err
	}
	protected, err := s.ProtectedPorts()
	if err != nil {
		return opsync.Snapshot{}, err
	}
	host, _ := os.Hostname()

	snap := opsync.NewSnapshot(host, s.now())
	snap.Aliases = aliases
	snap.Groups = groups
	snap.Blueprints = blueprints
	snap.Protected = protected
	return snap, nil
}

// ApplySnapshot replaces local state with a snapshot's contents.
func (s *FileStore) ApplySnapshot(snap opsync.Snapshot) error {
	if snap.Version > opsync.SnapshotVersion {
		return fmt.Errorf("snapshot version %d is newer than this build supports (%d)",
			snap.Version, opsync.SnapshotVersion)
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := writeJSON(s, fileAliases, snap.Aliases); err != nil {
		return err
	}
	if err := writeJSON(s, fileGroups, snap.Groups); err != nil {
		return err
	}
	if err := writeJSON(s, fileBlueprints, snap.Blueprints); err != nil {
		return err
	}
	return writeJSON(s, fileProtected, snap.Protected)
}
