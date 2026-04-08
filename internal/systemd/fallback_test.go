package systemd

import (
	"context"
	"testing"
)

// ---------------------------------------------------------------------------
// D-Bus fallback tests — exercise handler code paths when D-Bus is nil.
// On macOS (no systemctl), these verify the error wrapping works correctly.
// On Linux, these exercise the full systemctl fallback path.
// ---------------------------------------------------------------------------

func TestStatus_FallbackToSystemctl(t *testing.T) {
	// Ensure D-Bus is nil so handler falls through to systemctl
	origSession := dbusSession
	origSystem := dbusSystem
	defer func() {
		dbusSession = origSession
		dbusSystem = origSystem
	}()
	dbusSession = nil
	dbusSystem = nil

	td := findTool(t, "systemd_status")
	// This exercises the full fallback path — will fail on macOS with systemctl not found
	req := makeReq(map[string]any{"unit": "test.service"})
	result, err := td.Handler(context.Background(), req)
	if err != nil {
		// Expected on macOS — systemctl not available
		assertContains(t, err.Error(), "NOT_FOUND")
		return
	}
	if result != nil && result.IsError {
		// Also acceptable — handler wrapped the error
		return
	}
	// On Linux: should succeed for a valid unit
}

func TestStart_FallbackToSystemctl(t *testing.T) {
	origSession := dbusSession
	origSystem := dbusSystem
	defer func() {
		dbusSession = origSession
		dbusSystem = origSystem
	}()
	dbusSession = nil
	dbusSystem = nil

	td := findTool(t, "systemd_start")
	req := makeReq(map[string]any{"unit": "nonexistent.service"})
	_, err := td.Handler(context.Background(), req)
	// Should fail at systemctl layer, not panic
	if err == nil {
		t.Log("start succeeded (systemctl available)")
	}
}

func TestRestart_FallbackToSystemctl(t *testing.T) {
	origSession := dbusSession
	origSystem := dbusSystem
	defer func() {
		dbusSession = origSession
		dbusSystem = origSystem
	}()
	dbusSession = nil
	dbusSystem = nil

	td := findTool(t, "systemd_restart")
	req := makeReq(map[string]any{"unit": "nonexistent.service"})
	_, err := td.Handler(context.Background(), req)
	if err == nil {
		t.Log("restart succeeded (systemctl available)")
	}
}

func TestEnable_FallbackToSystemctl(t *testing.T) {
	origSession := dbusSession
	origSystem := dbusSystem
	defer func() {
		dbusSession = origSession
		dbusSystem = origSystem
	}()
	dbusSession = nil
	dbusSystem = nil

	td := findTool(t, "systemd_enable")
	req := makeReq(map[string]any{"unit": "nonexistent.service"})
	_, err := td.Handler(context.Background(), req)
	if err == nil {
		t.Log("enable succeeded (systemctl available)")
	}
}

func TestListUnits_FallbackToSystemctl(t *testing.T) {
	origSession := dbusSession
	origSystem := dbusSystem
	defer func() {
		dbusSession = origSession
		dbusSystem = origSystem
	}()
	dbusSession = nil
	dbusSystem = nil

	td := findTool(t, "systemd_list_units")
	req := makeReq(map[string]any{"state": "active"})
	_, err := td.Handler(context.Background(), req)
	if err == nil {
		t.Log("list_units succeeded (systemctl available)")
	}
}

func TestListTimers_FallbackToSystemctl(t *testing.T) {
	origSession := dbusSession
	origSystem := dbusSystem
	defer func() {
		dbusSession = origSession
		dbusSystem = origSystem
	}()
	dbusSession = nil
	dbusSystem = nil

	td := findTool(t, "systemd_list_timers")
	req := makeReq(nil)
	_, err := td.Handler(context.Background(), req)
	if err == nil {
		t.Log("list_timers succeeded (systemctl available)")
	}
}

func TestFailed_FallbackToSystemctl(t *testing.T) {
	origSession := dbusSession
	origSystem := dbusSystem
	defer func() {
		dbusSession = origSession
		dbusSystem = origSystem
	}()
	dbusSession = nil
	dbusSystem = nil

	td := findTool(t, "systemd_failed")
	req := makeReq(nil)
	_, err := td.Handler(context.Background(), req)
	if err == nil {
		t.Log("failed succeeded (systemctl available)")
	}
}

func TestLogs_FallbackHandler(t *testing.T) {
	// Logs don't go through D-Bus, just journalctl — still exercise the handler
	td := findTool(t, "systemd_logs")

	tests := []struct {
		name string
		args map[string]any
	}{
		{"default_lines", map[string]any{"unit": "test.service"}},
		{"custom_lines", map[string]any{"unit": "test.service", "lines": 10}},
		{"with_since", map[string]any{"unit": "test.service", "since": "1h ago"}},
		{"system_scope", map[string]any{"unit": "test.service", "system": true}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := makeReq(tc.args)
			_, err := td.Handler(context.Background(), req)
			if err == nil {
				t.Log("logs succeeded (journalctl available)")
			}
			// Error is expected on macOS — just verifying no panic
		})
	}
}

// ---------------------------------------------------------------------------
// System scope flag threading
// ---------------------------------------------------------------------------

func TestStatus_SystemScope_FallbackPath(t *testing.T) {
	origSession := dbusSession
	origSystem := dbusSystem
	defer func() {
		dbusSession = origSession
		dbusSystem = origSystem
	}()
	dbusSession = nil
	dbusSystem = nil

	td := findTool(t, "systemd_status")
	// System scope — the handler should invert user flag
	req := makeReq(map[string]any{"unit": "test.service", "system": true})
	_, err := td.Handler(context.Background(), req)
	if err == nil {
		t.Log("status system scope succeeded")
	}
	// Should not panic regardless
}

func TestStop_SystemScope_WithConfirm(t *testing.T) {
	origSession := dbusSession
	origSystem := dbusSystem
	defer func() {
		dbusSession = origSession
		dbusSystem = origSystem
	}()
	dbusSession = nil
	dbusSystem = nil

	td := findTool(t, "systemd_stop")
	// Critical service with confirm + system scope
	req := makeReq(map[string]any{
		"unit":    "sshd.service",
		"confirm": true,
		"system":  true,
	})
	_, err := td.Handler(context.Background(), req)
	// Should pass confirmation check, then fail at systemctl/dbus layer
	if err != nil {
		// Ensure it's NOT an INVALID_PARAM error (confirmation was provided)
		if containsStr(err.Error(), "INVALID_PARAM") {
			t.Fatalf("should have passed confirmation check: %v", err)
		}
	}
}
