//go:build mage
// +build mage

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
)

const Name = "dnsmasq-web"

var Default = Debug

func runBuildScript(args ...string) error {
	cmd := exec.Command("bash", append([]string{"scripts/build.sh"}, args...)...)
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func Debug() error {
	return runBuildScript("debug")
}

func Release() error {
	return runBuildScript("release")
}

func PrepareReleaseImage() error {
	return runBuildScript("prepare-release-image")
}

func Binaries() error {
	return Release()
}

func All() error {
	return Debug()
}

func CleanBinaries() error {
	return runBuildScript("clean")
}

func CleanDaemonFiles() error {
	debugDir := filepath.Join("dist", "debug")
	for _, extension := range []string{"pid", "sock"} {
		if err := os.RemoveAll(filepath.Join(debugDir, fmt.Sprintf("%s.%s", Name, extension))); err != nil {
			return err
		}
	}
	if err := os.RemoveAll(filepath.Join(debugDir, "reservations.d")); err != nil {
		return err
	}
	return nil
}

func Clean() {
	if err := CleanBinaries(); err != nil {
		fmt.Println(err)
	}
	if err := CleanDaemonFiles(); err != nil {
		fmt.Println(err)
	}
}

func Kill() error {
	// Get the PID from the file
	if pidStr, err := os.ReadFile(filepath.Join("dist", "debug", fmt.Sprintf("%s.pid", Name))); err != nil {
		return fmt.Errorf("failed to read PID file: %v", err)
	} else if pid, err := strconv.Atoi(string(pidStr)); err != nil {
		return fmt.Errorf("failed to convert PID to integer: %v", err)
	} else {
		return syscall.Kill(pid, syscall.SIGTERM)
	}
}

func Stop() error {
	if err := Kill(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to stop daemon: %v\n", err)
	}
	return CleanDaemonFiles()
}

func Start() error {
	if err := Debug(); err != nil {
		return err
	}
	cwd, _ := os.Getwd()
	debugDir := filepath.Join(cwd, "dist", "debug")
	cmdName := filepath.Join(debugDir, Name)
	fileNameOf := func(extension string) string {
		return filepath.Join(debugDir, fmt.Sprintf("%s.%s", Name, extension))
	}
	if err := os.MkdirAll(filepath.Join(debugDir, "reservations.d"), 0755); err != nil {
		return err
	}
	cmd := exec.Command(cmdName, "server", "-v", "-f", "leases.db", "-h", "reservations.d", "-d", "-S", fileNameOf("sock"), "-P", fileNameOf("pid"))
	cmd.Dir = debugDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
