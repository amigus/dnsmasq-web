package main

import (
	"net"
	"os"
	"os/signal"
	"os/user"
	"strconv"
	"syscall"
	"testing"
	"time"
)

func TestIdSupportsNamesAndNumericIdentifiers(t *testing.T) {
	currentUser, err := user.Current()
	if err != nil {
		t.Fatalf("failed to get current user: %v", err)
	}
	currentGroup, err := user.LookupGroupId(currentUser.Gid)
	if err != nil {
		t.Fatalf("failed to get current group: %v", err)
	}
	expectedUID, err := strconv.Atoi(currentUser.Uid)
	if err != nil {
		t.Fatalf("failed to parse current uid: %v", err)
	}
	expectedGID, err := strconv.Atoi(currentUser.Gid)
	if err != nil {
		t.Fatalf("failed to parse current gid: %v", err)
	}

	uid, gid, err := id(currentUser.Username, currentGroup.Name)
	if err != nil {
		t.Fatalf("failed to resolve user/group by name: %v", err)
	}
	if int(uid) != expectedUID || int(gid) != expectedGID {
		t.Fatalf("expected UID/GID %d/%d, got %d/%d", expectedUID, expectedGID, uid, gid)
	}

	uid, gid, err = id(currentUser.Uid, currentUser.Gid)
	if err != nil {
		t.Fatalf("failed to resolve user/group by numeric identifier: %v", err)
	}
	if int(uid) != expectedUID || int(gid) != expectedGID {
		t.Fatalf("expected UID/GID %d/%d, got %d/%d", expectedUID, expectedGID, uid, gid)
	}
}

func TestListenReturnsUsableFileDescriptor(t *testing.T) {
	file, err := Listen("127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}
	if file == nil {
		t.Fatal("expected listener file descriptor to be populated")
	}
	defer file.Close()
	listener, err := net.FileListener(file)
	if err != nil {
		t.Fatalf("failed to recreate listener from file: %v", err)
	}
	listener.Close()
}

func TestRunDaemonHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM)
	select {
	case <-signals:
		os.Exit(0)
	case <-time.After(5 * time.Second):
		os.Exit(0)
	}
}

func TestRunDaemonWritesPIDFile(t *testing.T) {
	pidFile, err := os.CreateTemp("", "pid")
	if err != nil {
		t.Fatalf("failed to create temporary file: %v", err)
	}
	pidFile.Close()
	defer os.Remove(pidFile.Name())

	oldArgs := os.Args
	os.Args = []string{oldArgs[0], "-test.run=^TestRunDaemonHelperProcess$"}
	defer func() {
		os.Args = oldArgs
	}()

	// preserveEnv=true only inherits the parent environment, so export the
	// helper marker before spawning the child process.
	t.Setenv("GO_WANT_HELPER_PROCESS", "1")
	process := RunDaemon("127.0.0.1:0", "", "", true, nil)
	if process == nil || process.Pid == 0 {
		t.Fatal("expected a started process with a non-zero pid")
	}
	WritePidFile(process, pidFile.Name())

	pidContent, err := os.ReadFile(pidFile.Name())
	if err != nil {
		t.Fatalf("failed to read PID file: %v", err)
	}
	pidStr := string(pidContent)
	if pidStr != strconv.Itoa(process.Pid) {
		t.Fatalf("expected PID %d, got %s", process.Pid, pidStr)
	}

	if err := syscall.Kill(process.Pid, syscall.SIGTERM); err != nil {
		t.Fatalf("failed to send SIGTERM to daemon process %d: %v", process.Pid, err)
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		var status syscall.WaitStatus
		reapedPID, err := syscall.Wait4(process.Pid, &status, syscall.WNOHANG, nil)
		if err == nil && reapedPID == process.Pid {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("daemon process %d did not exit after SIGTERM", process.Pid)
}
