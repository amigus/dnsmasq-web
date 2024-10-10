//go:build mage
// +build mage

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

const Name = "dnsmasq-web"

var Default = Binaries

// Get the build info from git and add a datetime stamp
func buildInfo() (string, error) {
	// Get the git describe output
	cmd := exec.Command("git", "describe", "--tags", "--always")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get build info: %v", err)
	}
	gitInfo := strings.TrimSpace(string(output))

	// Get the current datetime
	datetime := time.Now().Format("2006-01-02")

	// Get the build user
	user := os.Getenv("USER")
	hostname := os.Getenv("HOSTNAME")

	// Combine git info and datetime
	buildInfo := fmt.Sprintf("%s # %s (%s@%s)", gitInfo, datetime, user, hostname)
	return buildInfo, nil
}

// Run the go build command
func runBuild(name string, envVars ...string) error {
	buildInfo, err := buildInfo()
	if err != nil {
		return err
	}
	cmd := exec.Command("go", "build", "-buildmode=pie", "-ldflags",
		fmt.Sprintf("-w -s -X 'main.Version=%s'", buildInfo), "-o", name)
	if len(envVars) > 0 {
		cmd.Env = append(os.Environ(), envVars...)
	} else {
		cmd.Env = os.Environ()
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Build the dnsmasq-web for the current architecture
func Binary() error {
	return runBuild(Name)
}

// Build binaries for amd64 and arm64 architectures
func Binaries() error {
	for _, combo := range []struct {
		CC     string
		GOARCH string
		Name   string
	}{
		{"gcc", "amd64", "dnsmasq-web-amd64"},
		{"aarch64-suse-linux-gcc", "arm64", "dnsmasq-web-arm64"},
	} {
		envVars := []string{
			"CC=" + combo.CC,
			"CGO_ENABLED=1",
			"GOARCH=" + combo.GOARCH,
			"GOOS=linux",
		}
		if err := runBuild(combo.Name, envVars...); err != nil {
			return err
		}
	}
	return nil
}

// Clean the project
func Clean() error {
	if err := os.RemoveAll("dnsmasq-web"); err != nil {
		return err
	}
	for _, output := range []string{
		"dnsmasq-web-amd64",
		"dnsmasq-web-arm64",
	} {
		if err := os.RemoveAll(output); err != nil {
			return err
		}
	}
	return nil
}
