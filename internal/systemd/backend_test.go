package systemd

import (
	"errors"
	"testing"

	"github.com/hairglasses-studio/mcpkit/handler"
)

type fakeCommandResponse struct {
	stdout string
	stderr string
	err    error
}

type fakeCommandRunner struct {
	responses map[string]fakeCommandResponse
}

func (f fakeCommandRunner) Run(name string, args ...string) (string, string, error) {
	key := name
	for _, arg := range args {
		key += " " + arg
	}
	resp, ok := f.responses[key]
	if !ok {
		return "", "", errors.New("unexpected command: " + key)
	}
	return resp.stdout, resp.stderr, resp.err
}

func TestProbeScopeCapabilities_UserManagerUnavailable(t *testing.T) {
	restore := setCommandRunnerForTest(fakeCommandRunner{
		responses: map[string]fakeCommandResponse{
			"systemctl --user show-environment": {
				stderr: "Failed to connect to user scope bus via local transport",
				err:    errors.New("exit status 1"),
			},
			"journalctl --user -n 1 --no-pager": {
				stdout: "ok",
			},
		},
	})
	defer restore()

	origSession := dbusSession
	origSystem := dbusSystem
	dbusSession = nil
	dbusSystem = nil
	defer func() {
		dbusSession = origSession
		dbusSystem = origSystem
	}()

	caps := probeScopeCapabilities(false)
	if !caps.SystemctlAvailable {
		t.Fatal("expected systemctl to be detected as installed")
	}
	if caps.SystemctlUsable {
		t.Fatal("expected user-scope systemctl to be unusable")
	}
	if !caps.JournalctlUsable {
		t.Fatal("expected user-scope journalctl to be usable")
	}
}

func TestReadUnitStatus_UserManagerUnavailableReturnsUpstream(t *testing.T) {
	restore := setCommandRunnerForTest(fakeCommandRunner{
		responses: map[string]fakeCommandResponse{
			"systemctl --user show-environment": {
				stderr: "Failed to connect to user scope bus via local transport",
				err:    errors.New("exit status 1"),
			},
		},
	})
	defer restore()

	origSession := dbusSession
	origSystem := dbusSystem
	dbusSession = nil
	dbusSystem = nil
	defer func() {
		dbusSession = origSession
		dbusSystem = origSystem
	}()

	_, err := readUnitStatus(false, "dbus.service")
	if err == nil {
		t.Fatal("expected upstream error")
	}
	if containsStr(err.Error(), string(handler.ErrNotFound)) {
		t.Fatalf("expected manager availability failure, got not-found: %v", err)
	}
	assertContains(t, err.Error(), string(handler.ErrUpstreamError))
}

func TestReadUnitStatus_SystemctlNotFoundReturnsNotFound(t *testing.T) {
	restore := setCommandRunnerForTest(fakeCommandRunner{
		responses: map[string]fakeCommandResponse{
			"systemctl --user show-environment": {
				stdout: "PATH=/usr/bin",
			},
			"systemctl --user show --property=ActiveState,SubState,Description,LoadState,FragmentPath,ActiveEnterTimestamp,MainPID,MemoryCurrent,CPUUsageNSec missing.service": {
				stdout: "Description=missing.service\nLoadState=not-found\nActiveState=inactive\nSubState=dead",
			},
		},
	})
	defer restore()

	origSession := dbusSession
	origSystem := dbusSystem
	dbusSession = nil
	dbusSystem = nil
	defer func() {
		dbusSession = origSession
		dbusSystem = origSystem
	}()

	out, err := readUnitStatus(false, "missing.service")
	if err == nil {
		t.Fatal("expected not-found error")
	}
	assertContains(t, err.Error(), string(handler.ErrNotFound))
	if out.LoadState != "not-found" {
		t.Fatalf("expected load state not-found, got %q", out.LoadState)
	}
}
