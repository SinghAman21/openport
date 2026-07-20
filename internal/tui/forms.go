package tui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/SinghAman21/openport/internal/model"
	"github.com/SinghAman21/openport/internal/tui/components"
)

// Form IDs. The root dispatcher switches on these when a form completes, so
// each one names exactly one follow-up action.
const (
	formCheck          = "check"
	formKill           = "kill"
	formKillPID        = "kill_pid"
	formKillRange      = "kill_range"
	formSuggest        = "suggest"
	formReserve        = "reserve"
	formHealthTCP      = "health_tcp"
	formHealthHTTP     = "health_http"
	formAlias          = "alias"
	formGroup          = "group"
	formGroupShutdown  = "group_shutdown"
	formGroupHealth    = "group_health"
	formBlueprint      = "blueprint"
	formBlueprintApply = "blueprint_apply"
	formProjectConfig  = "project_config"
	formProtect        = "protect"
	formLogin          = "login"
	formWorkspace      = "workspace"
)

// validatePortField is the shared inline validator for port inputs, so every
// form rejects the same things with the same wording.
func validatePortField(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return fmt.Errorf("a port is required")
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return fmt.Errorf("%q is not a number", s)
	}
	return model.ValidatePort(n)
}

func validateOptionalPortField(s string) error {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return validatePortField(s)
}

func validateName(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return fmt.Errorf("a name is required")
	}
	if strings.ContainsAny(s, " \t") {
		return fmt.Errorf("names cannot contain spaces")
	}
	return nil
}

func validateCount(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return fmt.Errorf("%q is not a number", s)
	}
	if n < 1 || n > 50 {
		return fmt.Errorf("pick between 1 and 50")
	}
	return nil
}

func validatePortList(s string) error {
	if strings.TrimSpace(s) == "" {
		return fmt.Errorf("list at least one port")
	}
	_, err := parsePortList(s)
	return err
}

func validateServiceList(s string) error {
	if strings.TrimSpace(s) == "" {
		return fmt.Errorf("list at least one service")
	}
	_, err := parseServiceList(s)
	return err
}

func validateAliasList(s string) error {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	_, err := parseAliasList(s)
	return err
}

func validateOptionalPortList(s string) error {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	_, err := parsePortList(s)
	return err
}

// parsePortList decodes "3000, 5432,6379" into ports.
func parsePortList(s string) ([]int, error) {
	fields := strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == ' ' })
	out := make([]int, 0, len(fields))
	seen := map[int]bool{}
	for _, f := range fields {
		n, err := strconv.Atoi(strings.TrimSpace(f))
		if err != nil {
			return nil, fmt.Errorf("%q is not a port number", f)
		}
		if err := model.ValidatePort(n); err != nil {
			return nil, err
		}
		if !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no ports found")
	}
	sort.Ints(out)
	return out, nil
}

// parseServiceList decodes "api:8080, db:5432" into blueprint services. This is
// the pragmatic way to describe several services in one modal without turning
// the form into a nested editor.
func parseServiceList(s string) ([]model.BlueprintService, error) {
	entries := strings.Split(s, ",")
	out := make([]model.BlueprintService, 0, len(entries))
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		name, rawPort, found := strings.Cut(entry, ":")
		if !found {
			return nil, fmt.Errorf("%q should look like name:port", entry)
		}
		name = strings.TrimSpace(name)
		if name == "" {
			return nil, fmt.Errorf("%q has no service name", entry)
		}
		port, err := strconv.Atoi(strings.TrimSpace(rawPort))
		if err != nil {
			return nil, fmt.Errorf("%q is not a port number", rawPort)
		}
		if err := model.ValidatePort(port); err != nil {
			return nil, err
		}
		out = append(out, model.BlueprintService{Name: name, Port: port})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no services found")
	}
	return out, nil
}

// parseAliasList decodes "web:3000, api:8080" into an alias map.
func parseAliasList(s string) (map[string]int, error) {
	svcs, err := parseServiceList(s)
	if err != nil {
		return nil, err
	}
	out := make(map[string]int, len(svcs))
	for _, s := range svcs {
		out[s.Name] = s.Port
	}
	return out, nil
}

// --- form builders -------------------------------------------------------

func portForm(id, title, prompt string, initial string) *components.Form {
	return components.NewForm(id, title, []components.FormField{{
		Key:         "port",
		Title:       prompt,
		Placeholder: "3000",
		Value:       initial,
		Validate:    validatePortField,
	}})
}

func killForm() *components.Form {
	return components.NewForm(formKill, "Kill a port", []components.FormField{
		{
			Key:         "port",
			Title:       "Port",
			Description: "Whatever is listening here will be asked to stop.",
			Placeholder: "3000",
			Validate:    validatePortField,
		},
		{
			Key:         "force",
			Title:       "Force (SIGKILL)?",
			Description: "Off sends SIGTERM and lets the process shut down cleanly.",
			Kind:        components.FieldBool,
		},
	})
}

func killPIDForm() *components.Form {
	return components.NewForm(formKillPID, "Kill by PID", []components.FormField{
		{
			Key:         "pid",
			Title:       "Process ID",
			Placeholder: "1234",
			Validate: func(s string) error {
				n, err := strconv.Atoi(strings.TrimSpace(s))
				if err != nil {
					return fmt.Errorf("%q is not a number", s)
				}
				if n <= 0 {
					return fmt.Errorf("pid must be positive")
				}
				return nil
			},
		},
		{Key: "force", Title: "Force (SIGKILL)?", Kind: components.FieldBool},
	})
}

func killRangeForm() *components.Form {
	return components.NewForm(formKillRange, "Kill a port range", []components.FormField{
		{
			Key:         "lo",
			Title:       "From port",
			Placeholder: "3000",
			Validate:    validatePortField,
		},
		{
			Key:         "hi",
			Title:       "To port",
			Description: "Inclusive. Protected ports in the range are skipped.",
			Placeholder: "3010",
			Validate:    validatePortField,
		},
		{Key: "force", Title: "Force (SIGKILL)?", Kind: components.FieldBool},
	})
}

func suggestForm() *components.Form {
	return components.NewForm(formSuggest, "Suggest free ports", []components.FormField{
		{
			Key:         "near",
			Title:       "Near port",
			Description: "Search starts here and walks upward.",
			Placeholder: "3000",
			Validate:    validateOptionalPortField,
		},
		{
			Key:         "count",
			Title:       "How many",
			Placeholder: "3",
			Value:       "3",
			Validate:    validateCount,
		},
	})
}

func reserveForm() *components.Form {
	return components.NewForm(formReserve, "Reserve a port", []components.FormField{
		{
			Key:         "port",
			Title:       "Port",
			Placeholder: "4000",
			Validate:    validatePortField,
		},
		{
			Key:         "service",
			Title:       "Service name",
			Description: "Reserved ports are excluded from future suggestions.",
			Placeholder: "api",
			Validate:    validateName,
		},
	})
}

func aliasForm(existing model.Alias) *components.Form {
	title := "New alias"
	if existing.Name != "" {
		title = "Edit alias " + existing.Name
	}
	port := ""
	if existing.Port > 0 {
		port = strconv.Itoa(existing.Port)
	}
	return components.NewForm(formAlias, title, []components.FormField{
		{
			Key:         "name",
			Title:       "Name",
			Description: "What you want to call this port.",
			Placeholder: "frontend",
			Value:       existing.Name,
			Validate:    validateName,
		},
		{
			Key:         "port",
			Title:       "Port",
			Placeholder: "3000",
			Value:       port,
			Validate:    validatePortField,
		},
		{
			Key:         "note",
			Title:       "Note",
			Description: "Optional.",
			Value:       existing.Note,
		},
	})
}

func groupForm(existing model.Group) *components.Form {
	title := "New group"
	if existing.Name != "" {
		title = "Edit group " + existing.Name
	}
	portList := ""
	if len(existing.Ports) > 0 {
		parts := make([]string, 0, len(existing.Ports))
		for _, p := range existing.Ports {
			parts = append(parts, strconv.Itoa(p))
		}
		portList = strings.Join(parts, ", ")
	}
	return components.NewForm(formGroup, title, []components.FormField{
		{
			Key:         "name",
			Title:       "Name",
			Placeholder: "dev_stack",
			Value:       existing.Name,
			Validate:    validateName,
		},
		{
			Key:         "ports",
			Title:       "Ports",
			Description: "Comma-separated, e.g. 3000, 5432, 6379",
			Placeholder: "3000, 5432, 6379",
			Value:       portList,
			Validate:    validatePortList,
		},
		{Key: "note", Title: "Note", Description: "Optional.", Value: existing.Note},
	})
}

func blueprintForm() *components.Form {
	return components.NewForm(formBlueprint, "New blueprint", []components.FormField{
		{
			Key:         "name",
			Title:       "Name",
			Placeholder: "web",
			Validate:    validateName,
		},
		{Key: "description", Title: "Description", Description: "Optional."},
		{
			Key:         "services",
			Title:       "Services",
			Description: "name:port pairs, e.g. api:8080, db:5432, cache:6379",
			Placeholder: "api:8080, db:5432",
			Validate:    validateServiceList,
		},
	})
}

// pickerForm builds a one-of select. Used wherever an action needs the user to
// choose from something they have already defined.
func pickerForm(id, title, prompt string, options []string) *components.Form {
	return components.NewForm(id, title, []components.FormField{{
		Key:     "choice",
		Title:   prompt,
		Kind:    components.FieldSelect,
		Options: options,
		Value:   options[0],
	}})
}

func projectConfigForm(cfg model.ProjectConfig, dir string) *components.Form {
	expected := ""
	if len(cfg.ExpectedPorts) > 0 {
		parts := make([]string, 0, len(cfg.ExpectedPorts))
		for _, p := range cfg.ExpectedPorts {
			parts = append(parts, strconv.Itoa(p))
		}
		expected = strings.Join(parts, ", ")
	}

	aliases := ""
	if len(cfg.Aliases) > 0 {
		names := make([]string, 0, len(cfg.Aliases))
		for name := range cfg.Aliases {
			names = append(names, name)
		}
		sort.Strings(names)
		parts := make([]string, 0, len(names))
		for _, name := range names {
			parts = append(parts, name+":"+strconv.Itoa(cfg.Aliases[name]))
		}
		aliases = strings.Join(parts, ", ")
	}

	name := cfg.Name
	if name == "" {
		name = defaultProjectName(dir)
	}

	return components.NewForm(formProjectConfig, model.ProjectConfigFile, []components.FormField{
		{
			Key:      "name",
			Title:    "Project name",
			Value:    name,
			Validate: validateName,
		},
		{
			Key:         "expected",
			Title:       "Expected ports",
			Description: "Comma-separated. The ports this project normally uses.",
			Placeholder: "3000, 5432",
			Value:       expected,
			Validate:    validateOptionalPortList,
		},
		{
			Key:         "aliases",
			Title:       "Aliases",
			Description: "name:port pairs, e.g. web:3000, api:8080",
			Placeholder: "web:3000",
			Value:       aliases,
			Validate:    validateAliasList,
		},
		{
			Key:         "blueprint",
			Title:       "Blueprint",
			Description: "Optional: the blueprint this project follows.",
			Value:       cfg.Blueprint,
		},
	})
}

// defaultProjectName falls back to the directory name, which is right often
// enough to save a keystroke.
func defaultProjectName(dir string) string {
	dir = strings.TrimRight(dir, "/")
	if i := strings.LastIndex(dir, "/"); i >= 0 && i < len(dir)-1 {
		return dir[i+1:]
	}
	return dir
}
