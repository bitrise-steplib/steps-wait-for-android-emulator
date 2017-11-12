package main

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"

	"fmt"

	"time"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/log"
)

// ConfigsModel ...
type ConfigsModel struct {
	EmulatorSerial string
	BootTimeout    string
	AndroidHome    string
}

func createConfigsModelFromEnvs() ConfigsModel {
	return ConfigsModel{
		EmulatorSerial: os.Getenv("emulator_serial"),
		BootTimeout:    os.Getenv("boot_timeout"),
		AndroidHome:    os.Getenv("android_home"),
	}
}

func (configs ConfigsModel) validate() error {
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

	timeout, err := strconv.ParseInt(config.BootTimeout, 10, 64)
	if err != nil {
		failf("Failed to parse BootTimeout parameter, error: %s", err)
	}

	emulatorBootDone := false
	startTime := time.Now()

	adbCommands := []string{}

	if config.EmulatorSerial != "" {
		adbCommands = append(adbCommands, "-s", config.EmulatorSerial)
	}

	adbCommands = append(adbCommands, "shell", "getprop dev.bootcomplete '0' && getprop sys.boot_completed '0' && getprop init.svc.bootanim 'running'")

	for !emulatorBootDone {
		log.Printf("> Checking if device booted")

		bootCheckCmd := command.New(filepath.Join(os.Getenv("ANDROID_HOME"), "platform-tools/adb"), adbCommands...)
		bootCheckOut, err := bootCheckCmd.RunAndReturnTrimmedCombinedOutput()
		if err != nil {
			failf("Failed to check emulator boot status, error: %s", err)
		}

		if bootCheckOut == "1\n1\nstopped" {
			time.Sleep(25 * time.Second)
			break
		}

		if time.Now().Sub(startTime) >= time.Duration(timeout)*time.Second {
			failf("Waiting for emulator boot timed out after %d seconds", timeout)
		}

		time.Sleep(3 * time.Second)
	}

	log.Donef("> Device booted")
}
