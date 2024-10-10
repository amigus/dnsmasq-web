//go:build mage
// +build mage

package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

const Name = "dnsmasq-web"

var Default = All

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

// Build the client POSIX shell environment
func createClientEnv(full, minify bool) error {
	scripts := []string{"use.sh", "token.sh", "curl.sh"}

	if full {
		scripts = append(scripts, "curl_jq.sh", "jq_commands.sh", "reservations.sh")
	}

	var script bytes.Buffer

	script.WriteString("# Dnsmasq Web Client environment")
	for _, scriptName := range scripts {
		content, err := os.ReadFile("cli/" + scriptName)
		if err != nil {
			return fmt.Errorf("failed to read %s: %v", scriptName, err)
		}

		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			if !strings.HasPrefix(line, "#!") {
				script.WriteString(line + "\n")
			}
		}
	}

	cmdArgs := []string{"shfmt", "-p"}
	if minify {
		cmdArgs = append(cmdArgs, "-mn")
	}
	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Stdin = &script
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to minify script: %v", err)
	}

	return os.WriteFile(fmt.Sprintf("%s.env", Name), output, 0755)
}

// Build the client environment
func ClientEnv() error {
	return createClientEnv(false, true)
}

// Build the full client environment
func FullClientEnv() error {
	return createClientEnv(true, false)
}

// Build binaries for pre-selected architectures
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

// Build the client environment and the binaries
func All() error {
	if err := ClientEnv(); err != nil {
		return err
	}
	return Binaries()
}

func CleanBinaries() error {
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

func CleanClientEnv() error {
	return os.RemoveAll(fmt.Sprintf("%s.env", Name))
}


func Clean() {
	if err := CleanClientEnv(); err != nil {
		fmt.Println(err)
	}
	if err := CleanBinaries(); err != nil {
		fmt.Println(err)
	}
}
