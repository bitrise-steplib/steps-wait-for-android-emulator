package main

//type defaultCmdRunner struct{}

// RunCommandWithTimeout ...
//func (r defaultCmdRunner) RunCommandWithTimeout(name string, args []string) (string, error) {
//	cmd := exec.Command(name, args...)
//
//	var output bytes.Buffer
//	cmd.Stderr = &output
//	cmd.Stdout = &output
//
//	if err := cmd.Start(); err != nil {
//		return strings.TrimSpace(output.String()), err
//	}
//
//	done := make(chan error)
//
//	go func() { done <- cmd.Wait() }()
//	select {
//	case err := <-done:
//		return strings.TrimSpace(output.String()), err
//	case <-clock.After(20 * time.Second):
//		return strings.TrimSpace(output.String()), errTimedOut
//	}
//}

//func adbCommand(androidHome, serial, cmd string) (string, []string) {
//	name := filepath.Join(androidHome, "platform-tools/adb")
//	args := []string{}
//	if serial != "" {
//		args = append(args, "-s", serial)
//	}
//	args = append(args, cmd)
//
//	return name, args
//}

//func adbShellCommand(androidHome, serial, shellCmd string) (string, []string) {
//	name, args := adbCommand(androidHome, serial, "shell")
//	args = append(args, shellCmd)
//
//	return name, args
//}
