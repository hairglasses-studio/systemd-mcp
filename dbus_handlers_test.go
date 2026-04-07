package main

import (
	"context"
	"encoding/json"
	"testing"
)

// ---------------------------------------------------------------------------
// D-Bus handler path tests — exercise the D-Bus code paths in tool handlers.
// These require a running systemd instance (Linux only, skipped otherwise).
// ---------------------------------------------------------------------------

func requireDBus(t *testing.T) {
	t.Helper()
	requireSystemctl(t) // If no systemctl, no point testing D-Bus either

	// Try to connect via D-Bus
	sdb, err := NewSystemdDBus(false)
	if err != nil {
		t.Skipf("D-Bus session bus not available: %v", err)
	}
	_ = sdb.Close()
}

func withDBus(t *testing.T, fn func()) {
	t.Helper()
	origSession := dbusSession
	origSystem := dbusSystem
	defer func() {
		dbusSession = origSession
		dbusSystem = origSystem
	}()

	var err error
	dbusSession, err = NewSystemdDBus(false)
	if err != nil {
		t.Skipf("D-Bus session bus not available: %v", err)
	}
	defer func() { _ = dbusSession.Close() }()

	dbusSystem, err = NewSystemdDBus(true)
	if err != nil {
		// System bus may not be available in some CI environments
		t.Logf("D-Bus system bus not available: %v", err)
		dbusSystem = nil
	} else {
		defer func() { _ = dbusSystem.Close() }()
	}

	fn()
}

// ---------------------------------------------------------------------------
// systemd_status via D-Bus
// ---------------------------------------------------------------------------

func TestStatus_DBusPath_KnownUnit(t *testing.T) {
	requireDBus(t)

	withDBus(t, func() {
		td := findTool(t, "systemd_status")

		// Try common user-scope units
		for _, unit := range []string{"dbus.service", "pipewire.service", "dbus.socket"} {
			req := makeReq(map[string]any{"unit": unit})
			result, err := td.Handler(context.Background(), req)
			if err != nil {
				continue
			}
			var out StatusOutput
			unmarshalResult(t, result, &out)
			if out.ActiveState == "" {
				continue
			}
			// Successfully tested D-Bus path
			if out.LoadState == "" {
				t.Error("LoadState is empty via D-Bus")
			}
			return
		}
		t.Skip("no known user unit available via D-Bus")
	})
}

func TestStatus_DBusPath_NotFoundUnit(t *testing.T) {
	requireDBus(t)

	withDBus(t, func() {
		td := findTool(t, "systemd_status")
		req := makeReq(map[string]any{"unit": "nonexistent-unit-xyz-99999.service"})
		result, err := td.Handler(context.Background(), req)
		if err != nil {
			// D-Bus path returns error for not-found units
			assertContains(t, err.Error(), "NOT_FOUND")
			return
		}
		// Typed handler may wrap as IsError result
		if result != nil && result.IsError {
			return
		}
		// If D-Bus returned a result with not-found load state, that's also acceptable
		var out StatusOutput
		unmarshalResult(t, result, &out)
		if out.LoadState != "not-found" {
			t.Error("expected not-found load state for nonexistent unit")
		}
	})
}

// ---------------------------------------------------------------------------
// systemd_list_units via D-Bus
// ---------------------------------------------------------------------------

func TestListUnits_DBusPath_Default(t *testing.T) {
	requireDBus(t)

	withDBus(t, func() {
		td := findTool(t, "systemd_list_units")
		req := makeReq(nil)
		result, err := td.Handler(context.Background(), req)
		if err != nil {
			t.Fatalf("handler error: %v", err)
		}

		var out ListUnitsOutput
		unmarshalResult(t, result, &out)
		if !json.Valid(out.Units) {
			t.Errorf("Units is not valid JSON: %s", string(out.Units))
		}
	})
}

func TestListUnits_DBusPath_StateFilter(t *testing.T) {
	requireDBus(t)

	withDBus(t, func() {
		td := findTool(t, "systemd_list_units")
		req := makeReq(map[string]any{"state": "active"})
		result, err := td.Handler(context.Background(), req)
		if err != nil {
			t.Fatalf("handler error: %v", err)
		}

		var out ListUnitsOutput
		unmarshalResult(t, result, &out)
		if !json.Valid(out.Units) {
			t.Errorf("Units is not valid JSON: %s", string(out.Units))
		}
	})
}

// ---------------------------------------------------------------------------
// systemd_list_timers via D-Bus
// ---------------------------------------------------------------------------

func TestListTimers_DBusPath(t *testing.T) {
	requireDBus(t)

	withDBus(t, func() {
		td := findTool(t, "systemd_list_timers")
		req := makeReq(nil)
		result, err := td.Handler(context.Background(), req)
		if err != nil {
			t.Fatalf("handler error: %v", err)
		}

		var out ListTimersOutput
		unmarshalResult(t, result, &out)
		if !json.Valid(out.Timers) {
			t.Errorf("Timers is not valid JSON: %s", string(out.Timers))
		}
	})
}

// ---------------------------------------------------------------------------
// systemd_failed via D-Bus
// ---------------------------------------------------------------------------

func TestFailed_DBusPath(t *testing.T) {
	requireDBus(t)

	withDBus(t, func() {
		td := findTool(t, "systemd_failed")
		req := makeReq(nil)
		result, err := td.Handler(context.Background(), req)
		if err != nil {
			t.Fatalf("handler error: %v", err)
		}

		var out FailedOutput
		unmarshalResult(t, result, &out)
		if !json.Valid(out.Units) {
			t.Errorf("Units is not valid JSON: %s", string(out.Units))
		}
	})
}

// ---------------------------------------------------------------------------
// Module metadata
// ---------------------------------------------------------------------------

func TestSystemdModule_Description(t *testing.T) {
	m := &SystemdModule{}
	if m.Description() == "" {
		t.Error("Description() should not be empty")
	}
	if m.Description() != "Systemd service and timer management" {
		t.Errorf("Description() = %q, want %q", m.Description(), "Systemd service and timer management")
	}
}

// ---------------------------------------------------------------------------
// systemd_disable fallback (missing from fallback_test.go)
// ---------------------------------------------------------------------------

func TestDisable_FallbackToSystemctl(t *testing.T) {
	origSession := dbusSession
	origSystem := dbusSystem
	defer func() {
		dbusSession = origSession
		dbusSystem = origSystem
	}()
	dbusSession = nil
	dbusSystem = nil

	td := findTool(t, "systemd_disable")
	req := makeReq(map[string]any{"unit": "nonexistent.service"})
	_, err := td.Handler(context.Background(), req)
	if err == nil {
		t.Log("disable succeeded (systemctl available)")
	}
	// Should not panic regardless
}

func TestDisable_RequiresConfirmation_AllPrefixes(t *testing.T) {
	td := findTool(t, "systemd_disable")

	for _, unit := range []string{
		"sshd.service",
		"NetworkManager.service",
		"systemd-logind.service",
		"dbus.service",
		"polkit.service",
	} {
		t.Run(unit, func(t *testing.T) {
			req := makeReq(map[string]any{"unit": unit})
			result, err := td.Handler(context.Background(), req)
			if err != nil {
				assertContains(t, err.Error(), "INVALID_PARAM")
				return
			}
			if result == nil || !result.IsError {
				t.Fatalf("expected error for critical service %s without confirm", unit)
			}
			assertContains(t, extractText(t, result), "INVALID_PARAM")
		})
	}
}

// ---------------------------------------------------------------------------
// systemd_stop additional confirmation guard tests
// ---------------------------------------------------------------------------

func TestStop_RequiresConfirmation_AllPrefixes(t *testing.T) {
	td := findTool(t, "systemd_stop")

	for _, unit := range []string{
		"sshd.service",
		"NetworkManager.service",
		"systemd-resolved.service",
		"dbus.service",
		"polkit.service",
	} {
		t.Run(unit, func(t *testing.T) {
			req := makeReq(map[string]any{"unit": unit})
			result, err := td.Handler(context.Background(), req)
			if err != nil {
				assertContains(t, err.Error(), "INVALID_PARAM")
				return
			}
			if result == nil || !result.IsError {
				t.Fatalf("expected error for critical service %s without confirm", unit)
			}
			assertContains(t, extractText(t, result), "INVALID_PARAM")
		})
	}
}

// ---------------------------------------------------------------------------
// systemd_logs edge cases
// ---------------------------------------------------------------------------

func TestLogs_WithSince(t *testing.T) {
	requireSystemctl(t)

	td := findTool(t, "systemd_logs")
	// Exercise the "since" parameter path
	req := makeReq(map[string]any{
		"unit":  "dbus.service",
		"lines": 5,
		"since": "1h ago",
	})
	result, err := td.Handler(context.Background(), req)
	if err != nil {
		t.Skipf("journalctl failed: %v", err)
	}

	var out LogsOutput
	unmarshalResult(t, result, &out)
	if out.Lines != 5 {
		t.Errorf("expected lines=5, got %d", out.Lines)
	}
	if out.Unit != "dbus.service" {
		t.Errorf("expected unit=dbus.service, got %q", out.Unit)
	}
}

func TestLogs_NegativeLines(t *testing.T) {
	requireSystemctl(t)

	td := findTool(t, "systemd_logs")
	// Negative lines should default to 50
	req := makeReq(map[string]any{"unit": "dbus.service", "lines": -1})
	result, err := td.Handler(context.Background(), req)
	if err != nil {
		t.Skipf("journalctl failed: %v", err)
	}

	var out LogsOutput
	unmarshalResult(t, result, &out)
	if out.Lines != 50 {
		t.Errorf("expected default lines=50 for negative input, got %d", out.Lines)
	}
}

// ---------------------------------------------------------------------------
// unitsToJSON nil edge case
// ---------------------------------------------------------------------------

func TestUnitsToJSON_Nil(t *testing.T) {
	// nil slice should marshal to "null" in Go, not "[]"
	raw, err := unitsToJSON(nil)
	if err != nil {
		t.Fatalf("unitsToJSON error: %v", err)
	}
	// Go marshals nil slices as "null"
	if string(raw) != "null" {
		t.Errorf("expected null for nil slice, got %s", string(raw))
	}
}
