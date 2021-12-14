package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/bitrise-io/go-android/adbmanager"
	"github.com/bitrise-io/go-android/sdk"
	"github.com/bitrise-io/go-steputils/stepconf"
	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/env"
	"github.com/bitrise-io/go-utils/log"
)

// CmdRunner ...
type CmdRunner interface {
	RunCommandWithTimeout(name string, args []string) (string, error)
}

var cmdRunner CmdRunner = defaultCmdRunner{}

// Clock ...
type Clock interface {
	Now() time.Time
	Since(t time.Time) time.Duration
	Sleep(d time.Duration)
	After(d time.Duration) <-chan time.Time
}

var clock Clock = defaultClock{}

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

func waitForDeviceStateAndSYSBootComplete(androidHome, serial string) (string, error) {
	return cmdRunner.RunCommandWithTimeout(adbWaitForDeviceShellCommand(androidHome, serial, "getprop sys.boot_completed"))
}

func waitForDeviceStateAndDEVBootComplete(androidHome, serial string) (string, error) {
	return cmdRunner.RunCommandWithTimeout(adbWaitForDeviceShellCommand(androidHome, serial, "getprop dev.bootcomplete"))
}

func waitForDeviceStateAndSYSBootAnimComplete(androidHome, serial string) (string, error) {
	return cmdRunner.RunCommandWithTimeout(adbWaitForDeviceShellCommand(androidHome, serial, "getprop init.svc.bootanim"))
}

func terminateADBServer(androidHome string) error {
	name, args := adbCommand(androidHome, "", "kill-server")
	_, err := cmdRunner.RunCommandWithTimeout(name, args)
	return err
}

func handleDeviceBootStateError(err error, out, androidHome string) error {
	if err == nil {
		return nil
	}

	switch {
	case strings.Contains(err.Error(), "daemon not running; starting now at") || strings.Contains(out, "daemon not running; starting now at"):
		log.Warnf("adb daemon being restarted")
		return nil
	case err == errTimedOut:
		log.Warnf("Running command timed out, retry...")
		if err := terminateADBServer(androidHome); err != nil {
			if err != errTimedOut {
				return fmt.Errorf("unable to kill ADB daemon, error: %s", err)
			}
			log.Warnf("killing ADB daemon timed out")
		}
		return nil
	}

	return err
}

func checkEmulatorBootState(androidHome, emulatorSerial string, timeout time.Duration) error {
	startTime := clock.Now()

	log.Printf("Checking if device booted...")

	for {
		out, err := waitForDeviceStateAndSYSBootComplete(androidHome, emulatorSerial)
		if err := handleDeviceBootStateError(err, out, androidHome); err != nil {
			return err
		}

		if out == "1" {
			break
		}

		if clock.Since(startTime) >= timeout {
			return fmt.Errorf("waiting for emulator boot timed out after %d seconds", timeout)
		}

		clock.Sleep(5 * time.Second)
	}

	for {
		out, err := waitForDeviceStateAndDEVBootComplete(androidHome, emulatorSerial)
		if err := handleDeviceBootStateError(err, out, androidHome); err != nil {
			return err
		}

		if out == "1" {
			break
		}

		if clock.Since(startTime) >= timeout {
			return fmt.Errorf("waiting for emulator boot timed out after %d seconds", timeout)
		}

		clock.Sleep(5 * time.Second)
	}

	for {
		out, err := waitForDeviceStateAndSYSBootAnimComplete(androidHome, emulatorSerial)
		if err := handleDeviceBootStateError(err, out, androidHome); err != nil {
			return err
		}

		if out == "stopped" {
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
	envRepo := env.NewRepository()

	var inputs Inputs
	if err := stepconf.NewInputParser(envRepo).Parse(&inputs); err != nil {
		failf("Issue with inputs: %s", err)
	}

	stepconf.Print(inputs)

	fmt.Println()
	log.Infof("Waiting for emulator boot")

	sdk, err := sdk.New(inputs.AndroidHome)
	if err != nil {
		failf("Failed to create sdk, error: %s", err)
	}

	adb, err := adbmanager.New(sdk, command.NewFactory(envRepo))
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
