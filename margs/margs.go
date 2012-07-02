// Permission to use, copy, modify, and/or distribute this software for
// any purpose is hereby granted, provided this notice appear in all copies.

/*
Marg (map arguments) reads from standard input and for each line read,
it executes the command with the specified arguments followed by the
line that has been read. -I replstr flag causes each instance of replstr
(usually {}) found in the command to be replaced with the line read,
instead of the line being added at the end of the command.

Marg is equivalent to xargs -n1.
*/
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"

	_ "code.google.com/p/rbits/log"
)

var flagI = flag.String("I", "", "replace string (usually {}) with argument")

func usage() {
	fmt.Fprintln(os.Stderr, "usage: margs [-I {}] cmd [args...]")
	os.Exit(1)
}

func makeCmdLine(line string) []string {
	if *flagI == "" {
		return append(flag.Args(), line)
	}
	args := append([]string{}, flag.Args()...)
	for i, arg := range args {
		if arg == *flagI {
			args[i] = line
		}
	}
	return args
}

func makeCmd(line string) *exec.Cmd {
	cmdline := makeCmdLine(line)
	cmd := exec.Command(cmdline[0], cmdline[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}

func readLine(r *bufio.Reader) (string, error) {
	line := []byte{}
	for {
		linefrag, prefix, err := r.ReadLine()
		if err == io.EOF {
			return "", err
		}
		if err != nil {
			log.Fatalf("can't read a line from stdin: %v", err)
		}
		line = append(line, linefrag...)
		if !prefix {
			break
		}
	}
	return string(line), nil
}

func main() {
	flag.Usage = usage
	flag.Parse()
	if flag.NArg() == 0 {
		usage()
	}

	r := bufio.NewReader(os.Stdin)
	for {
		line, err := readLine(r)
		if err == io.EOF {
			break
		}

		cmd := makeCmd(line)
		err = cmd.Run()
		if err == exec.ErrNotFound {
			log.Printf("can't run command for line: %q: %v", line, err)
		}
	}
}
