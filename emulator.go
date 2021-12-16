package main

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bitrise-io/go-android/adbmanager"
	"github.com/bitrise-io/go-utils/log"
)

type waitForBootCompleteResult struct {
	Booted bool
	Error  error
}

func WaitForBootComplete(adbManager adbmanager.Manager, serial string, deadline time.Time) error {
	log.Printf("Checking if device booted...")

	const sleepTime = 5 * time.Second

	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("emulator boot status check timed out")
		}

		if err := adbManager.StartServer(); err != nil {
			log.TWarnf("failed to start adb server: %s", err)
			log.TWarnf("restarting adb server...")
			if err := adbManager.RestartServer(); err != nil {
				return fmt.Errorf("failed to restart adb server: %s", err)
			}
		}

		doneChan := waitForBootComplete(adbManager, serial)
		res := <-doneChan
		switch {
		case res.Error != nil:
			log.TWarnf("failed to check emulator boot status: %s", res.Error)
			log.TWarnf("terminating adb server...")
			if err := adbManager.KillServer(); err != nil {
				return fmt.Errorf("failed to terminate adb server: %s", err)
			}
		case res.Booted:
			return nil
		}

		time.Sleep(sleepTime)
	}
}

func UnlockDevice(adbManager adbmanager.Manager, serial string, deadline time.Time) error {
	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("unlock emulator timed out")
		}

		if err := adbManager.StartServer(); err != nil {
			log.TWarnf("failed to start adb server: %s", err)
			log.TWarnf("restarting adb server...")
			if err := adbManager.RestartServer(); err != nil {
				return fmt.Errorf("failed to restart adb server: %s", err)
			}
		}

		out, err := adbManager.UnlockDevice(serial)
		fmt.Println(out)
		if err != nil {
			log.TWarnf("failed to unlock emulator: %s", err)
			log.TWarnf("terminating adb server...")
			if err := adbManager.KillServer(); err != nil {
				return fmt.Errorf("failed to terminate adb server: %s", err)
			}

			time.Sleep(2 * time.Second)
			continue
		}

		return nil
	}
}

func waitForBootComplete(adbManager adbmanager.Manager, serial string) <-chan waitForBootCompleteResult {
	doneChan := make(chan waitForBootCompleteResult)

	go func() {
		const hangTimeout = 1 * time.Minute
		time.AfterFunc(hangTimeout, func() {
			doneChan <- waitForBootCompleteResult{Error: errors.New("timeout")}
		})
	}()

	go func() {
		out, err := adbManager.WaitForDeviceShell(serial, "getprop sys.boot_completed")
		fmt.Println(out)
		if err != nil {
			doneChan <- waitForBootCompleteResult{Error: err}
			return
		}

		lines := strings.Split(out, "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) == "1" {
				doneChan <- waitForBootCompleteResult{Booted: true}
				return
			}
		}

		doneChan <- waitForBootCompleteResult{Booted: false}
	}()

	return doneChan
}
