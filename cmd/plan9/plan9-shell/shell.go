// Copyright (c) 2019 Aram Hăvărneanu <aram@mgk.ro>
//
// Permission to use, copy, modify, and distribute this software for any
// purpose with or without fee is hereby granted, provided that the above
// copyright notice and this permission notice appear in all copies.
//
// THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
// WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
// MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
// ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
// WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
// ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
// OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.

/*
plan9-shell: Unix shell wrapper
	plan9-shell -addr addr [-c cmd]

This tool wraps the user's SHELL and sets some variables useful to
plan9port programs. It will set DEVDRAW_SERVER=addr, and
DEVDRAW=devdraw-proxy. If -c is present, rather than start an
interactive shell, it will pass cmd to the user's shell to execute.

This program is not intended to be called by the user, but by
plan9-ssh.
*/
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"mgk.ro/cmd/net/netutil"
	_ "mgk.ro/log"
)

var addr = flag.String("addr", "", "network address of the drawterm server")
var cmd = flag.String("c", "", "shell command to execute")

var usageString = `usage: plan9-shell -addr addr [-c cmd]
Options:
`

func usage() {
	fmt.Fprint(os.Stderr, usageString)
	flag.PrintDefaults()
	os.Exit(1)
}

func main() {
	flag.Usage = usage
	flag.Parse()

	if *addr == "" {
		usage()
	}
	defer cleanup(*addr)

	shell := exec.Command(os.Getenv("SHELL"))
	shell.Env = append(os.Environ(),
		fmt.Sprintf("DEVDRAW_SERVER=%s", *addr),
		"DEVDRAW=devdraw-proxy",
	)
	if *cmd == "" {
		shell.Args[0] = "-" + filepath.Base(shell.Args[0])
	} else {
		shell.Args = append(shell.Args, "-c", *cmd)
	}
	shell.Stdin = os.Stdin
	shell.Stdout = os.Stdout
	shell.Stderr = os.Stderr
	if err := shell.Run(); err != nil {
		cleanup(*addr)
		// If the process starts, but returns an error, propagate
		// it further without logging.
		if err, ok := err.(*exec.ExitError); ok {
			// By exiting with the error code from the shell,
			// we pass it further to our caller (usually ssh(1)).
			os.Exit(err.ProcessState.ExitCode())
		}
		log.Fatal(err)
	}
}

// Cleanup attempts to remove the unix domain socket left over by
// ssh(1). It's a best effort function, it doesn't complain about
// encountered errors because it's too late to do anything about them.
func cleanup(s string) {
	net, host, _ := netutil.SplitDialString(s)
	if net != "unix" { // only attempt to remove unix domain sockets
		return
	}
	os.Remove(host)
}
