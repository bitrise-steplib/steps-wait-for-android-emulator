package main

import (
	"fmt"
	"os"
	"strconv"
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

// Clock ...
type Clock interface {
	Now() time.Time
	Since(t time.Time) time.Duration
	Sleep(d time.Duration)
	After(d time.Duration) <-chan time.Time
}

var clock Clock = defaultClock{}

// Inputs ...
type Inputs struct {
	EmulatorSerial string `env:"emulator_serial,required"`
	BootTimeout    string `env:"boot_timeout,required"`

	AndroidHome    string `env:"ANDROID_HOME"`
	AndroidSDKRoot string `env:"ANDROID_SDK_ROOT"`
}

func failf(format string, v ...interface{}) {
	log.Errorf(format, v...)
	os.Exit(1)
}

func checkEmulatorBootState(adbManager adbmanager.Manager, emulatorSerial string, timeout time.Duration) error {
	startTime := clock.Now()

	log.Printf("Checking if device booted...")

	for {
		if err := adbManager.StartServer(); err != nil {
			log.Warnf("failed to start adb server: %s", err)
			log.Warnf("restarting adb server...")
			if err := adbManager.RestartServer(); err != nil {
				return fmt.Errorf("failed to start adb server: %s", err)
			}
		}

		out, err := adbManager.WaitForDeviceShell(emulatorSerial, "getprop sys.boot_completed")
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

		if clock.Since(startTime) >= timeout {
			return fmt.Errorf("emulator boot status checked timed out")
		}

		clock.Sleep(2 * time.Second)
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

	// Initialize Android SDK
	log.Printf("Initialize Android SDK")
	androidSDK, err := sdk.NewDefaultModel(sdk.Environment{
		AndroidHome:    inputs.AndroidHome,
		AndroidSDKRoot: inputs.AndroidSDKRoot,
	})
	if err != nil {
		failf("Failed to initialize Android SDK: %s", err)
	}

	adb, err := adbmanager.New(androidSDK, command.NewFactory(envRepo))
	if err != nil {
		failf("Failed to create adb model, error: %s", err)
	}

	timeout, err := strconv.ParseInt(inputs.BootTimeout, 10, 64)
	if err != nil {
		failf("Failed to parse BootTimeout parameter, error: %s", err)
	}

	adbManager := adbmanager.NewManager(androidSDK, command.NewFactory(env.NewRepository()), logv2.NewLogger())
	if err := checkEmulatorBootState(adbManager, inputs.EmulatorSerial, time.Duration(timeout)*time.Second); err != nil {
		failf(err.Error())
	}

	if err := adb.UnlockDevice(inputs.EmulatorSerial); err != nil {
		failf("UnlockDevice command failed, error: %s", err)
	}

	log.Donef("> Device booted")
}
