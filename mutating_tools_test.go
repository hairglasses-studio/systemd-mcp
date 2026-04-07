package main

import (
	"context"
	"testing"

	"github.com/hairglasses-studio/mcpkit/handler"
	"github.com/hairglasses-studio/mcpkit/registry"
)

// ---------------------------------------------------------------------------
// requireConfirmation guard
// ---------------------------------------------------------------------------

func TestRequireConfirmation_CriticalPrefixes(t *testing.T) {
	tests := []struct {
		unit    string
		confirm bool
		action  string
		wantErr bool
	}{
		// Critical services without confirm → error
		{"sshd.service", false, "stopp", true},
		{"NetworkManager.service", false, "stopp", true},
		{"systemd-logind.service", false, "stopp", true},
		{"dbus.service", false, "disabl", true},
		{"polkit.service", false, "disabl", true},
		// Critical services with confirm → ok
		{"sshd.service", true, "stopp", false},
		{"NetworkManager.service", true, "stopp", false},
		{"systemd-logind.service", true, "stopp", false},
		{"dbus.service", true, "disabl", false},
		{"polkit.service", true, "disabl", false},
		// Non-critical → ok regardless
		{"my-app.service", false, "stopp", false},
		{"nginx.service", false, "stopp", false},
		{"my-app.service", true, "stopp", false},
	}

	for _, tc := range tests {
		err := requireConfirmation(tc.unit, tc.confirm, tc.action)
		if tc.wantErr && err == nil {
			t.Errorf("requireConfirmation(%q, %v, %q) = nil, want error", tc.unit, tc.confirm, tc.action)
		}
		if !tc.wantErr && err != nil {
			t.Errorf("requireConfirmation(%q, %v, %q) = %v, want nil", tc.unit, tc.confirm, tc.action, err)
		}
		if tc.wantErr && err != nil {
			assertContains(t, err.Error(), string(handler.ErrInvalidParam))
		}
	}
}

func TestRequireConfirmation_ErrorMessage(t *testing.T) {
	err := requireConfirmation("sshd.service", false, "stopp")
	if err == nil {
		t.Fatal("expected error")
	}
	assertContains(t, err.Error(), "stopping critical service")
	assertContains(t, err.Error(), "sshd.service")
	assertContains(t, err.Error(), "confirm: true")
}

// ---------------------------------------------------------------------------
// Mutating tool handler validation (stop/disable confirmation guards)
// ---------------------------------------------------------------------------

func TestStop_RequiresConfirmation(t *testing.T) {
	td := findTool(t, "systemd_stop")

	// Attempt to stop sshd without confirm — should be rejected immediately
	// (before attempting any systemctl call)
	req := makeReq(map[string]any{"unit": "sshd.service"})
	result, err := td.Handler(context.Background(), req)
	if err != nil {
		assertContains(t, err.Error(), "INVALID_PARAM")
		return
	}
	if result == nil || !result.IsError {
		t.Fatal("expected error when stopping critical service without confirm")
	}
	assertContains(t, extractText(t, result), "INVALID_PARAM")
}

func TestDisable_RequiresConfirmation(t *testing.T) {
	td := findTool(t, "systemd_disable")

	req := makeReq(map[string]any{"unit": "dbus.service"})
	result, err := td.Handler(context.Background(), req)
	if err != nil {
		assertContains(t, err.Error(), "INVALID_PARAM")
		return
	}
	if result == nil || !result.IsError {
		t.Fatal("expected error when disabling critical service without confirm")
	}
	assertContains(t, extractText(t, result), "INVALID_PARAM")
}

func TestStop_NonCriticalProceedsToBackend(t *testing.T) {
	// A non-critical service without confirm should NOT trigger the guard,
	// but will fail at the systemctl/dbus layer (which is expected on macOS).
	td := findTool(t, "systemd_stop")
	req := makeReq(map[string]any{"unit": "my-test-app.service"})
	result, err := td.Handler(context.Background(), req)
	// We accept either:
	// - err != nil (systemctl/dbus not available) — but NOT with INVALID_PARAM
	// - result.IsError (same)
	if err != nil {
		if containsStr(err.Error(), "INVALID_PARAM") {
			t.Fatalf("non-critical service should not require confirmation, got: %v", err)
		}
		return // Expected: backend unavailable
	}
	if result != nil && result.IsError {
		text := extractText(t, result)
		if containsStr(text, "INVALID_PARAM") {
			t.Fatalf("non-critical service should not require confirmation, got: %s", text)
		}
	}
}

// ---------------------------------------------------------------------------
// Mutating tool metadata checks
// ---------------------------------------------------------------------------

func TestMutatingToolMetadata(t *testing.T) {
	m := &SystemdModule{}
	toolMap := make(map[string]registry.ToolDefinition)
	for _, td := range m.Tools() {
		toolMap[td.Tool.Name] = td
	}

	// Read-only tools
	readOnly := []string{"systemd_status", "systemd_logs", "systemd_list_units",
		"systemd_list_timers", "systemd_failed"}
	for _, name := range readOnly {
		td, ok := toolMap[name]
		if !ok {
			t.Errorf("tool %q not found", name)
			continue
		}
		if td.IsWrite {
			t.Errorf("%s: expected IsWrite=false", name)
		}
	}

	// Write tools (moderate complexity)
	moderate := []string{"systemd_start", "systemd_restart", "systemd_enable"}
	for _, name := range moderate {
		td, ok := toolMap[name]
		if !ok {
			t.Errorf("tool %q not found", name)
			continue
		}
		if !td.IsWrite {
			t.Errorf("%s: expected IsWrite=true", name)
		}
		if td.Complexity != registry.ComplexityModerate {
			t.Errorf("%s: expected ComplexityModerate, got %q", name, td.Complexity)
		}
	}

	// Destructive tools (complex)
	destructive := []string{"systemd_stop", "systemd_disable"}
	for _, name := range destructive {
		td, ok := toolMap[name]
		if !ok {
			t.Errorf("tool %q not found", name)
			continue
		}
		if !td.IsWrite {
			t.Errorf("%s: expected IsWrite=true", name)
		}
		if td.Complexity != registry.ComplexityComplex {
			t.Errorf("%s: expected ComplexityComplex, got %q", name, td.Complexity)
		}
	}
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
