package systemd

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

type commandRunner interface {
	Run(name string, args ...string) (string, string, error)
}

type execCommandRunner struct{}

func (execCommandRunner) Run(name string, args ...string) (string, string, error) {
	cmd := exec.Command(name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()), err
}

var (
	commandRunnerMu sync.RWMutex
	commands        commandRunner = execCommandRunner{}
)

func currentCommandRunner() commandRunner {
	commandRunnerMu.RLock()
	defer commandRunnerMu.RUnlock()
	return commands
}

func setCommandRunnerForTest(runner commandRunner) func() {
	commandRunnerMu.Lock()
	previous := commands
	commands = runner
	commandRunnerMu.Unlock()

	return func() {
		commandRunnerMu.Lock()
		commands = previous
		commandRunnerMu.Unlock()
	}
}

func runCmd(name string, args ...string) (string, string, error) {
	return currentCommandRunner().Run(name, args...)
}

func systemctlArgs(user bool, args ...string) []string {
	cmdArgs := make([]string, 0, len(args)+1)
	if user {
		cmdArgs = append(cmdArgs, "--user")
	}
	cmdArgs = append(cmdArgs, args...)
	return cmdArgs
}

func journalctlArgs(user bool, args ...string) []string {
	cmdArgs := make([]string, 0, len(args)+1)
	if user {
		cmdArgs = append(cmdArgs, "--user")
	}
	cmdArgs = append(cmdArgs, args...)
	return cmdArgs
}

func runSystemctl(user bool, args ...string) (string, error) {
	cmdArgs := systemctlArgs(user, args...)
	stdout, stderr, err := runCmd("systemctl", cmdArgs...)
	if err != nil {
		return "", fmt.Errorf("systemctl %s: %s: %w", strings.Join(cmdArgs, " "), stderr, err)
	}
	return stdout, nil
}

func runJournalctl(user bool, args ...string) (string, error) {
	cmdArgs := journalctlArgs(user, args...)
	stdout, stderr, err := runCmd("journalctl", cmdArgs...)
	if err != nil {
		return "", fmt.Errorf("journalctl %s: %s: %w", strings.Join(cmdArgs, " "), stderr, err)
	}
	return stdout, nil
}

func probeSystemctl(user bool) error {
	_, err := runSystemctl(user, "show-environment")
	return err
}

func probeJournalctl(user bool) error {
	_, err := runJournalctl(user, "-n", "1", "--no-pager")
	return err
}
