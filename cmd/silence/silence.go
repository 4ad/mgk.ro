package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

var (
	quiet  = flag.Bool("q", false, "be extra quiet")
	prefix = flag.String("x", "", "use prefix for output file")
)

func usage() {
	fmt.Fprint(os.Stderr, "usage: silence [ options ... ] cmd [ args ... ]\n")
	flag.PrintDefaults()
	os.Exit(1)
}

func main() {
	flag.Usage = usage
	flag.Parse()
	if flag.NArg() == 0 {
		usage()
	}
	cmd, out := command(*prefix, flag.Args()...)
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "silence: can't start: %v\n", err)
		os.Remove(out.Name())
		os.Exit(4)
	}
	if !*quiet {
		fmt.Printf("tail -f %s # pid=%d\n", out.Name(), cmd.Process.Pid)
	}
	if err := cmd.Wait(); err != nil {
		var exiterr *exec.ExitError
		if errors.As(err, &exiterr) {
			os.Exit(exiterr.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "silence: unexpected: %v\n", err)
		os.Exit(8)
	}
	os.Remove(out.Name())
}

func command(pre string, cmds ...string) (*exec.Cmd, *os.File) {
	out := outfile(pre, cmds[0])
	cmd := exec.Command(cmds[0], cmds[1:]...)
	cmd.Stdout = out
	cmd.Stderr = out
	return cmd, out
}

func outfile(pre, exe string) *os.File {
	out, err := os.CreateTemp("", pattern(pre, exe))
	if err != nil {
		fmt.Fprintf(os.Stderr, "silence: can't create output file: %v\n", err)
		os.Exit(2)
	}
	return out
}

func pattern(pre, exe string) string {
	if pre == "" {
		return filepath.Base(exe) + "-*.out"
	}
	return pre + "-*.out"
}
