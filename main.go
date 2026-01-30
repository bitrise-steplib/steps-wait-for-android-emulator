package main

import (
	"fmt"
	"os"
	"time"

	"github.com/bitrise-io/go-android/v2/adbmanager"
	"github.com/bitrise-io/go-android/v2/sdk"
	"github.com/bitrise-io/go-steputils/v2/stepconf"
	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-utils/v2/system"
)

var logger = log.NewLogger()

type Inputs struct {
	EmulatorSerial string `env:"emulator_serial,required"`
	BootTimeout    int    `env:"boot_timeout,required"`
	AndroidHome    string `env:"android_home,dir"`
}

func failf(format string, v ...interface{}) {
	logger.Errorf(format, v...)

	cpuIsARM, err := system.CPU.IsARM()
	if err != nil {
		logger.Errorf("Failed to check CPU: %s", err)
	} else if cpuIsARM {
		logger.Warnf("Android emulator is not supported on Apple Silicon (M1) build VMs. Try running this workflow on a Linux-based stack or use the Virtual Device Testing step. Learn more:\n* Set a workflow-specific stack: https://devcenter.bitrise.io/en/builds/configuring-build-settings/setting-the-stack-for-your-builds.html#setting-a-workflow-specific-stack-on-the-stacks---machines-tab\n * Virtual device testing step: https://github.com/bitrise-steplib/steps-virtual-device-testing-for-android")
	}

	os.Exit(1)
}

func main() {
	envRepo := env.NewRepository()
	cmdFactory := command.NewFactory(env.NewRepository())

	var inputs Inputs
	if err := stepconf.NewInputParser(envRepo).Parse(&inputs); err != nil {
		failf("Issue with inputs: %s", err)
	}
	stepconf.Print(inputs)

	fmt.Println()

	androidSdk, err := sdk.New(inputs.AndroidHome)
	if err != nil {
		failf("Failed to create Android SDK: %s", err)
	}

	adb, err := adbmanager.New(androidSdk, cmdFactory, logger)
	if err != nil {
		failf("Failed to create ADB model: %s", err)
	}

	if err := adb.WaitForDevice(inputs.EmulatorSerial, time.Duration(inputs.BootTimeout)*time.Second); err != nil {
		failf(err.Error())
	}

	logger.Println()
	logger.Printf("Unlocking device...")
	if err := adb.UnlockDevice(inputs.EmulatorSerial); err != nil {
		failf("UnlockDevice command failed: %s", err)
	}

	logger.Donef("Device is ready")
}
