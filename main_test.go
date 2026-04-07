package main

import (
	"context"
	"encoding/json"
	"os/exec"
	"testing"

	"github.com/hairglasses-studio/mcpkit/mcptest"
	"github.com/hairglasses-studio/mcpkit/registry"
)

// ---------------------------------------------------------------------------
// Registration
// ---------------------------------------------------------------------------

func TestModuleRegistration(t *testing.T) {
	m := &SystemdModule{}
	tools := m.Tools()
	if len(tools) != 10 {
		t.Fatalf("expected 10 tools, got %d", len(tools))
	}

	reg := registry.NewToolRegistry()
	reg.RegisterModule(m)
	srv := mcptest.NewServer(t, reg)

	names := srv.ToolNames()
	if len(names) != 10 {
		t.Fatalf("expected 10 registered tools, got %d", len(names))
	}

	for _, want := range []string{
		"systemd_status", "systemd_start", "systemd_stop",
		"systemd_restart", "systemd_enable", "systemd_disable",
		"systemd_logs", "systemd_list_units", "systemd_list_timers",
		"systemd_failed",
	} {
		if !srv.HasTool(want) {
			t.Errorf("missing tool: %s", want)
		}
	}
}

func TestContextRegistries(t *testing.T) {
	if got := buildSystemdResourceRegistry().ResourceCount(); got != 1 {
		t.Fatalf("expected 1 systemd resource, got %d", got)
	}
	if got := buildSystemdPromptRegistry().PromptCount(); got != 1 {
		t.Fatalf("expected 1 systemd prompt, got %d", got)
	}
}

// ---------------------------------------------------------------------------
// systemd_status
// ---------------------------------------------------------------------------

func TestStatus_KnownUnit(t *testing.T) {
	requireSystemctl(t)

	td := findTool(t, "systemd_status")
	// Query a unit that should exist in user scope
	req := makeReq(map[string]any{"unit": "dbus.service"})
	result, err := td.Handler(context.Background(), req)
	if err != nil {
		// dbus might not be a user unit; try pipewire
		req = makeReq(map[string]any{"unit": "pipewire.service"})
		result, err = td.Handler(context.Background(), req)
		if err != nil {
			t.Skipf("no known user unit available: %v", err)
		}
	}

	var out StatusOutput
	unmarshalResult(t, result, &out)
	if out.ActiveState == "" {
		t.Error("ActiveState is empty")
	}
	if out.LoadState == "" {
		t.Error("LoadState is empty")
	}
}

func TestStatus_NotFoundUnit(t *testing.T) {
	requireSystemctl(t)

	td := findTool(t, "systemd_status")
	req := makeReq(map[string]any{"unit": "nonexistent-unit-xyz-12345.service"})
	result, err := td.Handler(context.Background(), req)
	if err != nil {
		assertContains(t, err.Error(), "NOT_FOUND")
		return
	}
	// Typed handler wraps errors as IsError results
	if result != nil && result.IsError {
		return
	}
	t.Fatal("expected error for nonexistent unit")
}

// ---------------------------------------------------------------------------
// systemd_logs
// ---------------------------------------------------------------------------

func TestLogs_DefaultLines(t *testing.T) {
	requireSystemctl(t)

	td := findTool(t, "systemd_logs")
	// lines=0 should default to 50
	req := makeReq(map[string]any{"unit": "dbus.service", "lines": 0})
	result, err := td.Handler(context.Background(), req)
	if err != nil {
		t.Skipf("journalctl failed (unit may not exist): %v", err)
	}

	var out LogsOutput
	unmarshalResult(t, result, &out)
	if out.Lines != 50 {
		t.Errorf("expected default lines=50, got %d", out.Lines)
	}
}

func TestLogs_CustomLines(t *testing.T) {
	requireSystemctl(t)

	td := findTool(t, "systemd_logs")
	req := makeReq(map[string]any{"unit": "dbus.service", "lines": 5})
	result, err := td.Handler(context.Background(), req)
	if err != nil {
		t.Skipf("journalctl failed: %v", err)
	}

	var out LogsOutput
	unmarshalResult(t, result, &out)
	if out.Lines != 5 {
		t.Errorf("expected lines=5, got %d", out.Lines)
	}
}

// ---------------------------------------------------------------------------
// systemd_list_units
// ---------------------------------------------------------------------------

func TestListUnits_Default(t *testing.T) {
	requireSystemctl(t)

	td := findTool(t, "systemd_list_units")
	req := makeReq(nil)
	result, err := td.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}

	var out ListUnitsOutput
	unmarshalResult(t, result, &out)
	// Verify the Units field is valid JSON
	if !json.Valid(out.Units) {
		t.Errorf("Units is not valid JSON: %s", string(out.Units))
	}
}

func TestListUnits_StateFilter(t *testing.T) {
	requireSystemctl(t)

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
}

// ---------------------------------------------------------------------------
// systemd_list_timers
// ---------------------------------------------------------------------------

func TestListTimers(t *testing.T) {
	requireSystemctl(t)

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
}

// ---------------------------------------------------------------------------
// systemd_failed
// ---------------------------------------------------------------------------

func TestFailed(t *testing.T) {
	requireSystemctl(t)

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
}

// ---------------------------------------------------------------------------
// Scope handling
// ---------------------------------------------------------------------------

func TestStatus_SystemScope(t *testing.T) {
	requireSystemctl(t)

	td := findTool(t, "systemd_status")
	// System scope query — may succeed or fail based on permissions or unit not existing
	req := makeReq(map[string]any{"unit": "bluetooth.service", "system": true})
	result, err := td.Handler(context.Background(), req)
	if err != nil {
		t.Logf("system scope status (expected possible failure): %v", err)
		return
	}
	if result != nil && result.IsError {
		t.Log("system scope returned error result (unit may not exist on this host)")
		return
	}

	var out StatusOutput
	unmarshalResult(t, result, &out)
	if out.ActiveState == "" {
		t.Error("ActiveState is empty for system unit")
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func requireSystemctl(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("systemctl"); err != nil {
		t.Skip("systemctl not available")
	}
}

func findTool(t *testing.T, name string) registry.ToolDefinition {
	t.Helper()
	m := &SystemdModule{}
	for _, td := range m.Tools() {
		if td.Tool.Name == name {
			return td
		}
	}
	t.Fatalf("tool %q not found", name)
	return registry.ToolDefinition{}
}

func makeReq(args map[string]any) registry.CallToolRequest {
	req := registry.CallToolRequest{}
	if args == nil {
		args = map[string]any{}
	}
	req.Params.Arguments = args
	return req
}

func extractText(t *testing.T, result *registry.CallToolResult) string {
	t.Helper()
	if len(result.Content) == 0 {
		t.Fatal("result has no content")
	}
	tc, ok := result.Content[0].(registry.TextContent)
	if !ok {
		t.Fatalf("content is not TextContent, got %T", result.Content[0])
	}
	return tc.Text
}

func unmarshalResult(t *testing.T, result *registry.CallToolResult, out any) {
	t.Helper()
	text := extractText(t, result)
	if err := json.Unmarshal([]byte(text), out); err != nil {
		t.Fatalf("unmarshal error: %v; text=%s", err, text)
	}
}

func assertContains(t *testing.T, s, substr string) {
	t.Helper()
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return
		}
	}
	t.Errorf("expected %q to contain %q", s, substr)
}
