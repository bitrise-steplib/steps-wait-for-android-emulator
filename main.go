package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/bitrise-io/go-android/adbmanager"
	"github.com/bitrise-io/go-android/sdk"
	"github.com/bitrise-io/go-steputils/stepconf"
	"github.com/bitrise-io/go-utils/log"
)

var errTimedOut = errors.New("running command timed out")

// Inputs ...
type Inputs struct {
	EmulatorSerial string `env:"emulator_serial,required"`
	BootTimeout    string `env:"boot_timeout,required"`
	AndroidHome    string `env:"android_home,dir"`
}

func failf(format string, v ...interface{}) {
	log.Errorf(format, v...)
	os.Exit(1)
}

// CmdRunner ...
type CmdRunner interface {
	RunCommandWithTimeout(name string, args []string) (string, error)
}

// DefaultCmdRunner ...
type DefaultCmdRunner struct {
}

// RunCommandWithTimeout ...
func (r DefaultCmdRunner) RunCommandWithTimeout(name string, args []string) (string, error) {
	return runCommandWithTimeout(name, args)
}

var cmdRunner CmdRunner = DefaultCmdRunner{}

func runCommandWithTimeout(name string, args []string) (string, error) {
	cmd := exec.Command(name, args...)

	var output bytes.Buffer
	cmd.Stderr = &output
	cmd.Stdout = &output

	if err := cmd.Start(); err != nil {
		return strings.TrimSpace(output.String()), err
	}

	done := make(chan error)

	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		return strings.TrimSpace(output.String()), err
	case <-clock.After(20 * time.Second):
		return strings.TrimSpace(output.String()), errTimedOut
	}
}

func killADBDaemon(androidHome string) error {
	name, args := adbCommand(androidHome, "", "kill-server")
	_, err := cmdRunner.RunCommandWithTimeout(name, args)
	return err
}

func adbCommand(androidHome, serial, cmd string) (string, []string) {
	name := filepath.Join(androidHome, "platform-tools/adb")
	args := []string{}
	if serial != "" {
		args = append(args, "-s", serial)
	}
	args = append(args, cmd)

	return name, args
}

func adbShellCommand(androidHome, serial, shellCmd string) (string, []string) {
	name, args := adbCommand(androidHome, serial, "shell")
	args = append(args, shellCmd)

	return name, args
}

func isDeviceBooted(androidHome, serial string) (bool, error) {
	formatErr := func(out string, err error) error {
		if err == errTimedOut {
			return err
		}
		return fmt.Errorf("%s - %s", out, err)
	}

	dev, err := cmdRunner.RunCommandWithTimeout(adbShellCommand(androidHome, serial, "getprop dev.bootcomplete"))
	if err != nil {
		return false, formatErr(dev, err)
	}

	sys, err := cmdRunner.RunCommandWithTimeout(adbShellCommand(androidHome, serial, "getprop sys.boot_completed"))
	if err != nil {
		return false, formatErr(sys, err)
	}

	init, err := cmdRunner.RunCommandWithTimeout(adbShellCommand(androidHome, serial, "getprop init.svc.bootanim"))
	if err != nil {
		return false, formatErr(init, err)
	}

	return (dev == "1" && sys == "1" && init == "stopped"), nil
}

// Clock ...
type Clock interface {
	Now() time.Time
	Since(t time.Time) time.Duration
	Sleep(d time.Duration)
	After(d time.Duration) <-chan time.Time
}

// DefaultClock ...
type DefaultClock struct{}

// Now ...
func (c DefaultClock) Now() time.Time {
	return time.Now()
}

// Since ...
func (c DefaultClock) Since(t time.Time) time.Duration {
	return time.Since(t)
}

// Sleep ...
func (c DefaultClock) Sleep(d time.Duration) {
	time.Sleep(d)
}

// After ...
func (c DefaultClock) After(d time.Duration) <-chan time.Time {
	return time.After(d)
}

var clock Clock = DefaultClock{}

func checkEmulatorBootState(androidHome, emulatorSerial string, timeout time.Duration) error {
	startTime := clock.Now()

	for {
		log.Printf("> Checking if device booted...")

		booted, err := isDeviceBooted(androidHome, emulatorSerial)
		if err != nil {
			switch {
			case strings.Contains(err.Error(), "daemon not running; starting now at"):
				log.Warnf("adb daemon being restarted")
				log.Printf(err.Error())
			case err == errTimedOut:
				log.Warnf("Running command timed out, retry...")
				if err := killADBDaemon(androidHome); err != nil {
					if err != errTimedOut {
						return fmt.Errorf("unable to kill ADB daemon, error: %s", err)
					}
					log.Warnf("killing ADB daemon timed out")
				}
			case err != nil:
				return fmt.Errorf("failed to check emulator boot status, error: %s", err)
			}
		}

		if booted {
			break
		}

		if clock.Since(startTime) >= timeout {
			return fmt.Errorf("waiting for emulator boot timed out after %d seconds", timeout)
		}

		clock.Sleep(5 * time.Second)
	}

	return nil
}

func main() {
	var inputs Inputs
	if err := stepconf.Parse(&inputs); err != nil {
		failf("Issue with inputs: %s", err)
	}

	stepconf.Print(inputs)

	fmt.Println()
	log.Infof("Waiting for emulator boot")

	sdk, err := sdk.New(inputs.AndroidHome)
	if err != nil {
		failf("Failed to create sdk, error: %s", err)
	}

	adb, err := adbmanager.New(sdk)
	if err != nil {
		failf("Failed to create adb model, error: %s", err)
	}

	timeout, err := strconv.ParseInt(inputs.BootTimeout, 10, 64)
	if err != nil {
		failf("Failed to parse BootTimeout parameter, error: %s", err)
	}

	if err := checkEmulatorBootState(inputs.AndroidHome, inputs.EmulatorSerial, time.Duration(timeout)*time.Second); err != nil {
		failf(err.Error())
	}

	if err := adb.UnlockDevice(inputs.EmulatorSerial); err != nil {
		failf("UnlockDevice command failed, error: %s", err)
	}

	log.Donef("> Device booted")
}
