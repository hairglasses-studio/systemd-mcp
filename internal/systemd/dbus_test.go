package systemd

import (
	"encoding/json"
	"testing"
)

// ---------------------------------------------------------------------------
// unitStatusToOutput conversion
// ---------------------------------------------------------------------------

func TestUnitStatusToOutput_Full(t *testing.T) {
	us := &UnitStatus{
		ActiveState:          "active",
		SubState:             "running",
		Description:          "My Test Service",
		LoadState:            "loaded",
		FragmentPath:         "/etc/systemd/user/test.service",
		ActiveEnterTimestamp: 1700000000000000,
		MainPID:              12345,
		MemoryCurrent:        1048576,
		MemoryCurrentSet:     true,
		CPUUsageNSec:         500000000,
		CPUUsageNSecSet:      true,
	}

	out := unitStatusToOutput("test.service", us)

	if out.Unit != "test.service" {
		t.Errorf("Unit = %q, want %q", out.Unit, "test.service")
	}
	if out.ActiveState != "active" {
		t.Errorf("ActiveState = %q, want %q", out.ActiveState, "active")
	}
	if out.SubState != "running" {
		t.Errorf("SubState = %q, want %q", out.SubState, "running")
	}
	if out.Description != "My Test Service" {
		t.Errorf("Description = %q, want %q", out.Description, "My Test Service")
	}
	if out.LoadState != "loaded" {
		t.Errorf("LoadState = %q, want %q", out.LoadState, "loaded")
	}
	if out.FragmentPath != "/etc/systemd/user/test.service" {
		t.Errorf("FragmentPath = %q, want %q", out.FragmentPath, "/etc/systemd/user/test.service")
	}
	if out.MainPID != 12345 {
		t.Errorf("MainPID = %d, want %d", out.MainPID, 12345)
	}
	if out.MemoryCurrent != "1048576" {
		t.Errorf("MemoryCurrent = %q, want %q", out.MemoryCurrent, "1048576")
	}
	if out.CPUUsageNSec != "500000000" {
		t.Errorf("CPUUsageNSec = %q, want %q", out.CPUUsageNSec, "500000000")
	}
	if out.ActiveEnterTimestamp != "1700000000000000" {
		t.Errorf("ActiveEnterTimestamp = %q, want %q", out.ActiveEnterTimestamp, "1700000000000000")
	}
}

func TestUnitStatusToOutput_Minimal(t *testing.T) {
	us := &UnitStatus{
		ActiveState: "inactive",
		SubState:    "dead",
		LoadState:   "not-found",
	}

	out := unitStatusToOutput("missing.service", us)

	if out.Unit != "missing.service" {
		t.Errorf("Unit = %q, want %q", out.Unit, "missing.service")
	}
	if out.ActiveState != "inactive" {
		t.Errorf("ActiveState = %q, want %q", out.ActiveState, "inactive")
	}
	if out.LoadState != "not-found" {
		t.Errorf("LoadState = %q, want %q", out.LoadState, "not-found")
	}
	// Fields that should be zero values
	if out.MemoryCurrent != "" {
		t.Errorf("MemoryCurrent should be empty, got %q", out.MemoryCurrent)
	}
	if out.CPUUsageNSec != "" {
		t.Errorf("CPUUsageNSec should be empty, got %q", out.CPUUsageNSec)
	}
	if out.ActiveEnterTimestamp != "" {
		t.Errorf("ActiveEnterTimestamp should be empty, got %q", out.ActiveEnterTimestamp)
	}
	if out.MainPID != 0 {
		t.Errorf("MainPID should be 0, got %d", out.MainPID)
	}
}

func TestUnitStatusToOutput_ZeroTimestamp(t *testing.T) {
	us := &UnitStatus{
		ActiveState:          "active",
		SubState:             "running",
		LoadState:            "loaded",
		ActiveEnterTimestamp: 0, // zero should not be serialized
	}

	out := unitStatusToOutput("test.service", us)
	if out.ActiveEnterTimestamp != "" {
		t.Errorf("expected empty ActiveEnterTimestamp for zero value, got %q", out.ActiveEnterTimestamp)
	}
}

// ---------------------------------------------------------------------------
// unitsToJSON
// ---------------------------------------------------------------------------

func TestUnitsToJSON(t *testing.T) {
	units := []UnitInfo{
		{Unit: "a.service", Description: "Service A", LoadState: "loaded", ActiveState: "active", SubState: "running"},
		{Unit: "b.timer", Description: "Timer B", LoadState: "loaded", ActiveState: "active", SubState: "waiting"},
	}

	raw, err := unitsToJSON(units)
	if err != nil {
		t.Fatalf("unitsToJSON error: %v", err)
	}

	if !json.Valid(raw) {
		t.Fatalf("invalid JSON: %s", string(raw))
	}

	var decoded []UnitInfo
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if len(decoded) != 2 {
		t.Fatalf("expected 2 units, got %d", len(decoded))
	}
	if decoded[0].Unit != "a.service" {
		t.Errorf("first unit = %q, want %q", decoded[0].Unit, "a.service")
	}
	if decoded[1].ActiveState != "active" {
		t.Errorf("second unit active state = %q, want %q", decoded[1].ActiveState, "active")
	}
}

func TestUnitsToJSON_Empty(t *testing.T) {
	raw, err := unitsToJSON([]UnitInfo{})
	if err != nil {
		t.Fatalf("unitsToJSON error: %v", err)
	}
	if string(raw) != "[]" {
		t.Errorf("expected [], got %s", string(raw))
	}
}

// ---------------------------------------------------------------------------
// timersToJSON
// ---------------------------------------------------------------------------

func TestTimersToJSON(t *testing.T) {
	timers := []TimerInfo{
		{Unit: "backup.timer", NextElapseUSec: "1700000000", ActivatesUnit: "backup.service"},
	}

	raw, err := timersToJSON(timers)
	if err != nil {
		t.Fatalf("timersToJSON error: %v", err)
	}

	if !json.Valid(raw) {
		t.Fatalf("invalid JSON: %s", string(raw))
	}

	var decoded []TimerInfo
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if len(decoded) != 1 {
		t.Fatalf("expected 1 timer, got %d", len(decoded))
	}
	if decoded[0].Unit != "backup.timer" {
		t.Errorf("timer unit = %q, want %q", decoded[0].Unit, "backup.timer")
	}
}

func TestTimersToJSON_Nil(t *testing.T) {
	// nil slice should be converted to empty array
	raw, err := timersToJSON(nil)
	if err != nil {
		t.Fatalf("timersToJSON error: %v", err)
	}
	if string(raw) != "[]" {
		t.Errorf("expected [], got %s", string(raw))
	}
}

// ---------------------------------------------------------------------------
// getDBus returns nil when not initialized (fallback path)
// ---------------------------------------------------------------------------

func TestGetDBus_NilConnections(t *testing.T) {
	// Before initDBus is called (or on macOS), both should be nil
	// Save and restore the globals
	origSession := dbusSession
	origSystem := dbusSystem
	defer func() {
		dbusSession = origSession
		dbusSystem = origSystem
	}()

	dbusSession = nil
	dbusSystem = nil

	if got := getDBus(false); got != nil {
		t.Error("expected nil for session dbus when not initialized")
	}
	if got := getDBus(true); got != nil {
		t.Error("expected nil for system dbus when not initialized")
	}
}
