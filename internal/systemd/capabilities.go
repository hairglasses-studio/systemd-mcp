package systemd

import "strings"

type ScopeCapabilities struct {
	Scope                string `json:"scope"`
	DBusConnected        bool   `json:"dbus_connected"`
	DBusManagerReachable bool   `json:"dbus_manager_reachable"`
	DBusProbeError       string `json:"dbus_probe_error,omitempty"`
	SystemctlAvailable   bool   `json:"systemctl_available"`
	SystemctlUsable      bool   `json:"systemctl_usable"`
	SystemctlProbeError  string `json:"systemctl_probe_error,omitempty"`
	JournalctlAvailable  bool   `json:"journalctl_available"`
	JournalctlUsable     bool   `json:"journalctl_usable"`
	JournalctlProbeError string `json:"journalctl_probe_error,omitempty"`
}

type RuntimeCapabilities struct {
	User   ScopeCapabilities `json:"user"`
	System ScopeCapabilities `json:"system"`
}

func detectRuntimeCapabilities() RuntimeCapabilities {
	return RuntimeCapabilities{
		User:   probeScopeCapabilities(false),
		System: probeScopeCapabilities(true),
	}
}

func scopeName(system bool) string {
	if system {
		return "system"
	}
	return "user"
}

func probeScopeCapabilities(system bool) ScopeCapabilities {
	caps := ScopeCapabilities{Scope: scopeName(system)}

	if sdb := getDBus(system); sdb != nil {
		caps.DBusConnected = true
		if err := probeDBusManager(sdb); err != nil {
			caps.DBusProbeError = err.Error()
		} else {
			caps.DBusManagerReachable = true
		}
	} else {
		caps.DBusProbeError = "dbus connection not initialized"
	}

	user := !system
	if err := probeSystemctl(user); err != nil {
		if !isCommandMissingErr(err) {
			caps.SystemctlAvailable = true
		}
		caps.SystemctlProbeError = err.Error()
	} else {
		caps.SystemctlAvailable = true
		caps.SystemctlUsable = true
	}

	if err := probeJournalctl(user); err != nil {
		if !isCommandMissingErr(err) {
			caps.JournalctlAvailable = true
		}
		caps.JournalctlProbeError = err.Error()
	} else {
		caps.JournalctlAvailable = true
		caps.JournalctlUsable = true
	}

	return caps
}

func probeDBusManager(sdb *SystemdDBus) error {
	if _, err := sdb.ListUnits(); err != nil {
		return err
	}
	return nil
}

func isCommandMissingErr(err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "executable file not found") ||
		strings.Contains(lower, "file not found")
}
