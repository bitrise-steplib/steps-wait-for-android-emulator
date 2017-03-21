package main

import (
	"errors"
	"os"
	"strconv"

	"fmt"

	"time"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-tools/go-android/adbmanager"
)

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

	adb, err := adbmanager.New(config.AndroidHome)
	if err != nil {
		failf("Failed to create adb model, error: %s", err)
	}

	timeout, err := strconv.ParseInt(config.BootTimeout, 10, 64)
	if err != nil {
		failf("Failed to parse BootTimeout parameter, error: %s", err)
	}

	emulatorBootDone := false
	elapsedTime := int64(0)

	for !emulatorBootDone {
		if emulatorBootDone, err = adb.IsDeviceBooted(config.EmulatorSerial); err != nil {
			failf("Failed to check emulator boot status, error: %s", err)
		} else if emulatorBootDone {
			break
		}

		if elapsedTime >= timeout {
			failf("Waiting for emulator boot timed out after %d seconds", config.BootTimeout)
		}

		log.Printf("> Checking if device booted...")
		time.Sleep(5 * time.Second)
		elapsedTime += 5
	}

	if err := adb.UnlockDevice(config.EmulatorSerial); err != nil {
		failf("UnlockDevice command failed, error: %s", err)
	}

	log.Donef("> Device booted")
}
