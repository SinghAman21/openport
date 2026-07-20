package views

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/SinghAman21/openport/internal/model"
	"github.com/SinghAman21/openport/internal/tui/actions"
	"github.com/SinghAman21/openport/internal/tui/msg"
	"github.com/SinghAman21/openport/internal/tui/styles"
)

// configTab selects which pane of the config screen is showing.
type configTab int

const (
	tabProject configTab = iota
	tabProtected
)

// ConfigView shows the per-project .openport.yml and the never-kill list, and
// lets you edit both without leaving the tool.
type ConfigView struct {
	theme styles.Theme
	tab   configTab

	project    model.ProjectConfig
	hasProject bool
	protected  model.ProtectedPorts

	vp    viewport.Model
	ready bool
}

// NewConfigView builds the config screen.
func NewConfigView(theme styles.Theme) *ConfigView {
	return &ConfigView{theme: theme, vp: viewport.New(0, 0)}
}

// Title implements View.
func (v *ConfigView) Title() string {
	if v.tab == tabProtected {
		return "Protected ports"
	}
	return "Project config"
}

var configKeys = struct {
	edit, initCfg, protect, nextTab, prevTab key.Binding
}{
	edit:    key.NewBinding(key.WithKeys("e", "enter"), key.WithHelp("e", "edit")),
	initCfg: key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "create config")),
	protect: key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "protect a port")),
	nextTab: key.NewBinding(key.WithKeys("tab", "right", "l"), key.WithHelp("tab", "next tab")),
	prevTab: key.NewBinding(key.WithKeys("shift+tab", "left", "h"), key.WithHelp("shift+tab", "prev tab")),
}

// Help implements View.
func (v *ConfigView) Help() []key.Binding {
	if v.tab == tabProtected {
		return []key.Binding{configKeys.protect, configKeys.nextTab}
	}
	return []key.Binding{configKeys.edit, configKeys.initCfg, configKeys.nextTab}
}

// SelectionSummary feeds the status bar.
func (v *ConfigView) SelectionSummary() string {
	if v.tab == tabProtected {
		return strconv.Itoa(len(v.protected.Ports)) + " protected"
	}
	if v.hasProject {
		return v.project.Name
	}
	return "no .openport.yml"
}

// ShowProtected switches to the never-kill list.
func (v *ConfigView) ShowProtected() { v.tab = tabProtected; v.render() }

// ShowProject switches to the project config.
func (v *ConfigView) ShowProject() { v.tab = tabProject; v.render() }

// Project exposes the loaded config so the root model can prefill the editor.
func (v *ConfigView) Project() (model.ProjectConfig, bool) { return v.project, v.hasProject }

// Update implements View.
func (v *ConfigView) Update(message tea.Msg) (View, tea.Cmd) {
	switch m := message.(type) {
	case msg.DataLoadedMsg:
		if m.Err == nil {
			v.project, v.hasProject, v.protected = m.Project, m.HasProject, m.Protected
			v.render()
		}
		return v, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(m, configKeys.nextTab):
			v.tab = tabProtected
			v.render()
			return v, nil
		case key.Matches(m, configKeys.prevTab):
			v.tab = tabProject
			v.render()
			return v, nil
		case key.Matches(m, configKeys.protect):
			if v.tab == tabProtected {
				return v, emit(msg.RunActionMsg{ID: actions.ProtectedToggle})
			}
		case key.Matches(m, configKeys.initCfg):
			if v.tab == tabProject && !v.hasProject {
				return v, emit(msg.RunActionMsg{ID: actions.ConfigInit})
			}
		case key.Matches(m, configKeys.edit):
			if v.tab == tabProject {
				return v, emit(msg.RunActionMsg{ID: actions.ConfigView})
			}
		}
		vp, cmd := v.vp.Update(m)
		v.vp = vp
		return v, cmd
	}
	return v, nil
}

// render rebuilds the viewport contents for the active tab.
func (v *ConfigView) render() {
	if v.tab == tabProtected {
		v.vp.SetContent(v.protectedBody())
		return
	}
	v.vp.SetContent(v.projectBody())
	v.vp.GotoTop()
}

func (v *ConfigView) projectBody() string {
	if !v.hasProject {
		return v.theme.Faint.Render(
			"No .openport.yml in this directory or any parent.\n\n"+
				"A project config records the ports this project expects, the names you\n"+
				"give them, and any ranges it owns — so OpenPort knows what \"normal\"\n"+
				"looks like here.\n\n") +
			v.theme.Accent.Render("Press i to create one.")
	}

	var b strings.Builder
	b.WriteString(v.theme.PanelTitle.Render(orDash(v.project.Name)) + "\n")
	b.WriteString(v.theme.Faint.Render(v.project.Dir+"/"+model.ProjectConfigFile) + "\n\n")

	if len(v.project.ExpectedPorts) > 0 {
		b.WriteString(v.theme.Bold.Render("Expected ports") + "\n")
		for _, p := range v.project.ExpectedPorts {
			b.WriteString("  " + v.theme.Body.Render(strconv.Itoa(p)) + "\n")
		}
		b.WriteString("\n")
	}

	if len(v.project.Aliases) > 0 {
		b.WriteString(v.theme.Bold.Render("Aliases") + "\n")
		names := make([]string, 0, len(v.project.Aliases))
		for name := range v.project.Aliases {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			b.WriteString(fmt.Sprintf("  %s %s %d\n",
				v.theme.Body.Render(padRight(name, 16)),
				v.theme.Faint.Render("→"),
				v.project.Aliases[name]))
		}
		b.WriteString("\n")
	}

	if len(v.project.Ranges) > 0 {
		b.WriteString(v.theme.Bold.Render("Ranges") + "\n")
		for _, r := range v.project.Ranges {
			label := r.Name
			if label == "" {
				label = "unnamed"
			}
			b.WriteString(fmt.Sprintf("  %s %d–%d\n",
				v.theme.Body.Render(padRight(label, 16)), r.From, r.To))
		}
		b.WriteString("\n")
	}

	if v.project.Blueprint != "" {
		b.WriteString(v.theme.Bold.Render("Blueprint") + "\n")
		b.WriteString("  " + v.theme.Body.Render(v.project.Blueprint) + "\n\n")
	}

	b.WriteString(v.theme.Faint.Render("Press e to edit."))
	return b.String()
}

func (v *ConfigView) protectedBody() string {
	var b strings.Builder
	b.WriteString(v.theme.PanelTitle.Render("Never-kill list") + "\n")
	b.WriteString(v.theme.Faint.Render(
		"OpenPort refuses to signal anything on this list. Killing one anyway\n"+
			"needs an explicit override and a second confirmation.") + "\n\n")

	if len(v.protected.Ports) == 0 && len(v.protected.Ranges) == 0 &&
		len(v.protected.Processes) == 0 {
		b.WriteString(v.theme.Warning.Render("Nothing is protected. That is unusual — " +
			"consider protecting ssh (22) at least."))
		return b.String()
	}

	if len(v.protected.Ports) > 0 {
		b.WriteString(v.theme.Bold.Render("Ports") + "\n")
		for _, p := range v.protected.Ports {
			b.WriteString("  " + v.theme.BadgeProtected.Render(styles.GlyphProtected) +
				" " + v.theme.Body.Render(strconv.Itoa(p)) + "\n")
		}
		b.WriteString("\n")
	}

	if len(v.protected.Ranges) > 0 {
		b.WriteString(v.theme.Bold.Render("Ranges") + "\n")
		for _, r := range v.protected.Ranges {
			label := r.Name
			if label == "" {
				label = "unnamed"
			}
			b.WriteString(fmt.Sprintf("  %s %d–%d\n",
				v.theme.Body.Render(padRight(label, 16)), r.From, r.To))
		}
		b.WriteString("\n")
	}

	if len(v.protected.Processes) > 0 {
		b.WriteString(v.theme.Bold.Render("Processes") + "\n")
		for _, p := range v.protected.Processes {
			b.WriteString("  " + v.theme.Body.Render(p) + "\n")
		}
		b.WriteString("\n")
	}

	b.WriteString(v.theme.Faint.Render("Press p to protect or unprotect a port."))
	return b.String()
}

// View implements View.
func (v *ConfigView) View(width, height int) string {
	tabs := tabBar(int(v.tab), []string{"Project", "Protected"}, v.theme.TabActive, v.theme.Tab)

	bodyHeight := height - 3
	if bodyHeight < 3 {
		bodyHeight = 3
	}
	v.vp.Width, v.vp.Height = width, bodyHeight
	if !v.ready {
		v.render()
		v.ready = true
	}
	return tabs + "\n\n" + v.vp.View()
}
