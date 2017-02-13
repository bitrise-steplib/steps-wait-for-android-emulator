package tools

import (
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/pathutil"
)

// ADBModel ...
type ADBModel struct {
	pth string
}

// NewADB ...
func NewADB(androidHomeDir string) (*ADBModel, error) {
	adbPth := filepath.Join(androidHomeDir, "platform-tools", "adb")
	if exist, err := pathutil.IsPathExists(adbPth); err != nil {
		return nil, fmt.Errorf("failed to check if adb exist, error: %s", err)
	} else if !exist {
		return nil, fmt.Errorf("adb not exist at: %s", adbPth)
	}
	return &ADBModel{
		pth: adbPth,
	}, nil
}

// DevicesCmd ...
func (adb ADBModel) DevicesCmd() *command.Model {
	return command.New(adb.pth, "devices")
}

// IsDeviceBooted ...
func (adb ADBModel) IsDeviceBooted(serial string) (bool, error) {
	devBootCmd := command.New(adb.pth, "-s", serial, "shell", "getprop dev.bootcomplete")
	devBootOut, err := devBootCmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return false, err
	}

	sysBootCmd := command.New(adb.pth, "-s", serial, "shell", "getprop sys.boot_completed")
	sysBootOut, err := sysBootCmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return false, err
	}

	bootAnimCmd := command.New(adb.pth, "-s", serial, "shell", "getprop init.svc.bootanim")
	bootAnimOut, err := bootAnimCmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return false, err
	}

	return (devBootOut == "1" && sysBootOut == "1" && bootAnimOut == "stopped"), nil
}

// UnlockDevice ...
func (adb ADBModel) UnlockDevice(serial string) error {
	keyEvent82Cmd := command.New(adb.pth, "-s", serial, "shell", "input keyevent 82 &")
	if err := keyEvent82Cmd.Run(); err != nil {
		return err
	}

	keyEvent1Cmd := command.New(adb.pth, "-s", serial, "shell", "input keyevent 1 &")
	return keyEvent1Cmd.Run()
}

// EmulatorModel ...
type EmulatorModel struct {
	pth string
}

// NewEmulator ...
func NewEmulator(androidHomeDir string) (*EmulatorModel, error) {
	emulatorPth := filepath.Join(androidHomeDir, "tools", "emulator")
	if runtime.GOOS == "linux" {
		emulatorPth = filepath.Join(androidHomeDir, "tools", "emulator64-arm")
	}
	if exist, err := pathutil.IsPathExists(emulatorPth); err != nil {
		return nil, fmt.Errorf("failed to check if emulator exist, error: %s", err)
	} else if !exist {
		return nil, fmt.Errorf("emulator not exist at: %s", emulatorPth)
	}
	return &EmulatorModel{
		pth: emulatorPth,
	}, nil
}

// StartEmulatorCmd ...
func (emulator EmulatorModel) StartEmulatorCmd(name, skin string, options ...string) *command.Model {
	args := []string{emulator.pth, "-avd", name}
	if len(skin) == 0 {
		args = append(args, "-noskin")
	} else {
		args = append(args, "-skin", skin)
	}

	args = append(args, options...)

	return command.New(args[0], args[1:]...)
}
