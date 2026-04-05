// Command systemd-mcp is an MCP server for systemd service and timer
// management via the Model Context Protocol (stdio transport).
//
// Usage:
//
//	systemd-mcp
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/hairglasses-studio/mcpkit/handler"
	"github.com/hairglasses-studio/mcpkit/registry"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func runCmd(name string, args ...string) (string, string, error) {
	cmd := exec.Command(name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()), err
}

func runSystemctl(user bool, args ...string) (string, error) {
	cmdArgs := make([]string, 0, len(args)+1)
	if user {
		cmdArgs = append(cmdArgs, "--user")
	}
	cmdArgs = append(cmdArgs, args...)
	stdout, stderr, err := runCmd("systemctl", cmdArgs...)
	if err != nil {
		return "", fmt.Errorf("systemctl %s: %s: %w", strings.Join(cmdArgs, " "), stderr, err)
	}
	return stdout, nil
}

func runJournalctl(user bool, args ...string) (string, error) {
	cmdArgs := make([]string, 0, len(args)+1)
	if user {
		cmdArgs = append(cmdArgs, "--user-unit")
	} else {
		cmdArgs = append(cmdArgs, "-u")
	}
	cmdArgs = append(cmdArgs, args...)
	stdout, stderr, err := runCmd("journalctl", cmdArgs...)
	if err != nil {
		return "", fmt.Errorf("journalctl: %s: %w", stderr, err)
	}
	return stdout, nil
}

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
			// Try D-Bus first.
			if sdb := getDBus(input.System); sdb != nil {
				us, err := sdb.GetUnitStatus(input.Unit)
				if err == nil {
					if us.LoadState == "not-found" {
						out := unitStatusToOutput(input.Unit, us)
						return out, fmt.Errorf("[%s] unit %s not found", handler.ErrNotFound, input.Unit)
					}
					return unitStatusToOutput(input.Unit, us), nil
				}
				// D-Bus call failed — fall through to systemctl.
			}

			// Fallback: systemctl.
			user := !input.System
			out, err := runSystemctl(user, "show",
				"--property=ActiveState,SubState,Description,LoadState,FragmentPath,ActiveEnterTimestamp,MainPID,MemoryCurrent,CPUUsageNSec",
				input.Unit,
			)
			if err != nil {
				return StatusOutput{}, fmt.Errorf("[%s] %w", handler.ErrNotFound, err)
			}

			result := StatusOutput{Unit: input.Unit}
			for line := range strings.SplitSeq(out, "\n") {
				parts := strings.SplitN(line, "=", 2)
				if len(parts) != 2 {
					continue
				}
				key, val := parts[0], parts[1]
				switch key {
				case "ActiveState":
					result.ActiveState = val
				case "SubState":
					result.SubState = val
				case "Description":
					result.Description = val
				case "LoadState":
					result.LoadState = val
				case "FragmentPath":
					result.FragmentPath = val
				case "ActiveEnterTimestamp":
					result.ActiveEnterTimestamp = val
				case "MainPID":
					result.MainPID, _ = strconv.Atoi(val)
				case "MemoryCurrent":
					if val != "[not set]" {
						result.MemoryCurrent = val
					}
				case "CPUUsageNSec":
					if val != "[not set]" {
						result.CPUUsageNSec = val
					}
				}
			}

			if result.LoadState == "not-found" {
				return result, fmt.Errorf("[%s] unit %s not found", handler.ErrNotFound, input.Unit)
			}

			return result, nil
		},
	)

	listUnits := handler.TypedHandler[ListUnitsInput, ListUnitsOutput](
		"systemd_list_units",
		"List systemd units, optionally filtered by state.",
		func(_ context.Context, input ListUnitsInput) (ListUnitsOutput, error) {
			// Try D-Bus first.
			if sdb := getDBus(input.System); sdb != nil {
				var units []UnitInfo
				var err error
				if input.State != "" {
					units, err = sdb.ListUnitsFiltered([]string{input.State})
				} else {
					units, err = sdb.ListUnits()
				}
				if err == nil {
					raw, jerr := unitsToJSON(units)
					if jerr == nil {
						return ListUnitsOutput{Units: raw}, nil
					}
				}
				// D-Bus failed — fall through to systemctl.
			}

			// Fallback: systemctl.
			user := !input.System
			args := []string{"list-units", "--output=json"}
			if input.State != "" {
				args = append(args, "--state="+input.State)
			}
			out, err := runSystemctl(user, args...)
			if err != nil {
				return ListUnitsOutput{}, fmt.Errorf("[%s] %w", handler.ErrUpstreamError, err)
			}
			return ListUnitsOutput{
				Units: json.RawMessage(out),
			}, nil
		},
	)

	listTimers := handler.TypedHandler[ListTimersInput, ListTimersOutput](
		"systemd_list_timers",
		"List active systemd timers with their next/last trigger times.",
		func(_ context.Context, input ListTimersInput) (ListTimersOutput, error) {
			// Try D-Bus first.
			if sdb := getDBus(input.System); sdb != nil {
				timers, err := sdb.ListTimers()
				if err == nil {
					raw, jerr := timersToJSON(timers)
					if jerr == nil {
						return ListTimersOutput{Timers: raw}, nil
					}
				}
				// D-Bus failed — fall through to systemctl.
			}

			// Fallback: systemctl.
			user := !input.System
			out, err := runSystemctl(user, "list-timers", "--output=json", "--no-pager")
			if err != nil {
				return ListTimersOutput{}, fmt.Errorf("[%s] %w", handler.ErrUpstreamError, err)
			}
			return ListTimersOutput{
				Timers: json.RawMessage(out),
			}, nil
		},
	)

	logs := handler.TypedHandler[LogsInput, LogsOutput](
		"systemd_logs",
		"Fetch recent journal logs for a systemd unit.",
		func(_ context.Context, input LogsInput) (LogsOutput, error) {
			user := !input.System
			lines := input.Lines
			if lines <= 0 {
				lines = 50
			}

			args := []string{input.Unit, "-n", strconv.Itoa(lines)}
			if input.Since != "" {
				args = append(args, "--since", input.Since)
			}
			args = append(args, "--no-pager")

			out, err := runJournalctl(user, args...)
			if err != nil {
				return LogsOutput{}, fmt.Errorf("[%s] %w", handler.ErrUpstreamError, err)
			}
			return LogsOutput{
				Unit:  input.Unit,
				Lines: lines,
				Logs:  out,
			}, nil
		},
	)
	logs.SearchTerms = []string{"journal", "journald", "service logs", "unit logs"}
	logs.MaxResultChars = 8000

	failed := handler.TypedHandler[FailedInput, FailedOutput](
		"systemd_failed",
		"List failed systemd units.",
		func(_ context.Context, input FailedInput) (FailedOutput, error) {
			// Try D-Bus first.
			if sdb := getDBus(input.System); sdb != nil {
				units, err := sdb.GetFailedUnits()
				if err == nil {
					raw, jerr := unitsToJSON(units)
					if jerr == nil {
						return FailedOutput{Units: raw}, nil
					}
				}
				// D-Bus failed — fall through to systemctl.
			}

			// Fallback: systemctl.
			user := !input.System
			out, err := runSystemctl(user, "--failed", "--output=json", "--no-pager")
			if err != nil {
				return FailedOutput{}, fmt.Errorf("[%s] %w", handler.ErrUpstreamError, err)
			}
			return FailedOutput{
				Units: json.RawMessage(out),
			}, nil
		},
	)

	// ── Mutating tools (IsWrite: true, ComplexityModerate) ─────────────

	start := handler.TypedHandler[StartInput, StartOutput](
		"systemd_start",
		"Start a systemd unit.",
		func(_ context.Context, input StartInput) (StartOutput, error) {
			// Try D-Bus first.
			if sdb := getDBus(input.System); sdb != nil {
				if err := sdb.StartUnit(input.Unit, "replace"); err == nil {
					return StartOutput{
						Unit:    input.Unit,
						Message: input.Unit + " started",
					}, nil
				}
				// D-Bus failed — fall through to systemctl.
			}

			// Fallback: systemctl.
			user := !input.System
			_, err := runSystemctl(user, "start", input.Unit)
			if err != nil {
				return StartOutput{}, fmt.Errorf("[%s] %w", handler.ErrUpstreamError, err)
			}
			return StartOutput{
				Unit:    input.Unit,
				Message: input.Unit + " started",
			}, nil
		},
	)
	start.IsWrite = true
	start.Complexity = registry.ComplexityModerate

	restart := handler.TypedHandler[RestartInput, RestartOutput](
		"systemd_restart",
		"Restart a systemd unit.",
		func(_ context.Context, input RestartInput) (RestartOutput, error) {
			// Try D-Bus first.
			if sdb := getDBus(input.System); sdb != nil {
				if err := sdb.RestartUnit(input.Unit, "replace"); err == nil {
					return RestartOutput{
						Unit:    input.Unit,
						Message: input.Unit + " restarted",
					}, nil
				}
				// D-Bus failed — fall through to systemctl.
			}

			// Fallback: systemctl.
			user := !input.System
			_, err := runSystemctl(user, "restart", input.Unit)
			if err != nil {
				return RestartOutput{}, fmt.Errorf("[%s] %w", handler.ErrUpstreamError, err)
			}
			return RestartOutput{
				Unit:    input.Unit,
				Message: input.Unit + " restarted",
			}, nil
		},
	)
	restart.IsWrite = true
	restart.Complexity = registry.ComplexityModerate

	enable := handler.TypedHandler[EnableInput, EnableOutput](
		"systemd_enable",
		"Enable a systemd unit to start on boot/login.",
		func(_ context.Context, input EnableInput) (EnableOutput, error) {
			// Try D-Bus first.
			if sdb := getDBus(input.System); sdb != nil {
				if err := sdb.EnableUnit(input.Unit); err == nil {
					return EnableOutput{
						Unit:    input.Unit,
						Message: input.Unit + " enabled",
					}, nil
				}
				// D-Bus failed — fall through to systemctl.
			}

			// Fallback: systemctl.
			user := !input.System
			_, err := runSystemctl(user, "enable", input.Unit)
			if err != nil {
				return EnableOutput{}, fmt.Errorf("[%s] %w", handler.ErrUpstreamError, err)
			}
			return EnableOutput{
				Unit:    input.Unit,
				Message: input.Unit + " enabled",
			}, nil
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

			// Try D-Bus first.
			if sdb := getDBus(input.System); sdb != nil {
				if err := sdb.StopUnit(input.Unit, "replace"); err == nil {
					return StopOutput{
						Unit:    input.Unit,
						Message: input.Unit + " stopped",
					}, nil
				}
				// D-Bus failed — fall through to systemctl.
			}

			// Fallback: systemctl.
			user := !input.System
			_, err := runSystemctl(user, "stop", input.Unit)
			if err != nil {
				return StopOutput{}, fmt.Errorf("[%s] %w", handler.ErrUpstreamError, err)
			}
			return StopOutput{
				Unit:    input.Unit,
				Message: input.Unit + " stopped",
			}, nil
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

			// Try D-Bus first.
			if sdb := getDBus(input.System); sdb != nil {
				if err := sdb.DisableUnit(input.Unit); err == nil {
					return DisableOutput{
						Unit:    input.Unit,
						Message: input.Unit + " disabled",
					}, nil
				}
				// D-Bus failed — fall through to systemctl.
			}

			// Fallback: systemctl.
			user := !input.System
			_, err := runSystemctl(user, "disable", input.Unit)
			if err != nil {
				return DisableOutput{}, fmt.Errorf("[%s] %w", handler.ErrUpstreamError, err)
			}
			return DisableOutput{
				Unit:    input.Unit,
				Message: input.Unit + " disabled",
			}, nil
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
// main
// ---------------------------------------------------------------------------

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})).With("service", "systemd-mcp"))

	initDBus()

	reg := registry.NewToolRegistry(registry.Config{
		Middleware: []registry.Middleware{
			registry.AuditMiddleware(""),
			registry.SafetyTierMiddleware(),
		},
	})
	reg.RegisterModule(&SystemdModule{})

	s := registry.NewMCPServer("systemd-mcp", "1.0.0")
	reg.RegisterWithServer(s)
	buildSystemdResourceRegistry().RegisterWithServer(s)
	buildSystemdPromptRegistry().RegisterWithServer(s)

	if err := registry.ServeAuto(s); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
