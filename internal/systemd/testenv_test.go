package systemd

import (
	"os"
	"os/exec"
	"testing"
)

const liveEnvVar = "SYSTEMD_MCP_LIVE"

func requireLiveIntegration(t *testing.T) {
	t.Helper()
	if os.Getenv(liveEnvVar) != "1" {
		t.Skipf("set %s=1 to run live systemd integration tests", liveEnvVar)
	}
}

func requireSystemctlBinary(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("systemctl"); err != nil {
		t.Skip("systemctl not available")
	}
}

func requireUserManagerViaSystemctl(t *testing.T) {
	t.Helper()
	requireLiveIntegration(t)
	requireSystemctlBinary(t)
	if err := probeSystemctl(true); err != nil {
		t.Skipf("user manager unavailable via systemctl: %v", err)
	}
}

func requireSystemManagerViaSystemctl(t *testing.T) {
	t.Helper()
	requireLiveIntegration(t)
	requireSystemctlBinary(t)
	if err := probeSystemctl(false); err != nil {
		t.Skipf("system manager unavailable via systemctl: %v", err)
	}
}

func requireJournalctlScope(t *testing.T, system bool) {
	t.Helper()
	requireLiveIntegration(t)
	if err := probeJournalctl(!system); err != nil {
		t.Skipf("journalctl unavailable for %s scope: %v", scopeName(system), err)
	}
}

func requireSessionDBus(t *testing.T) *SystemdDBus {
	t.Helper()
	requireLiveIntegration(t)
	sdb, err := NewSystemdDBus(false)
	if err != nil {
		t.Skipf("D-Bus session bus not available: %v", err)
	}
	return sdb
}

func requireUserManagerViaDBus(t *testing.T) *SystemdDBus {
	t.Helper()
	sdb := requireSessionDBus(t)
	if err := probeDBusManager(sdb); err != nil {
		_ = sdb.Close()
		t.Skipf("user manager unavailable via D-Bus: %v", err)
	}
	return sdb
}

func requireSystemManagerViaDBus(t *testing.T) *SystemdDBus {
	t.Helper()
	requireLiveIntegration(t)
	sdb, err := NewSystemdDBus(true)
	if err != nil {
		t.Skipf("D-Bus system bus not available: %v", err)
	}
	if err := probeDBusManager(sdb); err != nil {
		_ = sdb.Close()
		t.Skipf("system manager unavailable via D-Bus: %v", err)
	}
	return sdb
}

func withInjectedDBus(t *testing.T, session *SystemdDBus, system *SystemdDBus, fn func()) {
	t.Helper()
	origSession := dbusSession
	origSystem := dbusSystem
	dbusSession = session
	dbusSystem = system
	defer func() {
		dbusSession = origSession
		dbusSystem = origSystem
		if session != nil {
			_ = session.Close()
		}
		if system != nil {
			_ = system.Close()
		}
	}()
	fn()
}
