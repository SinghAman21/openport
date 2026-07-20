package model

import "time"

// ActionKind classifies a history entry.
type ActionKind string

const (
	ActionKill           ActionKind = "kill"
	ActionKillRange      ActionKind = "kill_range"
	ActionKillPID        ActionKind = "kill_pid"
	ActionCheck          ActionKind = "check"
	ActionSuggest        ActionKind = "suggest"
	ActionReserve        ActionKind = "reserve"
	ActionHealth         ActionKind = "health"
	ActionApplyBlueprint ActionKind = "apply_blueprint"
	ActionAlias          ActionKind = "alias"
	ActionGroup          ActionKind = "group"
	ActionConfig         ActionKind = "config"
	ActionProtect        ActionKind = "protect"
	ActionSync           ActionKind = "sync"
)

// HistoryEntry is the audit record written for every mutating action.
type HistoryEntry struct {
	At      time.Time  `json:"at"`
	Kind    ActionKind `json:"kind"`
	Summary string     `json:"summary"`
	Detail  string     `json:"detail,omitempty"`
	Port    int        `json:"port,omitempty"`
	PID     int        `json:"pid,omitempty"`
	Process string     `json:"process,omitempty"`
	Success bool       `json:"success"`
	Err     string     `json:"err,omitempty"`
}

// LogLevel is the severity of a LogEntry.
type LogLevel string

const (
	LogInfo  LogLevel = "info"
	LogWarn  LogLevel = "warn"
	LogError LogLevel = "error"
)

// LogEntry is an internal diagnostic line — OS interaction errors, permission
// failures, fallback notices. Surfaced in the Logs tab.
type LogEntry struct {
	At      time.Time `json:"at"`
	Level   LogLevel  `json:"level"`
	Source  string    `json:"source"`
	Message string    `json:"message"`
}
