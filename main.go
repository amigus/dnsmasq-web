package main

import (
	"database/sql"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const (
	defaultDatabaseFile    = "/var/lib/misc/dnsmasq.leases.db"
	defaultReservationsDir = "/var/lib/misc/dnsmasq.reservations.d"
	defaultListenOn        = ":8080"
	defaultPidFile         = "/run/dnsmasq-web.pid"
	defaultUnixSocket      = "/run/dnsmasq-web.sock"
	listenerEnvVarName     = "LISTENING_ON"
	tokenHeader            = "X-Token"
	tokenEndpointPath      = "/"
)

var Version string = "development # 2030-12-31 (unknown@unknown)"

func main() {
	name := filepath.Base(os.Args[0])

	var databaseFilePath, hostDirPath, listenOn, pidFilePath, unixSocketPath, userFlag, groupFlag string
	var daemonize, preserveEnv, verbose bool
	var maxTokens, maxTokenUses int
	var tokenTimeout time.Duration

	flag.BoolVar(&daemonize, "d", false, "fork and run as a daemon")
	flag.BoolVar(&preserveEnv, "E", false, "preserve environment when daemonizing")
	flag.StringVar(&databaseFilePath, "f", defaultDatabaseFile, "the SQLite database file")
	flag.StringVar(&hostDirPath, "h", defaultReservationsDir, "the reservations files directory")
	flag.StringVar(&listenOn, "l", defaultListenOn, "the IP address and port to listen on")
	flag.StringVar(&groupFlag, "g", "", "group to run the process as (requires root)")
	flag.StringVar(&pidFilePath, "P", defaultPidFile, "the PID file")
	flag.StringVar(&unixSocketPath, "S", defaultUnixSocket, "the UNIX domain socket")
	flag.StringVar(&userFlag, "u", "", "user to run the process as (requires root)")
	flag.IntVar(&maxTokens, "T", 1, "the maximum number of tokens to issue at a time (0 disables token checking)")
	flag.IntVar(&maxTokenUses, "c", 0, "the maximum number of times a token can be used (the default 0 means unlimited)")
	flag.DurationVar(&tokenTimeout, "t", time.Duration(0), "the duration a token is valid (the default 0 means forever)")
	flag.BoolVar(&verbose, "v", false, "print verbose output")
	flag.Bool("V", false, "print the version and exit")
	flag.Usage = func() {
		fmt.Fprintf(
			flag.CommandLine.Output(), `
Dnsmasq Web is a database-backed JSON/HTTP API for Dnsmasq.

Usage: %s [options] [-d [daemonize options]]
Options:
    [-f database] [-h host-dir] [-l address] [-v]
Daemonize Options:
    [-E]
    [-u user] [-g group]
    [-T max-tokens] [-c max-uses] [-t timeout]
    [-P pid-file] [-S unix-socket]

`,
			name,
		)
		flag.PrintDefaults()
		fmt.Fprint(flag.CommandLine.Output(), `
Listening on ports below 1024, e.g., -l ":80" requires root privileges.
Using the -u and -g flags to drop root privilege after opening the port is recommended.
The tokens are kept in memory and are not persisted across restarts.
Setting -E copies all environment variables to the child process.
Setting -T 0 disables token checking entirely.
`,
		)
	}
	flag.Parse()

	if flag.Lookup("V").Value.String() == "true" {
		fmt.Printf("%s %s\n", name, Version)
		os.Exit(0)
	}

	// Exit if databaseFilePath does not exist or is not an SQLite database
	if db, err := sql.Open("sqlite3", databaseFilePath); err != nil {
		fmt.Fprintf(os.Stderr, "unable to open database '%s': %v\n",
			databaseFilePath, err)
		os.Exit(1)
	} else {
		defer db.Close()
		if err = db.Ping(); err != nil {
			fmt.Fprintf(os.Stderr, "database '%s' is not an SQLite database: %v\n",
				databaseFilePath, err)
			os.Exit(1)
		}
	}

	if verbose {
		fmt.Printf("using lease database: %s\n", databaseFilePath)
	}

	// Exit if hostDirPath is not an existing directory unless daemonizing
	hostDir, err := os.Stat(hostDirPath)
	if err != nil {
		if os.IsNotExist(err) && !daemonize {
			fmt.Fprintf(os.Stderr, "unable to stat host directory: %v\n", err)
			os.Exit(1)
		}
	} else if !hostDir.IsDir() {
		fmt.Fprintf(os.Stderr, "host-dir is not a directory: %s\n", hostDirPath)
		os.Exit(1)
	}

	if verbose {
		fmt.Printf("using host-dir: %s\n", hostDirPath)
	}

	if daemonize {
		// Check that host-dir is writable otherwise create it
		if hostDir != nil {
			if dir, err := os.MkdirTemp(hostDirPath, "*"); err != nil {
				fmt.Fprintf(os.Stderr, "unable to write to host directory: %v\n", err)
				os.Exit(1)
			} else {
				os.Remove(dir)
			}
		} else {
			if err := os.Mkdir(hostDirPath, 0750); err != nil {
				fmt.Fprintf(os.Stderr, "unable to create host directory: %v\n", err)
				os.Exit(1)
			}
		}

		// Inherent variables from the parent process if -E is set
		var envVars []string
		if preserveEnv {
			envVars = os.Environ()
		} else {
			envVars = make([]string, 0, 1)
		}

		// RunDaemon takes the listener(s) as file descriptors via cmd.ExtraFiles
		extraFiles := make([]*os.File, 2)

		// Start listening on the port and pass the descriptor to it
		if listener, err := Listen(listenOn); err != nil {
			fmt.Fprintf(os.Stderr, "unable to open socket: %v\n", err)
			os.Exit(1)
		} else {
			ev := fmt.Sprintf("%s=%s", listenerEnvVarName, listener.Name())
			if verbose {
				fmt.Printf("setting child process environment variable: %s\n", ev)
			}
			envVars = append(envVars, ev)
			extraFiles[0] = listener
		}

		// Create the UNIX domain socket to host the TokenPublisher when token checking is enabled
		if maxTokens > 0 {
			// Remove the stale UNIX domain socket if it still exists
			if err := os.Remove(unixSocketPath); err != nil && !os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "unable to remove unix socket: %v\n", err)
				os.Exit(1)
			}
			// Create a new UNIX domain socket for the child process
			if listener, err := net.Listen("unix", unixSocketPath); err == nil {
				if extraFiles[1], err = listener.(*net.UnixListener).File(); err != nil {
					fmt.Fprintf(os.Stderr, "unable to get unix socket file: %v\n", err)
					os.Exit(1)
				}
			} else {
				fmt.Fprintf(os.Stderr, "unable to listen on unix socket: %v\n", err)
				os.Exit(1)
			}

			if verbose {
				var sb strings.Builder
				sb.WriteString("token checking enabled using ")
				if maxTokens > 1 {
					sb.WriteString(fmt.Sprintf("%d", maxTokens))
				} else {
					sb.WriteString("one")
				}
				if maxTokenUses > 1 {
					sb.WriteString(fmt.Sprintf(", %d use", maxTokenUses))
				} else if maxTokenUses == 1 {
					sb.WriteString(", single use")
				}
				sb.WriteString(" token")
				if maxTokens > 1 {
					sb.WriteString("s")
				}
				sb.WriteString(" with")
				if tokenTimeout > 0 {
					sb.WriteString(fmt.Sprintf(" a %s timeout\n", tokenTimeout))
				} else {
					sb.WriteString(" no timeout\n")
				}
				fmt.Print(sb.String())
			}
		} else if verbose {
			fmt.Println("token checking disabled")
		}

		// Start the child process in the background
		pid := RunDaemon(pidFilePath, userFlag, groupFlag, envVars, extraFiles)

		if verbose {
			fmt.Printf("wrote pid file: %s\nstarted a daemon with PID: %d\nexiting with status 0\n", pidFilePath, pid)
		}
		// Exit the parent process having successfully started the child process
		os.Exit(0)
	} else {
		sigs := make(chan os.Signal, 1)
		// Set a signal handler to remove the UNIX domain socket and PID file before exiting
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigs
			os.Remove(pidFilePath)
			if maxTokens > 0 {
				syscall.Close(3)
				os.Remove(unixSocketPath)
			}
			os.Exit(0)
		}()
		// Open the database using gorm with sqlite3
		gormDb, err := gorm.Open(sqlite.Open(databaseFilePath), &gorm.Config{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "unable to open database '%s': %v\n",
				databaseFilePath, err)
			os.Exit(1)
		}
		// Run the server
		if os.Getenv(listenerEnvVarName) != "" {
			// As the child process by starting the server on the open socket
			r := gin.Default()
			// Use the token checker when running as a daemon
			ttc := NewTokenChecker(maxTokens, maxTokenUses, tokenTimeout)
			if maxTokens > 0 {
				r = TokenCheckerHeader(r, ttc, tokenHeader)
				go func() {
					// Serve the TokenPublisher over a Unix domain socket
					if err := TokenCheckerPublisher(gin.Default(), ttc, tokenEndpointPath).RunFd(4); err != nil {
						fmt.Fprintf(os.Stderr, "unable to serve on unix socket: %v\n", err)
						os.Exit(1)
					}
				}()
			}
			r = DhcpHostDir(LeaseDatabase(r, gormDb), hostDirPath)
			// Gin defaults to DebugMode so set this explicitly
			gin.SetMode(gin.ReleaseMode)
			// Run on the open socket; 3 because 0, 1 and 2 are stdin, stdout and stderr
			if err := r.RunFd(3); err != nil {
				fmt.Fprintf(os.Stderr, "unable to listen on already open socket: %v\n", err)
			}
		} else {
			// Without -d and not as the child process, i.e., running in the foreground
			if err := DhcpHostDir(
				LeaseDatabase(gin.Default(), gormDb), hostDirPath,
			).Run(listenOn); err != nil {
				fmt.Fprintf(os.Stderr, "unable to listen on '%s': %v\n", listenOn, err)
			}
		}
		os.Exit(1)
	}
}
