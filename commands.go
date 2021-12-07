package main

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/bitrise-io/go-utils/log"
)

type defaultCmdRunner struct{}

// RunCommandWithTimeout ...
func (r defaultCmdRunner) RunCommandWithTimeout(name string, args []string) (string, error) {
	cmd := exec.Command(name, args...)

	var output bytes.Buffer
	cmd.Stderr = &output
	cmd.Stdout = &output

	if err := cmd.Start(); err != nil {
		return strings.TrimSpace(output.String()), err
	}

	log.TPrintf("Started: %s", strings.Join(cmd.Args, " "))

	done := make(chan error)

	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		log.TPrintf("Finished: %s", strings.TrimSpace(output.String()))
		return strings.TrimSpace(output.String()), err
	case <-clock.After(60 * time.Second):
		log.TPrintf("Finished: %s", strings.TrimSpace(output.String()))
		return strings.TrimSpace(output.String()), errTimedOut
	}
}

func adbCommand(androidHome, serial string, args ...string) (string, []string) {
	name := filepath.Join(androidHome, "platform-tools/adb")
	var cmd []string
	if serial != "" {
		cmd = append(cmd, "-s", serial)
	}
	cmd = append(cmd, args...)

	return name, cmd
}

func adbWaitForDeviceShellCommand(androidHome, serial, shellCmd string) (string, []string) {
	name, args := adbCommand(androidHome, serial, "wait-for-device", "shell")
	args = append(args, shellCmd)

	return name, args
}

func adbShellCommand(androidHome, serial, shellCmd string) (string, []string) {
	name, args := adbCommand(androidHome, serial, "shell")
	args = append(args, shellCmd)

	return name, args
}
