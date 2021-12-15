package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bitrise-io/go-android/adbmanager"
	"github.com/bitrise-io/go-android/sdk"
	"github.com/bitrise-io/go-steputils/v2/stepconf"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/env"
	logv2 "github.com/bitrise-io/go-utils/v2/log"
)

// Inputs ...
type Inputs struct {
	EmulatorSerial string `env:"emulator_serial,required"`
	BootTimeout    int    `env:"boot_timeout,required"`

	AndroidHome    string `env:"ANDROID_HOME"`
	AndroidSDKRoot string `env:"ANDROID_SDK_ROOT"`
}

func failf(format string, v ...interface{}) {
	log.Errorf(format, v...)
	os.Exit(1)
}

func checkEmulatorBootState(adbManager adbmanager.Manager, serial string, deadline time.Time) error {
	log.Printf("Checking if device booted...")

	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("emulator boot status checked timed out")
		}

		if err := adbManager.StartServer(); err != nil {
			log.Warnf("failed to start adb server: %s", err)
			log.Warnf("restarting adb server...")
			if err := adbManager.RestartServer(); err != nil {
				return fmt.Errorf("failed to start adb server: %s", err)
			}
		}

		out, err := adbManager.WaitForDeviceShell(serial, "getprop sys.boot_completed")
		fmt.Println(out)
		if err != nil {
			log.Warnf("failed to check emulator boot status: %s", err)
			log.Warnf("restarting adb server...")
			if err := adbManager.KillServer(); err != nil {
				return fmt.Errorf("failed to kill adb server: %s", err)
			}

			time.Sleep(2 * time.Second)
			continue
		}

		lines := strings.Split(out, "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) == "1" {
				return nil
			}
		}

		time.Sleep(2 * time.Second)
	}
}

func unlockDevice(adbManager adbmanager.Manager, serial string, deadline time.Time) error {
	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("unlock emulator timed out")
		}

		if err := adbManager.StartServer(); err != nil {
			log.Warnf("failed to start adb server: %s", err)
			log.Warnf("restarting adb server...")
			if err := adbManager.RestartServer(); err != nil {
				return fmt.Errorf("failed to start adb server: %s", err)
			}
		}

		out, err := adbManager.UnlockDevice(serial)
		fmt.Println(out)
		if err != nil {
			log.Warnf("failed to unlock emulator: %s", err)
			log.Warnf("restarting adb server...")
			if err := adbManager.KillServer(); err != nil {
				return fmt.Errorf("failed to kill adb server: %s", err)
			}

			time.Sleep(2 * time.Second)
			continue
		}

		return nil
	}
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

	// adb shell input help
	// Initialize Android SDK
	log.Printf("Initialize Android SDK")
	androidSDK, err := sdk.NewDefaultModel(sdk.Environment{
		AndroidHome:    inputs.AndroidHome,
		AndroidSDKRoot: inputs.AndroidSDKRoot,
	})
	if err != nil {
		failf("Failed to initialize Android SDK: %s", err)
	}

	deadline := time.Now().Add(time.Duration(inputs.BootTimeout) * time.Second)

	adbManager := adbmanager.NewManager(androidSDK, command.NewFactory(env.NewRepository()), logv2.NewLogger())
	if err := checkEmulatorBootState(adbManager, inputs.EmulatorSerial, deadline); err != nil {
		failf(err.Error())
	}

	if err := unlockDevice(adbManager, inputs.EmulatorSerial, deadline); err != nil {
		failf("UnlockDevice command failed, error: %s", err)
	}

	log.Donef("> Device booted")
}
