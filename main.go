package main

import (
	"os"

	"fmt"

	"time"

	"strconv"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-steplib/steps-start-android-emulator/tools"
)

// ConfigsModel ...
type ConfigsModel struct {
	emulatorSerial string
	bootTimeout    int64
	AndroidHome    string
}

// -----------------------
// --- Functions
// -----------------------

func createConfigsModelFromEnvs() ConfigsModel {

	timeout := int64(180)

	if inputTimeout, err := strconv.ParseInt(os.Getenv("boot_timeout"), 10, 64); err == nil {
		timeout = inputTimeout
	}

	return ConfigsModel{
		emulatorSerial: os.Getenv("emulator_serial"),
		bootTimeout:    timeout,
		AndroidHome:    os.Getenv("android_home"),
	}
}

func (configs ConfigsModel) print() {
	log.Infof("Configs:")

	log.Printf("- emulatorSerial: %s", configs.emulatorSerial)
	log.Printf("- bootTimeout: %d", configs.bootTimeout)
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

	fmt.Println()

	config := createConfigsModelFromEnvs()

	config.print()

	fmt.Println()

	log.Infof("Waiting for emulator boot")

	adb, err := tools.NewADB(config.AndroidHome)
	if err != nil {
		failf("Failed to create adb model, error: %s", err)
	}

	emulatorBootDone := false
	elapsedTime := int64(0)

	for !emulatorBootDone {
		if emulatorBootDone, err = adb.IsDeviceBooted(config.emulatorSerial); err != nil {
			failf("Failed to check emulator boot status, error: %s", err)
		}

		if !emulatorBootDone {
			log.Printf("> Checking if device booted...")
			time.Sleep(5 * time.Second)
			elapsedTime += 5
		}
		if elapsedTime >= config.bootTimeout {
			failf("Waiting for emulator boot timed out after %d seconds", config.bootTimeout)
		}
	}

	if err := adb.UnlockDevice(config.emulatorSerial); err != nil {
		failf("UnlockDevice command failed, error: %s", err)
	}

	log.Donef("> Device booted")
}
