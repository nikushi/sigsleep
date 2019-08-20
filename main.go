// https://github.com/weaveworks/weave/blob/master/prog/sigproxy/main.go
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

const usage = `USAGE: sigproxy [options] -- <command> [arguments ...]

sigproxy - wrapper command to proxy signals to command process and inject sleep around command execution.

Options:
	-after=n     Sleep n seconds after the command exits.
`

func printUsage() {
	fmt.Fprintf(os.Stderr, usage)
}

func main() {
	var optAfter int

	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags.Usage = func() { printUsage() }
	flags.IntVar(&optAfter, "after", 0, "Sleep n seconds after command exit")
	if err := flags.Parse(os.Args[1:]); err != nil {
		flags.Usage()
		os.Exit(1)
	}

	args := flags.Args()
	if len(args) == 0 {
		flags.Usage()
		os.Exit(1)
	}

	// Install signal handler as soon as possible - channel is buffered so
	// we'll catch signals that arrive whilst child process is starting
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)

	cmd := exec.Command(args[0], args[1:]...)

	// These default to /dev/null, so set them explicitly to ours
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	// Only begin delivering signals after the child has started
	go func() {
		for {
			// Signalling PID 0 delivers to our process group
			syscall.Kill(0, (<-sc).(syscall.Signal))
		}
	}()

	if err := cmd.Wait(); err != nil {
		// Exit status is platform specific so not directly accessible - casts
		// required to access system-dependent exit information
		if exitErr, ok := err.(*exec.ExitError); ok {
			waitStatus := exitErr.Sys().(syscall.WaitStatus)
			os.Exit(waitStatus.ExitStatus())
		}
		os.Exit(1)
	}

	// sleep n after finished the command execution
	time.Sleep(time.Duration(optAfter) * time.Second)

	os.Exit(0)
}
