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

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	_ "mgk.ro/log"
)

func main() {
	tmp, err := ioutil.TempFile("", "")
	if err != nil {
		log.Fatal(err)
	}
	unix := tmp.Name()
	unix = filepath.Join("/tmp", filepath.Base(unix))
	os.Remove(tmp.Name())
	go serve(unix)
	network, addr := cmdsplit(os.Args[1:])
	ssh(network, addr, unix)
	os.Remove(unix)
}

func serve(name string) {
	l, err := net.Listen("unix", name)
	if err != nil {
		log.Fatal(err)
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			panic(err)
		}
		go devdraw(conn)
	}
}

func devdraw(conn net.Conn) {
	cmd := exec.Command("devdraw")
	cmd.Stdin = conn
	cmd.Stdout = conn
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		panic(err)
	}
}

func ssh(args []string, command string, unix string) {
	cmd := exec.Command("ssh", args...)
	cmd.Args = append(cmd.Args, "-M", "-S", "none", "-R",
		fmt.Sprintf("%s:%s", unix, unix),
	)
	cmd.Args = append(cmd.Args, "plan9-shell")
	cmd.Args = append(cmd.Args, "-addr", fmt.Sprintf("unix!%s", unix))
	if command != "" {
		cmd.Args = append(cmd.Args, "-c", command)
	} else {
		pre := []string{cmd.Args[0], "-t"}
		cmd.Args = append(pre, cmd.Args[1:]...)
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
}

// cmdsplit splits the ssh args by isolating the command (if any).
func cmdsplit(args []string) (params []string, cmd string) {
	lastNonFlag := -1
	dash := false
	for k, v := range args {
		if strings.HasPrefix(v, "-") {
			dash = true
		} else {
			if lastNonFlag != k-1 {
				return args[:k+1], strings.Join(args[k+1:], " ")
			}
			lastNonFlag = k
		}
	}
	if !dash {
		return []string{args[0]}, strings.Join(args[1:], " ")
	}
	return args, ""
}
