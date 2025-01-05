package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/user"
	"slices"
	"strconv"
	"syscall"
)

// id returns the UID and GID for the given user and group which can be either identifiers or names
func id(u, g string) (uint32, uint32, error) {
	var uid, gid int
	if g != "" {
		if grp, err := user.LookupGroup(g); err == nil {
			if gid, err = strconv.Atoi(grp.Gid); err != nil {
				return 0, 0, fmt.Errorf("invalid GID for group %s: %v", g, err)
			}
		} else {
			return 0, 0, fmt.Errorf("failed to lookup group %s: %v", g, err)
		}
	}
	if u != "" {
		if usr, err := user.Lookup(u); err == nil {
			if uid, err = strconv.Atoi(usr.Uid); err != nil {
				return 0, 0, fmt.Errorf("invalid UID for user %s: %v", u, err)
			}
			if g == "" {
				if gid, err = strconv.Atoi(usr.Gid); err != nil {
					return 0, 0, fmt.Errorf("invalid GID for user %s: %v", u, err)
				}
			}
		} else {
			return 0, 0, fmt.Errorf("failed to lookup user %s: %v", u, err)
		}
	}
	return uint32(uid), uint32(gid), nil
}

// Listen returns the listener file descriptor
func Listen(on string) (*os.File, error) {
	if listener, err := net.Listen("tcp", on); err != nil {
		return nil, fmt.Errorf("unable to listen on %s: %v", on, err)
	} else if file, err := listener.(*net.TCPListener).File(); err != nil {
		return nil, fmt.Errorf("unable to get listener file: %v", err)
	} else {
		return file, nil
	}
}

// RunDaemon starts the child process in the background and exits the parent process
func RunDaemon(pidFilePath, user, group string, env []string, extraFiles []*os.File) int {
	// cmd is the background (child) process this (parent) process will start then exit
	cmd := exec.Command(os.Args[0])
	cmd.Env = env
	cmd.ExtraFiles = extraFiles
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	// Remove flags only relavent to the parent process
	dFlagIndex := slices.Index(os.Args, "-d")
	gFlagIndex := slices.Index(os.Args, "-g")
	uFlagIndex := slices.Index(os.Args, "-u")
	vFlagIndex := slices.Index(os.Args, "-v")
	for i := 1; i < len(os.Args); i++ {
		switch i {
		case gFlagIndex, uFlagIndex:
			i++ // Skip the flag's argument
		case dFlagIndex, vFlagIndex:
		default:
			cmd.Args = append(cmd.Args, os.Args[i])
		}
	}
	// Set the user and group for the child process, i.e., drop root privileges
	if user != "" || group != "" {
		if uid, gid, err := id(user, group); err == nil {
			cmd.SysProcAttr.Credential = &syscall.Credential{Uid: uid, Gid: gid}
		} else {
			fmt.Fprintf(os.Stderr, "unable to set user and group: %v\n", err)
			os.Exit(1)
		}
	}

	// Start the child process in the background
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "unable to start as a daemon: %v\n", err)
		os.Exit(1)
	}
	// exitAll exits the parent process after killing the child process
	exitAll := func(status int, message string, args ...any) {
		fmt.Fprintf(os.Stderr, message, args...)
		cmd.Process.Kill()
		os.Exit(status)

	}
	// Write the PID file
	if pidFile, err := os.Create(pidFilePath); err != nil {
		exitAll(1, "unable to create PID file: '%s': %v\n", pidFilePath, err)
	} else {
		defer pidFile.Close()
		if _, err := pidFile.WriteString(strconv.Itoa(cmd.Process.Pid)); err != nil {
			exitAll(1, "unable to write PID to file: %v\n", err)
		}
	}
	return cmd.Process.Pid
}
