package main

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/mock"
)

type MockCmdRunner struct {
	mock.Mock
}

func (r *MockCmdRunner) RunCommandWithTimeout(name string, args []string) (string, error) {
	a := r.Called(name, args)
	return a.String(0), a.Error(1)
}

type MockClock struct {
	mock.Mock
}

func (c *MockClock) Now() time.Time {
	args := c.Called()
	return args.Get(0).(time.Time)
}

func (c *MockClock) Since(t time.Time) time.Duration {
	args := c.Called(t)
	return args.Get(0).(time.Duration)
}

func (c *MockClock) Sleep(d time.Duration) {
	c.Called(d)
}

func (c *MockClock) After(d time.Duration) <-chan time.Time {
	args := c.Called(d)
	return args.Get(0).(<-chan time.Time)
}

func Test_checkEmulatorBootState_daemonRestart(t *testing.T) {
	androidHome := "android-home"
	emulatorSerial := "serial"
	timeoutSec := 20

	mockCmdRunner := new(MockCmdRunner)

	name, args := adbWaitForDeviceShellCommand(androidHome, emulatorSerial, "getprop sys.boot_completed")
	mockCmdRunner.On("RunCommandWithTimeout", name, args).Return("1", nil).Once()

	name, args = adbWaitForDeviceShellCommand(androidHome, emulatorSerial, "getprop dev.bootcomplete")
	mockCmdRunner.On("RunCommandWithTimeout", name, args).Return("daemon not running; starting now at tcp:5037", errors.New("exit status 1")).Once()
	mockCmdRunner.On("RunCommandWithTimeout", name, args).Return("1", nil).Once()

	name, args = adbWaitForDeviceShellCommand(androidHome, emulatorSerial, "getprop init.svc.bootanim")
	mockCmdRunner.On("RunCommandWithTimeout", name, args).Return("stopped", nil).Once()

	cmdRunner = mockCmdRunner

	mockClock := new(MockClock)
	mockClock.On("Now").Return(time.Time{}).Once()
	mockClock.On("Since", mock.Anything).Return(time.Duration(timeoutSec-1) * time.Second).Once()
	mockClock.On("Sleep", mock.Anything).Return().Once()
	clock = mockClock

	err := checkEmulatorBootState(androidHome, emulatorSerial, time.Duration(timeoutSec)*time.Second)
	require.NoError(t, err)

	mockCmdRunner.AssertExpectations(t)
	mockClock.AssertExpectations(t)
}

func Test_checkEmulatorBootState_timeout(t *testing.T) {
	androidHome := "android-home"
	emulatorSerial := "serial"
	timeoutSec := 20

	mockCmdRunner := new(MockCmdRunner)

	name, args := adbWaitForDeviceShellCommand(androidHome, emulatorSerial, "getprop sys.boot_completed")
	mockCmdRunner.On("RunCommandWithTimeout", name, args).Return("1", nil).Once()

	name, args = adbWaitForDeviceShellCommand(androidHome, emulatorSerial, "getprop dev.bootcomplete")
	mockCmdRunner.On("RunCommandWithTimeout", name, args).Return("", errTimedOut).Once()

	name, args = adbCommand(androidHome, "", "kill-server")
	mockCmdRunner.On("RunCommandWithTimeout", name, args).Return("", nil).Once()

	name, args = adbWaitForDeviceShellCommand(androidHome, emulatorSerial, "getprop dev.bootcomplete")
	mockCmdRunner.On("RunCommandWithTimeout", name, args).Return("1", nil).Once()

	name, args = adbWaitForDeviceShellCommand(androidHome, emulatorSerial, "getprop init.svc.bootanim")
	mockCmdRunner.On("RunCommandWithTimeout", name, args).Return("stopped", nil).Once()

	cmdRunner = mockCmdRunner

	mockClock := new(MockClock)
	mockClock.On("Now").Return(time.Time{}).Once()
	mockClock.On("Since", mock.Anything).Return(time.Duration(timeoutSec-1) * time.Second).Once()
	mockClock.On("Sleep", mock.Anything).Return().Once()
	clock = mockClock

	err := checkEmulatorBootState(androidHome, emulatorSerial, time.Duration(timeoutSec)*time.Second)
	require.NoError(t, err)

	mockCmdRunner.AssertExpectations(t)
	mockClock.AssertExpectations(t)
}
