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

func runCommandWithTimeout(name string, args ...string) (string, error) {
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
	case <-time.After(20 * time.Second):
		return strings.TrimSpace(output.String()), errTimedOut
	}
}

func killADBDaemon(androidHome string) error {
	_, err := runCommandWithTimeout(filepath.Join(androidHome, "platform-tools/adb"), "kill-server")
	return err
}

func isDeviceBooted(androidHome, serial string) (bool, error) {
	formatErr := func(out string, err error) error {
		if err == errTimedOut {
			return err
		}
		return fmt.Errorf("%s - %s", out, err)
	}

	dev, err := runCommandWithTimeout(filepath.Join(androidHome, "platform-tools/adb"), "-s", serial, "shell", "getprop dev.bootcomplete")
	if err != nil {
		return false, formatErr(dev, err)
	}

	sys, err := runCommandWithTimeout(filepath.Join(androidHome, "platform-tools/adb"), "-s", serial, "shell", "getprop sys.boot_completed")
	if err != nil {
		return false, formatErr(sys, err)
	}

	init, err := runCommandWithTimeout(filepath.Join(androidHome, "platform-tools/adb"), "-s", serial, "shell", "getprop init.svc.bootanim")
	if err != nil {
		return false, formatErr(init, err)
	}

	return (dev == "1" && sys == "1" && init == "stopped"), nil
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

	emulatorBootDone := false
	startTime := time.Now()

	for !emulatorBootDone {
		log.Printf("> Checking if device booted...")
		if emulatorBootDone, err = isDeviceBooted(inputs.AndroidHome, inputs.EmulatorSerial); err != nil {
			if strings.Contains(err.Error(), "daemon not running; starting now at") {
				log.Warnf("adb daemon being restarted")
				log.Printf(err.Error())
				continue
			} else if err != errTimedOut {
				failf("Failed to check emulator boot status, error: %s", err)
			}

			log.Warnf("Running command timed out, retry...")
			if err := killADBDaemon(inputs.AndroidHome); err != nil {
				if err != errTimedOut {
					failf("unable to kill ADB daemon, error: %s", err)
				}
				log.Warnf("killing ADB daemon timed out")
			}
		} else if emulatorBootDone {
			break
		}

		if time.Since(startTime) >= time.Duration(timeout)*time.Second {
			failf("Waiting for emulator boot timed out after %d seconds", timeout)
		}

		time.Sleep(5 * time.Second)
	}

	if err := adb.UnlockDevice(inputs.EmulatorSerial); err != nil {
		failf("UnlockDevice command failed, error: %s", err)
	}

	log.Donef("> Device booted")
}
