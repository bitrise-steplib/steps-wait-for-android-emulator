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

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-tools/go-android/adbmanager"
	"github.com/bitrise-tools/go-android/sdk"
)

var errTimedOut = errors.New("running command timed out")

// ConfigsModel ...
type ConfigsModel struct {
	EmulatorSerial string
	BootTimeout    string
	AndroidHome    string
}

// -----------------------
// --- Functions
// -----------------------

func createConfigsModelFromEnvs() ConfigsModel {
	return ConfigsModel{
		EmulatorSerial: os.Getenv("emulator_serial"),
		BootTimeout:    os.Getenv("boot_timeout"),
		AndroidHome:    os.Getenv("android_home"),
	}
}

func (configs ConfigsModel) validate() error {
	if configs.EmulatorSerial == "" {
		return errors.New("no EmulatorSerial parameter specified")
	}
	if configs.AndroidHome == "" {
		return errors.New("no AndroidHome parameter specified")
	}
	if configs.BootTimeout == "" {
		return errors.New("no BootTimeout parameter specified")
	}

	return nil
}

func (configs ConfigsModel) print() {
	log.Infof("Configs:")

	log.Printf("- emulatorSerial: %s", configs.EmulatorSerial)
	log.Printf("- bootTimeout: %s", configs.BootTimeout)
	log.Printf("- AndroidHome: %s", configs.AndroidHome)
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

// -----------------------
// --- Main
// -----------------------

func main() {
	config := createConfigsModelFromEnvs()

	fmt.Println()
	config.print()

	if err := config.validate(); err != nil {
		failf("Issue with input: %s", err)
	}

	fmt.Println()
	log.Infof("Waiting for emulator boot")

	sdk, err := sdk.New(config.AndroidHome)
	if err != nil {
		failf("Failed to create sdk, error: %s", err)
	}

	adb, err := adbmanager.New(sdk)
	if err != nil {
		failf("Failed to create adb model, error: %s", err)
	}

	timeout, err := strconv.ParseInt(config.BootTimeout, 10, 64)
	if err != nil {
		failf("Failed to parse BootTimeout parameter, error: %s", err)
	}

	emulatorBootDone := false
	startTime := time.Now()

	for !emulatorBootDone {
		log.Printf("> Checking if device booted...")
		if emulatorBootDone, err = isDeviceBooted(config.AndroidHome, config.EmulatorSerial); err != nil {
			if err != errTimedOut {
				failf("Failed to check emulator boot status, error: %s", err)
			}
			log.Warnf("Running command timed out, retry...")
			if err := killADBDaemon(config.AndroidHome); err != nil {
				if err != errTimedOut {
					failf("unable to kill ADB daemon, error: %s", err)
				}
				log.Warnf("killing ADB daemon timed out")
			}
		} else if emulatorBootDone {
			break
		}

		if time.Now().Sub(startTime) >= time.Duration(timeout)*time.Second {
			failf("Waiting for emulator boot timed out after %d seconds", timeout)
		}

		time.Sleep(5 * time.Second)
	}

	if err := adb.UnlockDevice(config.EmulatorSerial); err != nil {
		failf("UnlockDevice command failed, error: %s", err)
	}

	log.Donef("> Device booted")
}
