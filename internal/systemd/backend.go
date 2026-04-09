package systemd

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/hairglasses-studio/mcpkit/handler"
)

type backendFailure struct {
	Operation       string
	Scope           string
	PrimaryBackend  string
	PrimaryErr      error
	FallbackBackend string
	FallbackErr     error
}

func (e *backendFailure) Error() string {
	parts := make([]string, 0, 2)
	if e.PrimaryErr != nil {
		parts = append(parts, fmt.Sprintf("%s failed: %v", e.PrimaryBackend, e.PrimaryErr))
	}
	if e.FallbackErr != nil && e.FallbackBackend != "" {
		parts = append(parts, fmt.Sprintf("%s failed: %v", e.FallbackBackend, e.FallbackErr))
	}
	return fmt.Sprintf("%s failed for %s scope: %s", e.Operation, e.Scope, strings.Join(parts, "; "))
}

func (e *backendFailure) Unwrap() []error {
	errs := make([]error, 0, 2)
	if e.PrimaryErr != nil {
		errs = append(errs, e.PrimaryErr)
	}
	if e.FallbackErr != nil {
		errs = append(errs, e.FallbackErr)
	}
	return errs
}

func backendErrorCode(primaryErr, fallbackErr error) string {
	if isPermissionErr(primaryErr) || isPermissionErr(fallbackErr) {
		return handler.ErrPermission
	}
	return handler.ErrUpstreamError
}

func newBackendFailure(operation string, system bool, primaryBackend string, primaryErr error, fallbackBackend string, fallbackErr error) error {
	failure := &backendFailure{
		Operation:       operation,
		Scope:           scopeName(system),
		PrimaryBackend:  primaryBackend,
		PrimaryErr:      primaryErr,
		FallbackBackend: fallbackBackend,
		FallbackErr:     fallbackErr,
	}
	return fmt.Errorf("[%s] %w", backendErrorCode(primaryErr, fallbackErr), failure)
}

func newSingleBackendFailure(operation string, system bool, backend string, err error) error {
	return newBackendFailure(operation, system, backend, err, "", nil)
}

func isPermissionErr(err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(err.Error())
	for _, marker := range []string{
		"permission denied",
		"access denied",
		"authentication is required",
		"interactive authentication required",
		"not authorized",
		"authorization failed",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func isUnitMissingErr(err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(err.Error())
	for _, marker := range []string{
		"could not be found",
		"no such unit",
		"nosuchunit",
		"loadstate=not-found",
		"unit not found",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func requireDBusBackend(system bool) (*SystemdDBus, error) {
	sdb := getDBus(system)
	if sdb == nil {
		return nil, fmt.Errorf("dbus backend unavailable for %s scope: dbus connection not initialized", scopeName(system))
	}
	if err := probeDBusManager(sdb); err != nil {
		return nil, fmt.Errorf("dbus backend unavailable for %s scope: %w", scopeName(system), err)
	}
	return sdb, nil
}

func requireSystemctlBackend(system bool) error {
	if err := probeSystemctl(!system); err != nil {
		return fmt.Errorf("systemctl backend unavailable for %s scope: %w", scopeName(system), err)
	}
	return nil
}

func requireJournalctlBackend(system bool) error {
	if err := probeJournalctl(!system); err != nil {
		return fmt.Errorf("journalctl backend unavailable for %s scope: %w", scopeName(system), err)
	}
	return nil
}

func parseStatusOutput(unit, out string) StatusOutput {
	result := StatusOutput{Unit: unit}
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
	return result
}

func readUnitStatus(system bool, unit string) (StatusOutput, error) {
	result, primaryErr := readUnitStatusViaDBus(system, unit)
	switch {
	case primaryErr == nil && result.LoadState == "not-found":
		return result, fmt.Errorf("[%s] unit %s not found", handler.ErrNotFound, unit)
	case primaryErr == nil:
		return result, nil
	case isUnitMissingErr(primaryErr):
		return StatusOutput{}, fmt.Errorf("[%s] unit %s not found", handler.ErrNotFound, unit)
	}

	result, fallbackErr := readUnitStatusViaSystemctl(system, unit)
	switch {
	case fallbackErr == nil && result.LoadState == "not-found":
		return result, fmt.Errorf("[%s] unit %s not found", handler.ErrNotFound, unit)
	case fallbackErr == nil:
		return result, nil
	case isUnitMissingErr(fallbackErr):
		return StatusOutput{}, fmt.Errorf("[%s] unit %s not found", handler.ErrNotFound, unit)
	default:
		return StatusOutput{}, newBackendFailure("systemd_status", system, "dbus", primaryErr, "systemctl", fallbackErr)
	}
}

func readUnitStatusViaDBus(system bool, unit string) (StatusOutput, error) {
	sdb, err := requireDBusBackend(system)
	if err != nil {
		return StatusOutput{}, err
	}
	us, err := sdb.GetUnitStatus(unit)
	if err != nil {
		return StatusOutput{}, fmt.Errorf("dbus status %s: %w", unit, err)
	}
	return unitStatusToOutput(unit, us), nil
}

func readUnitStatusViaSystemctl(system bool, unit string) (StatusOutput, error) {
	if err := requireSystemctlBackend(system); err != nil {
		return StatusOutput{}, err
	}
	out, err := runSystemctl(!system, "show",
		"--property=ActiveState,SubState,Description,LoadState,FragmentPath,ActiveEnterTimestamp,MainPID,MemoryCurrent,CPUUsageNSec",
		unit,
	)
	if err != nil {
		return StatusOutput{}, fmt.Errorf("systemctl status %s: %w", unit, err)
	}
	return parseStatusOutput(unit, out), nil
}

func listUnits(system bool, state string) (ListUnitsOutput, error) {
	out, primaryErr := listUnitsViaDBus(system, state)
	if primaryErr == nil {
		return out, nil
	}

	out, fallbackErr := listUnitsViaSystemctl(system, state)
	if fallbackErr == nil {
		return out, nil
	}
	return ListUnitsOutput{}, newBackendFailure("systemd_list_units", system, "dbus", primaryErr, "systemctl", fallbackErr)
}

func listUnitsViaDBus(system bool, state string) (ListUnitsOutput, error) {
	sdb, err := requireDBusBackend(system)
	if err != nil {
		return ListUnitsOutput{}, err
	}

	var units []UnitInfo
	if state != "" {
		units, err = sdb.ListUnitsFiltered([]string{state})
	} else {
		units, err = sdb.ListUnits()
	}
	if err != nil {
		return ListUnitsOutput{}, fmt.Errorf("dbus list units: %w", err)
	}
	raw, err := unitsToJSON(units)
	if err != nil {
		return ListUnitsOutput{}, fmt.Errorf("marshal dbus units: %w", err)
	}
	return ListUnitsOutput{Units: raw}, nil
}

func listUnitsViaSystemctl(system bool, state string) (ListUnitsOutput, error) {
	if err := requireSystemctlBackend(system); err != nil {
		return ListUnitsOutput{}, err
	}

	args := []string{"list-units", "--output=json"}
	if state != "" {
		args = append(args, "--state="+state)
	}
	out, err := runSystemctl(!system, args...)
	if err != nil {
		return ListUnitsOutput{}, fmt.Errorf("systemctl list units: %w", err)
	}
	return ListUnitsOutput{Units: json.RawMessage(out)}, nil
}

func listTimers(system bool) (ListTimersOutput, error) {
	out, primaryErr := listTimersViaDBus(system)
	if primaryErr == nil {
		return out, nil
	}

	out, fallbackErr := listTimersViaSystemctl(system)
	if fallbackErr == nil {
		return out, nil
	}
	return ListTimersOutput{}, newBackendFailure("systemd_list_timers", system, "dbus", primaryErr, "systemctl", fallbackErr)
}

func listTimersViaDBus(system bool) (ListTimersOutput, error) {
	sdb, err := requireDBusBackend(system)
	if err != nil {
		return ListTimersOutput{}, err
	}
	timers, err := sdb.ListTimers()
	if err != nil {
		return ListTimersOutput{}, fmt.Errorf("dbus list timers: %w", err)
	}
	raw, err := timersToJSON(timers)
	if err != nil {
		return ListTimersOutput{}, fmt.Errorf("marshal dbus timers: %w", err)
	}
	return ListTimersOutput{Timers: raw}, nil
}

func listTimersViaSystemctl(system bool) (ListTimersOutput, error) {
	if err := requireSystemctlBackend(system); err != nil {
		return ListTimersOutput{}, err
	}
	out, err := runSystemctl(!system, "list-timers", "--output=json", "--no-pager")
	if err != nil {
		return ListTimersOutput{}, fmt.Errorf("systemctl list timers: %w", err)
	}
	return ListTimersOutput{Timers: json.RawMessage(out)}, nil
}

func readUnitLogs(system bool, unit string, lines int, since string) (LogsOutput, error) {
	if lines <= 0 {
		lines = 50
	}
	if err := requireJournalctlBackend(system); err != nil {
		return LogsOutput{}, newSingleBackendFailure("systemd_logs", system, "journalctl", err)
	}

	args := []string{"-u", unit, "-n", strconv.Itoa(lines)}
	if !system {
		args = []string{"--user-unit", unit, "-n", strconv.Itoa(lines)}
	}
	if since != "" {
		args = append(args, "--since", since)
	}
	args = append(args, "--no-pager")

	out, err := runJournalctl(!system, args...)
	if err != nil {
		return LogsOutput{}, newSingleBackendFailure("systemd_logs", system, "journalctl", fmt.Errorf("journalctl logs %s: %w", unit, err))
	}
	return LogsOutput{
		Unit:  unit,
		Lines: lines,
		Logs:  out,
	}, nil
}

func failedUnits(system bool) (FailedOutput, error) {
	out, primaryErr := failedUnitsViaDBus(system)
	if primaryErr == nil {
		return out, nil
	}

	out, fallbackErr := failedUnitsViaSystemctl(system)
	if fallbackErr == nil {
		return out, nil
	}
	return FailedOutput{}, newBackendFailure("systemd_failed", system, "dbus", primaryErr, "systemctl", fallbackErr)
}

func failedUnitsViaDBus(system bool) (FailedOutput, error) {
	sdb, err := requireDBusBackend(system)
	if err != nil {
		return FailedOutput{}, err
	}
	units, err := sdb.GetFailedUnits()
	if err != nil {
		return FailedOutput{}, fmt.Errorf("dbus failed units: %w", err)
	}
	raw, err := unitsToJSON(units)
	if err != nil {
		return FailedOutput{}, fmt.Errorf("marshal failed units: %w", err)
	}
	return FailedOutput{Units: raw}, nil
}

func failedUnitsViaSystemctl(system bool) (FailedOutput, error) {
	if err := requireSystemctlBackend(system); err != nil {
		return FailedOutput{}, err
	}
	out, err := runSystemctl(!system, "--failed", "--output=json", "--no-pager")
	if err != nil {
		return FailedOutput{}, fmt.Errorf("systemctl failed units: %w", err)
	}
	return FailedOutput{Units: json.RawMessage(out)}, nil
}

func performWriteOperation(system bool, operation, unit, successMessage string, dbusAction func(*SystemdDBus) error, systemctlArgs ...string) (string, error) {
	backend, primaryErr := performWriteViaDBus(system, operation, unit, successMessage, dbusAction)
	if primaryErr == nil {
		return backend, nil
	}
	if isUnitMissingErr(primaryErr) {
		return "", fmt.Errorf("[%s] unit %s not found", handler.ErrNotFound, unit)
	}

	backend, fallbackErr := performWriteViaSystemctl(system, operation, unit, successMessage, systemctlArgs...)
	if fallbackErr == nil {
		return backend, nil
	}
	if isUnitMissingErr(fallbackErr) {
		return "", fmt.Errorf("[%s] unit %s not found", handler.ErrNotFound, unit)
	}

	return "", newBackendFailure(operation, system, "dbus", primaryErr, "systemctl", fallbackErr)
}

func performWriteViaDBus(system bool, operation, unit, successMessage string, action func(*SystemdDBus) error) (string, error) {
	sdb, err := requireDBusBackend(system)
	if err != nil {
		return "", err
	}
	if err := action(sdb); err != nil {
		return "", fmt.Errorf("dbus %s %s: %w", operation, unit, err)
	}
	return "dbus", nil
}

func performWriteViaSystemctl(system bool, operation, unit, successMessage string, args ...string) (string, error) {
	if err := requireSystemctlBackend(system); err != nil {
		return "", err
	}
	if _, err := runSystemctl(!system, args...); err != nil {
		return "", fmt.Errorf("systemctl %s %s: %w", operation, unit, err)
	}
	return "systemctl", nil
}

func startUnit(system bool, unit string) (StartOutput, error) {
	slog.Info("starting unit", "unit", unit, "system", system)
	backend, err := performWriteOperation(system, "systemd_start", unit, unit+" started", func(sdb *SystemdDBus) error {
		return sdb.StartUnit(unit, "replace")
	}, "start", unit)
	if err != nil {
		slog.Error("unit start failed", "unit", unit, "error", err)
		return StartOutput{}, err
	}
	slog.Info("unit started", "unit", unit, "via", backend)
	return StartOutput{Unit: unit, Message: unit + " started"}, nil
}

func restartUnit(system bool, unit string) (RestartOutput, error) {
	slog.Info("restarting unit", "unit", unit, "system", system)
	backend, err := performWriteOperation(system, "systemd_restart", unit, unit+" restarted", func(sdb *SystemdDBus) error {
		return sdb.RestartUnit(unit, "replace")
	}, "restart", unit)
	if err != nil {
		slog.Error("unit restart failed", "unit", unit, "error", err)
		return RestartOutput{}, err
	}
	slog.Info("unit restarted", "unit", unit, "via", backend)
	return RestartOutput{Unit: unit, Message: unit + " restarted"}, nil
}

func enableUnit(system bool, unit string) (EnableOutput, error) {
	slog.Info("enabling unit", "unit", unit, "system", system)
	backend, err := performWriteOperation(system, "systemd_enable", unit, unit+" enabled", func(sdb *SystemdDBus) error {
		return sdb.EnableUnit(unit)
	}, "enable", unit)
	if err != nil {
		slog.Error("unit enable failed", "unit", unit, "error", err)
		return EnableOutput{}, err
	}
	slog.Info("unit enabled", "unit", unit, "via", backend)
	return EnableOutput{Unit: unit, Message: unit + " enabled"}, nil
}

func stopUnit(system bool, unit string) (StopOutput, error) {
	slog.Info("stopping unit", "unit", unit, "system", system)
	backend, err := performWriteOperation(system, "systemd_stop", unit, unit+" stopped", func(sdb *SystemdDBus) error {
		return sdb.StopUnit(unit, "replace")
	}, "stop", unit)
	if err != nil {
		slog.Error("unit stop failed", "unit", unit, "error", err)
		return StopOutput{}, err
	}
	slog.Info("unit stopped", "unit", unit, "via", backend)
	return StopOutput{Unit: unit, Message: unit + " stopped"}, nil
}

func disableUnit(system bool, unit string) (DisableOutput, error) {
	slog.Info("disabling unit", "unit", unit, "system", system)
	backend, err := performWriteOperation(system, "systemd_disable", unit, unit+" disabled", func(sdb *SystemdDBus) error {
		return sdb.DisableUnit(unit)
	}, "disable", unit)
	if err != nil {
		slog.Error("unit disable failed", "unit", unit, "error", err)
		return DisableOutput{}, err
	}
	slog.Info("unit disabled", "unit", unit, "via", backend)
	return DisableOutput{Unit: unit, Message: unit + " disabled"}, nil
}
