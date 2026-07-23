package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/SinghAman21/openport/internal/model"
	"github.com/SinghAman21/openport/internal/tui/components"
	"github.com/SinghAman21/openport/internal/tui/msg"
)

// TestSectionArrowsCycleSections pins the keyboard-first navigation model: ←/→
// (and h/l) walk the top-level sections and wrap at both ends, so nothing is
// reachable only by the number keys.
func TestSectionArrowsCycleSections(t *testing.T) {
	tests := []struct {
		name string
		key  tea.KeyMsg
		from msg.Route
		want msg.Route
	}{
		{"right advances", tea.KeyMsg{Type: tea.KeyRight}, msg.RoutePorts, msg.RouteActions},
		{"l advances", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")}, msg.RoutePorts, msg.RouteActions},
		{"left wraps to last", tea.KeyMsg{Type: tea.KeyLeft}, msg.RoutePorts, msg.RouteWorkspace},
		{"h retreats", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")}, msg.RouteActions, msg.RoutePorts},
		{"right wraps to first", tea.KeyMsg{Type: tea.KeyRight}, msg.RouteWorkspace, msg.RoutePorts},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t, nil)
			// Reach the starting section, then assert the arrow moves from it.
			m, _ := h.model.navigate(tt.from)
			h.model = m.(Model)
			if h.model.route != tt.from {
				t.Fatalf("setup: could not reach %v", tt.from)
			}
			h.send(tt.key)
			if h.model.route != tt.want {
				t.Errorf("%s: from %v pressed key → %v, want %v", tt.name, tt.from, h.model.route, tt.want)
			}
		})
	}
}

// TestStepRouteWraps unit-tests the cycle arithmetic directly.
func TestStepRouteWraps(t *testing.T) {
	if got := stepRoute(msg.RoutePorts, -1); got != msg.RouteWorkspace {
		t.Errorf("left from first should wrap to last, got %v", got)
	}
	if got := stepRoute(msg.RouteWorkspace, +1); got != msg.RoutePorts {
		t.Errorf("right from last should wrap to first, got %v", got)
	}
}

// TestBlueprintEditOpensPrefilledForm covers the gap the user hit: blueprints
// could be created and deleted but never edited.
func TestBlueprintEditOpensPrefilledForm(t *testing.T) {
	h := newHarness(t, nil)
	h.sendKey("4") // blueprints
	h.sendKey("e")

	if !h.hasModal() {
		t.Fatal("e on a blueprint should open the edit form")
	}
	form, ok := h.model.modal.Content().(*components.Form)
	if !ok {
		t.Fatalf("expected a form, got %T", h.model.modal.Content())
	}
	if name := form.Values()["name"]; name != "web" {
		t.Errorf("edit form should be prefilled with the selected blueprint, got name=%q", name)
	}
	if svcs := form.Values()["services"]; !strings.Contains(svcs, "api:3000") {
		t.Errorf("edit form should carry the existing services, got %q", svcs)
	}
}

// TestProjectConfigBlueprintNoneClears checks the "(none)" dropdown choice maps
// back to an empty blueprint rather than a project pointing at a literal
// blueprint named "(none)".
func TestProjectConfigBlueprintNoneClears(t *testing.T) {
	h := newHarness(t, nil)
	cfg, err := h.model.projectConfigFromForm(map[string]string{"name": "proj", "blueprint": blueprintNoneOption})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Blueprint != "" {
		t.Errorf("(none) should clear the blueprint, got %q", cfg.Blueprint)
	}
	kept, _ := h.model.projectConfigFromForm(map[string]string{"name": "proj", "blueprint": "web"})
	if kept.Blueprint != "web" {
		t.Errorf("a real blueprint choice should be kept, got %q", kept.Blueprint)
	}
}

// TestPortsFilterKeepsNavigable checks that while the search box is open, arrow
// keys still move the selection (the user's complaint) while printable keys
// still narrow the query.
func TestPortsFilterKeepsNavigable(t *testing.T) {
	h := newHarness(t, nil)
	h.sendKey("/") // open the filter
	if !h.model.portsView.CapturingInput() {
		t.Fatal("/ should open the filter")
	}

	// Sample ports sort to 22, 3000, 5432; selection starts on the first.
	before, _ := h.model.portsView.Selection()
	h.send(tea.KeyMsg{Type: tea.KeyDown})
	after, ok := h.model.portsView.Selection()
	if !ok || after.Port == before.Port {
		t.Errorf("down while filtering should move the selection: before=%d after=%d", before.Port, after.Port)
	}
	if !h.model.portsView.CapturingInput() {
		t.Error("navigating should not exit the search box")
	}

	// A printable key still narrows rather than navigating.
	h.sendKey("5432")
	if sel, ok := h.model.portsView.Selection(); !ok || sel.Port != 5432 {
		t.Errorf("typing should filter the table to 5432, got %+v", sel)
	}
}

// TestSubTabsCycle checks Tab wraps around the sub-tabs rather than dead-ending.
func TestSubTabsCycle(t *testing.T) {
	tab := tea.KeyMsg{Type: tea.KeyTab}

	t.Run("aliases", func(t *testing.T) {
		h := newHarness(t, nil)
		h.sendKey("3")
		if got := h.model.aliasesView.Title(); got != "Aliases" {
			t.Fatalf("start on Aliases, got %s", got)
		}
		h.send(tab)
		if got := h.model.aliasesView.Title(); got != "Groups" {
			t.Errorf("tab → Groups, got %s", got)
		}
		h.send(tab)
		if got := h.model.aliasesView.Title(); got != "Aliases" {
			t.Errorf("tab should wrap back to Aliases, got %s", got)
		}
	})

	t.Run("history", func(t *testing.T) {
		h := newHarness(t, nil)
		h.sendKey("6")
		if got := h.model.historyView.Title(); got != "History" {
			t.Fatalf("start on History, got %s", got)
		}
		h.send(tab)
		if got := h.model.historyView.Title(); got != "Logs" {
			t.Errorf("tab → Logs, got %s", got)
		}
		h.send(tab)
		if got := h.model.historyView.Title(); got != "History" {
			t.Errorf("tab should wrap back to History, got %s", got)
		}
	})
}

// TestHistorySearchFilters checks the new History filter narrows the audit trail
// and hides what does not match.
func TestHistorySearchFilters(t *testing.T) {
	at := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	entries := []model.HistoryEntry{
		{At: at, Kind: model.ActionKind("kill"), Summary: "killed vite on port 5173", Success: true},
		{At: at, Kind: model.ActionKind("reserve"), Summary: "reserved 4000 for api", Success: true},
	}

	h := newHarness(t, nil)
	h.store.history = entries
	h.sendKey("6") // history
	h.send(msg.DataLoadedMsg{History: entries})

	frame := ansi.Strip(h.model.View())
	if !strings.Contains(frame, "vite") || !strings.Contains(frame, "reserved 4000") {
		t.Fatalf("both entries should show before filtering:\n%s", frame)
	}

	h.sendKey("/")
	h.sendKey("vite")

	frame = ansi.Strip(h.model.View())
	if !strings.Contains(frame, "vite") {
		t.Errorf("the matching entry should remain:\n%s", frame)
	}
	if strings.Contains(frame, "reserved 4000") {
		t.Errorf("the non-matching entry should be filtered out:\n%s", frame)
	}
}
