/*
plan9-ssh: ssh(1) wrapper and devdraw server
	plan9-ssh [ssh-options...] [user@]host [cmd [args...]]
	plan9-ssh [user@]host acme

This tool is a drop-in ssh(1) replacement that is also a local
devdraw server. It exposes the devdraw server to remote clients by
starting plan9-shell on the remote end and forwarding devdraw
connections to itself.

This program wraps ssh(1), so $HOME/.ssh/config is honored, as well
as any extra ssh(1) options passed on the command line.
*/
package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"

	_ "mgk.ro/log"
)

func main() {
	local := tmpfile()
	go serve(local)
	network, addr := cmdsplit(os.Args[1:])
	ssh(network, addr, local, tmpfile()) // different filename, so ssh localhost works.
	os.Remove(local)
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
	exe := os.Getenv("DEVDRAW")
	if exe == "" {
		exe = "devdraw"
	}
	cmd := exec.Command(exe)
	cmd.Stdin = conn
	cmd.Stdout = conn
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		panic(err)
	}
	conn.Close()
}

func ssh(args []string, command string, local, remote string) {
	cmd := exec.Command("ssh", args...)
	cmd.Args = append(cmd.Args,
		"-R", fmt.Sprintf("%s:%s", remote, local),
		"-o", "ExitOnForwardFailure=yes",
		"plan9-shell",
		"-addr", fmt.Sprintf("unix!%s", remote),
	)
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

func tmpfile() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		log.Fatal("can't generate random filename")
	}
	return "/tmp/devdraw-" + hex.EncodeToString(b)
}
