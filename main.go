package main

import (
	"database/sql"
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"slices"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	name := filepath.Base(os.Args[0])

	var databaseFilePath, hostDirPath, listeningOn, pidFilePath, unixSocketPath, subnet, userFlag, groupFlag string
	var daemonize bool
	var maxDays, maxTokens, maxTokenUses int
	var tokenTimeout time.Duration

	flag.BoolVar(&daemonize, "d", false, "fork and run as a daemon")
	flag.StringVar(&databaseFilePath, "f", "/var/lib/misc/dnsmasq.leases.db", "the SQLite database file")
	flag.StringVar(&hostDirPath, "h", "/var/lib/misc/dnsmasq.reservations.d", "the reservations files directory")
	flag.StringVar(&listeningOn, "l", ":8080", "the IP address and port to listen on")
	flag.IntVar(&maxDays, "m", 0, "the maximum number of days of requests to query (< 1 means no limit)")
	flag.StringVar(&subnet, "s", "192.168.0.0/16", "the subnet of in scope devices")
	flag.StringVar(&groupFlag, "g", "", "group to run the process as (requires root)")
	flag.StringVar(&pidFilePath, "P", fmt.Sprintf("/run/%s.pid", name), "the PID file")
	flag.StringVar(&unixSocketPath, "S", fmt.Sprintf("/run/%s.sock", name), "the Unix domain socket")
	flag.StringVar(&userFlag, "u", "", "user to run the process as (requires root)")
	flag.IntVar(&maxTokens, "T", 1, "the maximum number of tokens to issue at a time (0 disables token checking)")
	flag.IntVar(&maxTokenUses, "c", 0, "the maximum number of times a token can be used (the default 0 means unlimited)")
	flag.DurationVar(&tokenTimeout, "t", time.Duration(0), "the duration a token is valid (the default 0 means forever)")
	flag.Usage = func() {
		fmt.Fprintf(
			flag.CommandLine.Output(), `Dnsmasq Lease Database Web Server
Provides a RESTful API to query the the lease database maintained by dnsmasq.

Usage: %s [options] [-d [daemonize options]]
Options:
	[-f database] [-h host-dir] [-l address] [-m days] [-s subnet]
Daemonize Options:
	[-u user] [-g group]
	[-T max-tokens] [-c max-uses] [-t timeout]
	[-P path] [-S path ]

`,
			name,
		)
		flag.PrintDefaults()
		fmt.Fprint(flag.CommandLine.Output(), `
Listening on ports below 1024, e.g., -l ":80" requires root privileges.
Using the -u and -g flags to drop root privilege after opening the port is recommended.
`,
		)
	}
	flag.Parse()

	hostDir, err := os.Stat(hostDirPath)
	if err != nil {
		if os.IsNotExist(err) && !daemonize {
			fmt.Fprintf(os.Stderr, "unable to stat host directory: %v\n", err)
			os.Exit(1)
		}
	} else if !hostDir.IsDir() {
		fmt.Fprintf(os.Stderr, "host directory is not a directory: %s\n", hostDirPath)
		os.Exit(1)
	}

	if daemonize {
		// cmd is the background (child) process this (parent) process will start then exit
		cmd := exec.Command(os.Args[0])
		cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
		// It will run without -d, -u or -g because they only affect the parent process
		dFlagIndex := slices.Index(os.Args, "-d")
		gFlagIndex := slices.Index(os.Args, "-g")
		uFlagIndex := slices.Index(os.Args, "-u")
		for i := 1; i < len(os.Args); i++ {
			if i == gFlagIndex || i == uFlagIndex {
				i++
			} else if i != dFlagIndex {
				cmd.Args = append(cmd.Args, os.Args[i])
			}
		}
		// Start listening on the port and pass the descriptor to it
		if listener, err := net.Listen("tcp", listeningOn); err == nil {
			// There's no .Close() because this process will exit with the listener open
			if file, err := listener.(*net.TCPListener).File(); err == nil {
				cmd.Env = append(cmd.Env, "LISTENER_ON="+file.Name())
				cmd.ExtraFiles = []*os.File{file}
			} else {
				fmt.Fprintf(os.Stderr, "unable to get listener file: %v\n", err)
				os.Exit(1)
			}
		} else {
			fmt.Fprintf(os.Stderr, "unable to open socket: %v\n", err)
			os.Exit(1)
		}
		// Create the host directory if it doesn't exist
		if hostDir == nil {
			if err := os.Mkdir(hostDirPath, 0750); err != nil {
				fmt.Fprintf(os.Stderr, "unable to create host directory: %v\n", err)
				os.Exit(1)
			}
		}
		// Set the user and group for the child process, i.e., drop root privileges
		if userFlag != "" || groupFlag != "" {
			cmd.SysProcAttr.Credential = &syscall.Credential{}
			// user. and group.Lookup() accept names or numeric IDs
			if userFlag != "" {
				if usr, err := user.Lookup(userFlag); err == nil {
					if uid, err := strconv.Atoi(usr.Uid); err == nil {
						cmd.SysProcAttr.Credential.Uid = uint32(uid)
						if hostDir == nil {
							// Change the owner of the host directory if it was created
							os.Chown(hostDirPath, uid, -1)
						}
					} else {
						fmt.Fprintf(os.Stderr, "invalid UID for user %s: %v\n", userFlag, err)
						os.Exit(1)
					}
				} else {
					fmt.Fprintf(os.Stderr, "failed to lookup user %s: %v\n", userFlag, err)
					os.Exit(1)
				}
			}
			if groupFlag != "" {
				if grp, err := user.LookupGroup(groupFlag); err == nil {
					if gid, err := strconv.Atoi(grp.Gid); err == nil {
						cmd.SysProcAttr.Credential.Gid = uint32(gid)
						if hostDir == nil {
							os.Chown(hostDirPath, -1, gid)
						}
					} else {
						fmt.Fprintf(os.Stderr, "invalid GID for group %s: %v\n", groupFlag, err)
						os.Exit(1)
					}
				} else {
					fmt.Fprintf(os.Stderr, "failed to lookup group %s: %v\n", groupFlag, err)
					os.Exit(1)
				}
			}
		}
		// If token checking is enabled, set up the UNIX domain socket to host the TokenPublisher
		if maxTokens > 0 {
			// Remove the old UNIX domain socket if it still exists
			if err := os.Remove(unixSocketPath); err != nil && !os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "unable to remove unix socket: %v\n", err)
				os.Exit(1)
			}
			// Create a new UNIX domain socket for the child process in its place
			if listener, err := net.Listen("unix", unixSocketPath); err == nil {
				if file, err := listener.(*net.UnixListener).File(); err == nil {
					cmd.ExtraFiles = append(cmd.ExtraFiles, file)
				} else {
					fmt.Fprintf(os.Stderr, "unable to get unix socket file: %v\n", err)
					os.Exit(1)
				}
			} else {
				fmt.Fprintf(os.Stderr, "unable to listen on unix socket: %v\n", err)
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
		fmt.Fprintf(os.Stderr, "started as a daemon with PID %d\n", cmd.Process.Pid)
		os.Exit(0)
	} else {
		// Open the database using the sqlite3 package
		db, err := sql.Open("sqlite3", databaseFilePath)
		if err != nil {
			fmt.Fprintf(gin.DefaultErrorWriter, "unable to open database %s: %v",
				databaseFilePath, err)
			os.Exit(1)
		}
		defer db.Close()

		// Pass the database connection to gorm.Open
		gormDb, err := gorm.Open(sqlite.Dialector{Conn: db})
		if err != nil {
			fmt.Fprintf(os.Stderr, "unable to connect database: %v", err)
			os.Exit(1)
		}

		if os.Getenv("LISTENER_ON") != "" {
			r := gin.Default()
			// Use the token checker when running as a daemon
			ttc := NewTokenChecker(maxTokens, maxTokenUses, tokenTimeout)
			if maxTokens > 0 {
				r = TokenCheckerHeader(r, ttc, "X-Token")
				go func() {
					// Serve the TokenPublisher over a Unix domain socket
					if err := TokenCheckerPublisher(gin.Default(), ttc).RunFd(4); err != nil {
						fmt.Fprintf(os.Stderr, "unable to serve on unix socket: %v\n", err)
						os.Exit(1)
					}
				}()
			}
			r = DhcpHostDir(LeaseDatabase(r, gormDb, maxDays, subnet), hostDirPath)
			// Gin defaults to DebugMode so set this explicitly
			gin.SetMode(gin.ReleaseMode)
			// Run on the open socket; 3 because 0, 1 and 2 are stdin, stdout and stderr
			if err := r.RunFd(3); err != nil {
				fmt.Fprintf(os.Stderr, "unable to listen on already open socket: %v\n", err)
			}
		} else {
			// Run without -d and not as the child process, i.e., running in the foreground
			if err := DhcpHostDir(
				LeaseDatabase(gin.Default(), gormDb, maxDays, subnet), hostDirPath,
			).Run(listeningOn); err != nil {
				fmt.Fprintf(os.Stderr, "unable to listen on '%s': %v\n", listeningOn, err)
			}
		}
		os.Exit(1)
	}

}
