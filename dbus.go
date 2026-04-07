package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/godbus/dbus/v5"
)

// SystemdDBus wraps a D-Bus connection to systemd's Manager interface.
type SystemdDBus struct {
	conn     *dbus.Conn
	isSystem bool
}

// NewSystemdDBus creates a new D-Bus connection to systemd.
// If system is true, connects to the system bus; otherwise the session bus.
func NewSystemdDBus(system bool) (*SystemdDBus, error) {
	var conn *dbus.Conn
	var err error
	if system {
		conn, err = dbus.ConnectSystemBus()
	} else {
		conn, err = dbus.ConnectSessionBus()
	}
	if err != nil {
		return nil, fmt.Errorf("dbus connect: %w", err)
	}
	return &SystemdDBus{conn: conn, isSystem: system}, nil
}

// Close closes the D-Bus connection.
func (s *SystemdDBus) Close() error {
	if s.conn != nil {
		return s.conn.Close()
	}
	return nil
}

// manager returns a BusObject for the systemd Manager interface.
func (s *SystemdDBus) manager() dbus.BusObject {
	return s.conn.Object("org.freedesktop.systemd1", "/org/freedesktop/systemd1")
}

// StartUnit starts a systemd unit. Mode is typically "replace".
func (s *SystemdDBus) StartUnit(name, mode string) error {
	var jobPath dbus.ObjectPath
	err := s.manager().Call("org.freedesktop.systemd1.Manager.StartUnit", 0, name, mode).Store(&jobPath)
	if err != nil {
		return fmt.Errorf("StartUnit %s: %w", name, err)
	}
	return nil
}

// StopUnit stops a systemd unit.
func (s *SystemdDBus) StopUnit(name, mode string) error {
	var jobPath dbus.ObjectPath
	err := s.manager().Call("org.freedesktop.systemd1.Manager.StopUnit", 0, name, mode).Store(&jobPath)
	if err != nil {
		return fmt.Errorf("StopUnit %s: %w", name, err)
	}
	return nil
}

// RestartUnit restarts a systemd unit.
func (s *SystemdDBus) RestartUnit(name, mode string) error {
	var jobPath dbus.ObjectPath
	err := s.manager().Call("org.freedesktop.systemd1.Manager.RestartUnit", 0, name, mode).Store(&jobPath)
	if err != nil {
		return fmt.Errorf("RestartUnit %s: %w", name, err)
	}
	return nil
}

// EnableUnit enables a systemd unit via EnableUnitFiles.
func (s *SystemdDBus) EnableUnit(name string) error {
	var carries bool
	var changes [][]any
	err := s.manager().Call(
		"org.freedesktop.systemd1.Manager.EnableUnitFiles", 0,
		[]string{name}, // files
		false,          // runtime (persistent)
		false,          // force
	).Store(&carries, &changes)
	if err != nil {
		return fmt.Errorf("EnableUnit %s: %w", name, err)
	}
	// Reload after enable for the symlinks to take effect.
	return s.daemon("Reload")
}

// DisableUnit disables a systemd unit via DisableUnitFiles.
func (s *SystemdDBus) DisableUnit(name string) error {
	var changes [][]any
	err := s.manager().Call(
		"org.freedesktop.systemd1.Manager.DisableUnitFiles", 0,
		[]string{name}, // files
		false,          // runtime (persistent)
	).Store(&changes)
	if err != nil {
		return fmt.Errorf("DisableUnit %s: %w", name, err)
	}
	return s.daemon("Reload")
}

// daemon calls a no-arg Manager method like Reload or Reexecute.
func (s *SystemdDBus) daemon(method string) error {
	return s.manager().Call("org.freedesktop.systemd1.Manager."+method, 0).Err
}

// UnitStatus holds the status properties for a single unit.
type UnitStatus struct {
	ActiveState          string
	SubState             string
	Description          string
	LoadState            string
	FragmentPath         string
	ActiveEnterTimestamp uint64
	MainPID              uint32
	MemoryCurrent        uint64
	MemoryCurrentSet     bool
	CPUUsageNSec         uint64
	CPUUsageNSecSet      bool
}

// GetUnitStatus queries properties of a specific unit via D-Bus.
func (s *SystemdDBus) GetUnitStatus(name string) (*UnitStatus, error) {
	// GetUnit returns the object path for the unit.
	var unitPath dbus.ObjectPath
	err := s.manager().Call("org.freedesktop.systemd1.Manager.GetUnit", 0, name).Store(&unitPath)
	if err != nil {
		// If the unit is not loaded, try LoadUnit which loads it transiently.
		err2 := s.manager().Call("org.freedesktop.systemd1.Manager.LoadUnit", 0, name).Store(&unitPath)
		if err2 != nil {
			return nil, fmt.Errorf("GetUnit %s: %w", name, err)
		}
	}

	unit := s.conn.Object("org.freedesktop.systemd1", unitPath)
	propIface := "org.freedesktop.systemd1.Unit"
	svcIface := "org.freedesktop.systemd1.Service"

	getProp := func(iface, prop string) (dbus.Variant, error) {
		v, err := unit.GetProperty(iface + "." + prop)
		if err != nil {
			return dbus.Variant{}, err
		}
		return v, nil
	}

	getStr := func(iface, prop string) string {
		v, err := getProp(iface, prop)
		if err != nil {
			return ""
		}
		s, ok := v.Value().(string)
		if !ok {
			return ""
		}
		return s
	}

	getU64 := func(iface, prop string) (uint64, bool) {
		v, err := getProp(iface, prop)
		if err != nil {
			return 0, false
		}
		switch val := v.Value().(type) {
		case uint64:
			if val == ^uint64(0) { // [not set] sentinel
				return 0, false
			}
			return val, true
		default:
			return 0, false
		}
	}

	getU32 := func(iface, prop string) uint32 {
		v, err := getProp(iface, prop)
		if err != nil {
			return 0
		}
		val, ok := v.Value().(uint32)
		if !ok {
			return 0
		}
		return val
	}

	us := &UnitStatus{
		ActiveState:  getStr(propIface, "ActiveState"),
		SubState:     getStr(propIface, "SubState"),
		Description:  getStr(propIface, "Description"),
		LoadState:    getStr(propIface, "LoadState"),
		FragmentPath: getStr(propIface, "FragmentPath"),
	}

	ts, ok := getU64(propIface, "ActiveEnterTimestamp")
	if ok {
		us.ActiveEnterTimestamp = ts
	}

	// Service-specific properties (only available for .service units).
	us.MainPID = getU32(svcIface, "MainPID")
	mem, memOk := getU64(svcIface, "MemoryCurrent")
	us.MemoryCurrent = mem
	us.MemoryCurrentSet = memOk
	cpu, cpuOk := getU64(svcIface, "CPUUsageNSec")
	us.CPUUsageNSec = cpu
	us.CPUUsageNSecSet = cpuOk

	return us, nil
}

// UnitInfo represents a unit entry from ListUnits.
type UnitInfo struct {
	Unit        string `json:"unit"`
	Description string `json:"description"`
	LoadState   string `json:"load"`
	ActiveState string `json:"active"`
	SubState    string `json:"sub"`
	Following   string `json:"following,omitempty"`
}

// ListUnits returns all loaded units via D-Bus ListUnits method.
func (s *SystemdDBus) ListUnits() ([]UnitInfo, error) {
	type rawUnit struct {
		Name        string
		Description string
		LoadState   string
		ActiveState string
		SubState    string
		Following   string
		UnitPath    dbus.ObjectPath
		JobID       uint32
		JobType     string
		JobPath     dbus.ObjectPath
	}

	var raw []rawUnit
	err := s.manager().Call("org.freedesktop.systemd1.Manager.ListUnits", 0).Store(&raw)
	if err != nil {
		return nil, fmt.Errorf("ListUnits: %w", err)
	}

	units := make([]UnitInfo, len(raw))
	for i, r := range raw {
		units[i] = UnitInfo{
			Unit:        r.Name,
			Description: r.Description,
			LoadState:   r.LoadState,
			ActiveState: r.ActiveState,
			SubState:    r.SubState,
			Following:   r.Following,
		}
	}
	return units, nil
}

// ListUnitsFiltered returns units filtered by state via D-Bus.
func (s *SystemdDBus) ListUnitsFiltered(states []string) ([]UnitInfo, error) {
	type rawUnit struct {
		Name        string
		Description string
		LoadState   string
		ActiveState string
		SubState    string
		Following   string
		UnitPath    dbus.ObjectPath
		JobID       uint32
		JobType     string
		JobPath     dbus.ObjectPath
	}

	var raw []rawUnit
	err := s.manager().Call("org.freedesktop.systemd1.Manager.ListUnitsFiltered", 0, states).Store(&raw)
	if err != nil {
		return nil, fmt.Errorf("ListUnitsFiltered: %w", err)
	}

	units := make([]UnitInfo, len(raw))
	for i, r := range raw {
		units[i] = UnitInfo{
			Unit:        r.Name,
			Description: r.Description,
			LoadState:   r.LoadState,
			ActiveState: r.ActiveState,
			SubState:    r.SubState,
			Following:   r.Following,
		}
	}
	return units, nil
}

// TimerInfo represents a timer from ListTimers.
type TimerInfo struct {
	Unit            string `json:"unit"`
	NextElapseUSec  string `json:"next,omitempty"`
	LastTriggerUSec string `json:"last,omitempty"`
	ActivatesUnit   string `json:"activates,omitempty"`
}

// ListTimers returns active timers via the D-Bus ListTimers method.
func (s *SystemdDBus) ListTimers() ([]TimerInfo, error) {
	// ListTimers is not in the base Manager interface on all versions.
	// We use the property approach: list units filtered to timers, then
	// read timer-specific properties.
	units, err := s.ListUnits()
	if err != nil {
		return nil, err
	}

	var timers []TimerInfo
	for _, u := range units {
		if !strings.HasSuffix(u.Unit, ".timer") {
			continue
		}
		if u.ActiveState != "active" {
			continue
		}
		ti := TimerInfo{
			Unit: u.Unit,
		}

		// Try to read timer properties.
		var unitPath dbus.ObjectPath
		if err := s.manager().Call("org.freedesktop.systemd1.Manager.GetUnit", 0, u.Unit).Store(&unitPath); err == nil {
			timerObj := s.conn.Object("org.freedesktop.systemd1", unitPath)
			timerIface := "org.freedesktop.systemd1.Timer"

			if v, err := timerObj.GetProperty(timerIface + ".NextElapseUSecRealtime"); err == nil {
				if usec, ok := v.Value().(uint64); ok && usec > 0 {
					ti.NextElapseUSec = strconv.FormatUint(usec, 10)
				}
			}
			if v, err := timerObj.GetProperty(timerIface + ".LastTriggerUSec"); err == nil {
				if usec, ok := v.Value().(uint64); ok && usec > 0 {
					ti.LastTriggerUSec = strconv.FormatUint(usec, 10)
				}
			}
		}

		// The unit a timer activates is the same name with .service suffix.
		ti.ActivatesUnit = strings.TrimSuffix(u.Unit, ".timer") + ".service"
		timers = append(timers, ti)
	}

	return timers, nil
}

// GetFailedUnits returns units in the "failed" state.
func (s *SystemdDBus) GetFailedUnits() ([]UnitInfo, error) {
	return s.ListUnitsFiltered([]string{"failed"})
}

// ---------------------------------------------------------------------------
// D-Bus connection pool (session + system)
// ---------------------------------------------------------------------------

var (
	dbusSession *SystemdDBus
	dbusSystem  *SystemdDBus
)

// initDBus initializes the D-Bus connections. Errors are non-fatal;
// the nil connections signal fallback to systemctl.
func initDBus() {
	dbusSession, _ = NewSystemdDBus(false)
	dbusSystem, _ = NewSystemdDBus(true)
}

// getDBus returns the appropriate D-Bus connection for the given scope.
// Returns nil if D-Bus is unavailable for that scope.
func getDBus(system bool) *SystemdDBus {
	if system {
		return dbusSystem
	}
	return dbusSession
}

// ---------------------------------------------------------------------------
// D-Bus result formatting helpers
// ---------------------------------------------------------------------------

// unitStatusToOutput converts a D-Bus UnitStatus to the handler's StatusOutput,
// preserving the same field semantics as the systemctl --show parser.
func unitStatusToOutput(name string, us *UnitStatus) StatusOutput {
	out := StatusOutput{
		Unit:         name,
		ActiveState:  us.ActiveState,
		SubState:     us.SubState,
		Description:  us.Description,
		LoadState:    us.LoadState,
		FragmentPath: us.FragmentPath,
		MainPID:      int(us.MainPID),
	}
	if us.ActiveEnterTimestamp > 0 {
		out.ActiveEnterTimestamp = strconv.FormatUint(us.ActiveEnterTimestamp, 10)
	}
	if us.MemoryCurrentSet {
		out.MemoryCurrent = strconv.FormatUint(us.MemoryCurrent, 10)
	}
	if us.CPUUsageNSecSet {
		out.CPUUsageNSec = strconv.FormatUint(us.CPUUsageNSec, 10)
	}
	return out
}

// unitsToJSON marshals a []UnitInfo slice to json.RawMessage.
func unitsToJSON(units []UnitInfo) (json.RawMessage, error) {
	b, err := json.Marshal(units)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(b), nil
}

// timersToJSON marshals a []TimerInfo slice to json.RawMessage.
func timersToJSON(timers []TimerInfo) (json.RawMessage, error) {
	if timers == nil {
		timers = []TimerInfo{}
	}
	b, err := json.Marshal(timers)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(b), nil
}
