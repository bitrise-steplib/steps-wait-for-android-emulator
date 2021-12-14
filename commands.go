package main

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/bitrise-io/go-utils/log"
)

const commandTimeout = 45 * time.Second

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
		out := strings.TrimSpace(output.String())
		if err != nil {
			log.TPrintf("Failed with output: %s, error: %s", out, err)
		} else {
			log.TPrintf("Finished with output: %s", out)
		}
		return out, err
	case <-clock.After(commandTimeout):
		out := strings.TrimSpace(output.String())
		log.TPrintf("Timeout with output: %s", out)
		return out, errTimedOut
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
