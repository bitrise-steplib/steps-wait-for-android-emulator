package main

import (
	"fmt"
	"os"
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

	deadline := time.Now().Add(time.Duration(inputs.BootTimeout) * time.Second)

	adbManager := adbmanager.NewManager(androidSDK, command.NewFactory(env.NewRepository()), logv2.NewLogger())
	if err := WaitForBootComplete(adbManager, inputs.EmulatorSerial, deadline); err != nil {
		failf(err.Error())
	}

	if err := UnlockDevice(adbManager, inputs.EmulatorSerial, deadline); err != nil {
		failf("UnlockDevice command failed, error: %s", err)
	}

	log.Donef("> Device booted")
}
