package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/bitrise-io/go-android/v2/adbmanager"
	"github.com/bitrise-io/go-android/v2/sdk"
	"github.com/bitrise-io/go-steputils/v2/stepconf"
	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/log"
)

// Clock ...
type Clock interface {
	Now() time.Time
	Since(t time.Time) time.Duration
	Sleep(d time.Duration)
	After(d time.Duration) <-chan time.Time
}

var clock Clock = defaultClock{}
var logger = log.NewLogger()

// Inputs ...
type Inputs struct {
	EmulatorSerial string `env:"emulator_serial,required"`
	BootTimeout    string `env:"boot_timeout,required"`
	AndroidHome    string `env:"android_home,dir"`
}

func failf(format string, v ...interface{}) {
	logger.Errorf(format, v...)
	os.Exit(1)
}

type WaitForBootCompleteResult struct {
	Booted bool
	Error  error
}

func getBootCompleteEvent(adbManager *adbmanager.Model, serial string, timeout time.Duration) <-chan WaitForBootCompleteResult {
	doneChan := make(chan WaitForBootCompleteResult)

	go func() {
		time.AfterFunc(timeout, func() {
			doneChan <- WaitForBootCompleteResult{Error: errors.New("timeout")}
		})
	}()

	go func() {
		out, err := adbManager.WaitForDeviceThenShellCmd(serial, nil, "getprop sys.boot_completed").RunAndReturnTrimmedCombinedOutput()
		if err != nil {
			fmt.Println(out)
			doneChan <- WaitForBootCompleteResult{Error: err}
			return
		}

		lines := strings.Split(out, "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) == "1" {
				doneChan <- WaitForBootCompleteResult{Booted: true}
				return
			}
		}

		doneChan <- WaitForBootCompleteResult{Booted: false}
	}()

	return doneChan
}

func waitForDevice(adb *adbmanager.Model, emulatorSerial string, timeout time.Duration) error {
	startTime := clock.Now()

	for {
		logger.Printf("Waiting for emulator boot...")

		bootCompleteChan := getBootCompleteEvent(adb, emulatorSerial, timeout)
		result := <-bootCompleteChan
		switch {
		case result.Error != nil:
			logger.Warnf("Failed to check emulator boot status: %s", result.Error)
			logger.Warnf("Killing ADB server before retry...")
			if err := adb.KillServerCmd(nil).Run(); err != nil {
				return fmt.Errorf("failed to terminate adb server: %s", err)
			}
		case result.Booted:
			logger.Donef("Device boot completed in %d seconds", time.Since(startTime)/time.Second)
			return nil
		}

		if time.Now().After(startTime.Add(timeout)) {
			return fmt.Errorf("emulator boot check timed out after %s seconds", time.Since(startTime)/time.Second)
		}

		delay := 5 * time.Second
		logger.Printf("Device is online but still booting, retrying in %d seconds", delay/time.Second)
		time.Sleep(delay)
	}
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

	adb, err := adbmanager.New(androidSdk, cmdFactory)
	if err != nil {
		failf("Failed to create ADB model: %s", err)
	}

	timeout, err := strconv.ParseInt(inputs.BootTimeout, 10, 64)
	if err != nil {
		failf("Failed to parse boot_timeout parameter: %s", err)
	}

	if err := waitForDevice(adb, inputs.EmulatorSerial, time.Duration(timeout)*time.Second); err != nil {
		failf(err.Error())
	}

	logger.Println()
	logger.Printf("Unlocking device...")
	if err := adb.UnlockDevice(inputs.EmulatorSerial); err != nil {
		failf("UnlockDevice command failed: %s", err)
	}

	logger.Donef("Device is ready")
}
