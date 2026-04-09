// Command systemd-mcp is an MCP server for systemd service and timer
// management via the Model Context Protocol (stdio transport).
//
// Usage:
//
//	systemd-mcp
package systemd

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/hairglasses-studio/mcpkit/handler"
	"github.com/hairglasses-studio/mcpkit/middleware/gate"
	"github.com/hairglasses-studio/mcpkit/registry"
)

// ---------------------------------------------------------------------------
// I/O types
// ---------------------------------------------------------------------------

// ── systemd_status ─────────────────────────────────────────────────────────

type StatusInput struct {
	Unit   string `json:"unit" jsonschema:"required,description=Systemd unit name (e.g. makima.service or shader-rotate.timer)"`
	System bool   `json:"system,omitempty" jsonschema:"description=Target system scope instead of user scope. Default: user scope."`
}

type StatusOutput struct {
	Unit                 string `json:"unit"`
	ActiveState          string `json:"active_state"`
	SubState             string `json:"sub_state"`
	Description          string `json:"description"`
	LoadState            string `json:"load_state"`
	FragmentPath         string `json:"fragment_path,omitempty"`
	ActiveEnterTimestamp string `json:"active_enter_timestamp,omitempty"`
	MainPID              int    `json:"main_pid,omitempty"`
	MemoryCurrent        string `json:"memory_current,omitempty"`
	CPUUsageNSec         string `json:"cpu_usage_nsec,omitempty"`
}

// ── systemd_start ──────────────────────────────────────────────────────────

type StartInput struct {
	Unit   string `json:"unit" jsonschema:"required,description=Systemd unit name to start"`
	System bool   `json:"system,omitempty" jsonschema:"description=Target system scope instead of user scope. Default: user scope."`
}

type StartOutput struct {
	Unit    string `json:"unit"`
	Message string `json:"message"`
}

// ── systemd_stop ───────────────────────────────────────────────────────────

type StopInput struct {
	Unit    string `json:"unit" jsonschema:"required,description=Systemd unit name to stop"`
	System  bool   `json:"system,omitempty" jsonschema:"description=Target system scope instead of user scope. Default: user scope."`
	Confirm bool   `json:"confirm,omitempty" jsonschema:"description=Required for critical services (sshd, NetworkManager, systemd-*, dbus, polkit)"`
}

type StopOutput struct {
	Unit    string `json:"unit"`
	Message string `json:"message"`
}

// ── systemd_restart ────────────────────────────────────────────────────────

type RestartInput struct {
	Unit   string `json:"unit" jsonschema:"required,description=Systemd unit name to restart"`
	System bool   `json:"system,omitempty" jsonschema:"description=Target system scope instead of user scope. Default: user scope."`
}

type RestartOutput struct {
	Unit    string `json:"unit"`
	Message string `json:"message"`
}

// ── systemd_enable ─────────────────────────────────────────────────────────

type EnableInput struct {
	Unit   string `json:"unit" jsonschema:"required,description=Systemd unit name to enable"`
	System bool   `json:"system,omitempty" jsonschema:"description=Target system scope instead of user scope. Default: user scope."`
}

type EnableOutput struct {
	Unit    string `json:"unit"`
	Message string `json:"message"`
}

// ── systemd_disable ────────────────────────────────────────────────────────

type DisableInput struct {
	Unit    string `json:"unit" jsonschema:"required,description=Systemd unit name to disable"`
	System  bool   `json:"system,omitempty" jsonschema:"description=Target system scope instead of user scope. Default: user scope."`
	Confirm bool   `json:"confirm,omitempty" jsonschema:"description=Required for critical services (sshd, NetworkManager, systemd-*, dbus, polkit)"`
}

type DisableOutput struct {
	Unit    string `json:"unit"`
	Message string `json:"message"`
}

// ── systemd_logs ───────────────────────────────────────────────────────────

type LogsInput struct {
	Unit   string `json:"unit" jsonschema:"required,description=Systemd unit name to fetch logs for"`
	Lines  int    `json:"lines,omitempty" jsonschema:"description=Number of log lines to return. Default 50."`
	Since  string `json:"since,omitempty" jsonschema:"description=Show logs since this time (e.g. '1h ago' or '2024-01-01')"`
	System bool   `json:"system,omitempty" jsonschema:"description=Target system scope instead of user scope. Default: user scope."`
}

type LogsOutput struct {
	Unit  string `json:"unit"`
	Lines int    `json:"lines"`
	Logs  string `json:"logs"`
}

// ── systemd_list_units ─────────────────────────────────────────────────────

type ListUnitsInput struct {
	State  string `json:"state,omitempty" jsonschema:"description=Filter by unit state (e.g. active, inactive, failed)"`
	System bool   `json:"system,omitempty" jsonschema:"description=Target system scope instead of user scope. Default: user scope."`
}

type ListUnitsOutput struct {
	Units json.RawMessage `json:"units"`
}

// ── systemd_list_timers ────────────────────────────────────────────────────

type ListTimersInput struct {
	System bool `json:"system,omitempty" jsonschema:"description=Target system scope instead of user scope. Default: user scope."`
}

type ListTimersOutput struct {
	Timers json.RawMessage `json:"timers"`
}

// ── systemd_failed ─────────────────────────────────────────────────────────

type FailedInput struct {
	System bool `json:"system,omitempty" jsonschema:"description=Target system scope instead of user scope. Default: user scope."`
}

type FailedOutput struct {
	Units json.RawMessage `json:"units"`
}

// ---------------------------------------------------------------------------
// Critical service guard
// ---------------------------------------------------------------------------

// criticalPrefixes lists service name prefixes that require explicit confirmation
// before being stopped or disabled.
var criticalPrefixes = []string{"sshd", "NetworkManager", "systemd-", "dbus", "polkit"}

func requireConfirmation(unit string, confirm bool, action string) error {
	for _, prefix := range criticalPrefixes {
		if strings.HasPrefix(unit, prefix) && !confirm {
			return fmt.Errorf("[%s] %sing critical service %q requires confirm: true",
				handler.ErrInvalidParam, action, unit)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// SystemdModule
// ---------------------------------------------------------------------------

type SystemdModule struct{}

func (m *SystemdModule) Name() string        { return "systemd" }
func (m *SystemdModule) Description() string { return "Systemd service and timer management" }

func (m *SystemdModule) Tools() []registry.ToolDefinition {
	// ── Read-only tools (IsWrite: false) ───────────────────────────────

	status := handler.TypedHandler[StatusInput, StatusOutput](
		"systemd_status",
		"Show detailed status of a systemd unit including active state, PID, memory, and CPU usage.",
		func(_ context.Context, input StatusInput) (StatusOutput, error) {
			return readUnitStatus(input.System, input.Unit)
		},
	)

	listUnits := handler.TypedHandler[ListUnitsInput, ListUnitsOutput](
		"systemd_list_units",
		"List systemd units, optionally filtered by state.",
		func(_ context.Context, input ListUnitsInput) (ListUnitsOutput, error) {
			return listUnits(input.System, input.State)
		},
	)

	listTimers := handler.TypedHandler[ListTimersInput, ListTimersOutput](
		"systemd_list_timers",
		"List active systemd timers with their next/last trigger times.",
		func(_ context.Context, input ListTimersInput) (ListTimersOutput, error) {
			return listTimers(input.System)
		},
	)

	logs := handler.TypedHandler[LogsInput, LogsOutput](
		"systemd_logs",
		"Fetch recent journal logs for a systemd unit.",
		func(_ context.Context, input LogsInput) (LogsOutput, error) {
			return readUnitLogs(input.System, input.Unit, input.Lines, input.Since)
		},
	)
	logs.SearchTerms = []string{"journal", "journald", "service logs", "unit logs"}
	logs.MaxResultChars = 8000

	failed := handler.TypedHandler[FailedInput, FailedOutput](
		"systemd_failed",
		"List failed systemd units.",
		func(_ context.Context, input FailedInput) (FailedOutput, error) {
			return failedUnits(input.System)
		},
	)

	// ── Mutating tools (IsWrite: true, ComplexityModerate) ─────────────

	start := handler.TypedHandler[StartInput, StartOutput](
		"systemd_start",
		"Start a systemd unit.",
		func(_ context.Context, input StartInput) (StartOutput, error) {
			return startUnit(input.System, input.Unit)
		},
	)
	start.IsWrite = true
	start.Complexity = registry.ComplexityModerate

	restart := handler.TypedHandler[RestartInput, RestartOutput](
		"systemd_restart",
		"Restart a systemd unit.",
		func(_ context.Context, input RestartInput) (RestartOutput, error) {
			return restartUnit(input.System, input.Unit)
		},
	)
	restart.IsWrite = true
	restart.Complexity = registry.ComplexityModerate

	enable := handler.TypedHandler[EnableInput, EnableOutput](
		"systemd_enable",
		"Enable a systemd unit to start on boot/login.",
		func(_ context.Context, input EnableInput) (EnableOutput, error) {
			return enableUnit(input.System, input.Unit)
		},
	)
	enable.IsWrite = true
	enable.Complexity = registry.ComplexityModerate

	// ── Destructive tools (IsWrite: true, ComplexityComplex) ───────────

	stop := handler.TypedHandler[StopInput, StopOutput](
		"systemd_stop",
		"Stop a systemd unit. Critical services (sshd, NetworkManager, systemd-*, dbus, polkit) require confirm: true.",
		func(_ context.Context, input StopInput) (StopOutput, error) {
			if err := requireConfirmation(input.Unit, input.Confirm, "stopp"); err != nil {
				return StopOutput{}, err
			}
			return stopUnit(input.System, input.Unit)
		},
	)
	stop.IsWrite = true
	stop.Complexity = registry.ComplexityComplex

	disable := handler.TypedHandler[DisableInput, DisableOutput](
		"systemd_disable",
		"Disable a systemd unit from starting on boot/login. Critical services (sshd, NetworkManager, systemd-*, dbus, polkit) require confirm: true.",
		func(_ context.Context, input DisableInput) (DisableOutput, error) {
			if err := requireConfirmation(input.Unit, input.Confirm, "disabl"); err != nil {
				return DisableOutput{}, err
			}
			return disableUnit(input.System, input.Unit)
		},
	)
	disable.IsWrite = true
	disable.Complexity = registry.ComplexityComplex

	return []registry.ToolDefinition{
		status,
		start,
		stop,
		restart,
		enable,
		disable,
		logs,
		listUnits,
		listTimers,
		failed,
	}
}

// ---------------------------------------------------------------------------
// Setup / main
// ---------------------------------------------------------------------------

func Setup() (*registry.ToolRegistry, *registry.MCPServer) {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})).With("service", "systemd-mcp"))

	slog.Info("server starting", "name", "systemd-mcp", "version", "1.2.0")

	initDBus()

	reg := registry.NewToolRegistry(registry.Config{
		Middleware: []registry.Middleware{
			gate.Middleware(gate.Config{Gate: gate.PauseWrites}),
			registry.AuditMiddleware(""),
			registry.SafetyTierMiddleware(),
		},
	})
	mod := &SystemdModule{}
	reg.RegisterModule(mod)
	slog.Info("tools registered", "module", mod.Name(), "count", len(mod.Tools()))

	s := registry.NewMCPServer("systemd-mcp", "1.2.0")
	reg.RegisterWithServer(s)
	buildSystemdResourceRegistry().RegisterWithServer(s)
	buildSystemdPromptRegistry().RegisterWithServer(s)

	return reg, s
}

func main() {
	_, s := Setup()
	if err := registry.ServeAuto(s); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
