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

// CmdRunner ...
type CmdRunner interface {
	RunCommandWithTimeout(name string, args []string) (string, error)
}

var cmdRunner CmdRunner = defaultCmdRunner{}

// Clock ...
type Clock interface {
	Now() time.Time
	Since(t time.Time) time.Duration
	Sleep(d time.Duration)
	After(d time.Duration) <-chan time.Time
}

var clock Clock = defaultClock{}

var errTimedOut = errors.New("running command timed out")

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

func isDeviceBooted(androidHome, serial string) (bool, error) {
	formatErr := func(out string, err error) error {
		if err == errTimedOut {
			return err
		}
		return fmt.Errorf("%s - %s", out, err)
	}

	dev, err := cmdRunner.RunCommandWithTimeout(adbShellCommand(androidHome, serial, "getprop dev.bootcomplete"))
	if err != nil {
		return false, formatErr(dev, err)
	}

	sys, err := cmdRunner.RunCommandWithTimeout(adbShellCommand(androidHome, serial, "getprop sys.boot_completed"))
	if err != nil {
		return false, formatErr(sys, err)
	}

	init, err := cmdRunner.RunCommandWithTimeout(adbShellCommand(androidHome, serial, "getprop init.svc.bootanim"))
	if err != nil {
		return false, formatErr(init, err)
	}

	return dev == "1" && sys == "1" && init == "stopped", nil
}

func terminateADBServer(androidHome string) error {
	name, args := adbCommand(androidHome, "", "kill-server")
	_, err := cmdRunner.RunCommandWithTimeout(name, args)
	return err
}

func checkEmulatorBootState(androidHome, emulatorSerial string, timeout time.Duration) error {
	startTime := clock.Now()

	for {
		logger.Printf("> Checking if boot completed...")

		booted, err := isDeviceBooted(androidHome, emulatorSerial)
		if err != nil {
			switch {
			case strings.Contains(err.Error(), "daemon not running; starting now at"):
				logger.Warnf("adb daemon being restarted")
				logger.Printf(err.Error())
			case err == errTimedOut:
				logger.Warnf("Running command timed out, retry...")
				if err := terminateADBServer(androidHome); err != nil {
					if err != errTimedOut {
						return fmt.Errorf("unable to kill ADB daemon, error: %s", err)
					}
					logger.Warnf("killing ADB daemon timed out")
				}
			case err != nil:
				return fmt.Errorf("failed to check emulator boot status: %s", err)
			}
		}

		if booted {
			break
		}

		if clock.Since(startTime) >= timeout {
			return fmt.Errorf("waiting for emulator boot timed out after %d seconds", timeout)
		}

		clock.Sleep(5 * time.Second)
	}

	return nil
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
	logger.Infof("Waiting for emulator to boot...")

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
		failf("Failed to parse BootTimeout parameter: %s", err)
	}

	if err := checkEmulatorBootState(inputs.AndroidHome, inputs.EmulatorSerial, time.Duration(timeout)*time.Second); err != nil {
		failf(err.Error())
	}

	if err := adb.UnlockDevice(inputs.EmulatorSerial); err != nil {
		failf("UnlockDevice command failed: %s", err)
	}

	logger.Donef("> Device booted")
}
