package main

import (
	"fmt"
	"go/build"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
)

const (
	targetGoroot = "/export/home/aram/go"
	targetGopath = "/tmp/gopath"
	targetGotmp  = "/tmp/go.emul"
	targetHost   = "daffodil.mgk.ro"
)

func ssh(args ...string) error {
	cmd := exec.Command("ssh", append([]string{targetHost}, args...)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func scp(file, dir string) {
	cmd := exec.Command("scp", file, targetHost+":"+dir)
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatalf("scp %s %s:%s: %v", file, targetHost, dir, err)
	}
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("go_solaris_sparc64_exec: ")

	ssh("mkdir", "-p", targetGotmp)

	// Determine the package by examining the current working
	// directory, which will look something like
	// "$GOROOT/src/mime/multipart" or "$GOPATH/src/golang.org/x/mobile".
	// We extract everything after the $GOROOT or $GOPATH to run on the
	// same relative directory on the targetHost host.
	subdir, inGoRoot := subdir()
	targetCwd := filepath.Join(targetGoroot, subdir)
	if !inGoRoot {
		targetCwd = filepath.Join(targetGopath, subdir)
	}

	binName := filepath.Base(os.Args[1])
	targetBin := fmt.Sprintf("%s/%s", targetGotmp, binName)
	scp(os.Args[1], targetBin)

	cmd := `export TMPDIR="` + targetGotmp + `"` +
		`; export GOROOT="` + targetGoroot + `"` +
		`; export GOPATH="` + targetGopath + `"` +
		`; export GOTRACEBACK="system"` +
		`; export GOOS="solaris"` +
		`; export GOARCH="sparc64"` +
		//		`; export GODEBUG="gcstackbarrierall=1"` +
		`; cd "` + targetCwd + `"` +
		"; '" + targetBin + "' " + strings.Join(os.Args[2:], " ")
	err := ssh(cmd)
	if err == nil {
		os.Exit(0)
	}
	if exiterr, ok := err.(*exec.ExitError); ok {
		if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
			os.Exit(status.ExitStatus())
		}
	}
	log.Fatal(err)
}

// subdir determines the package based on the current working directory,
// and returns the path to the package source relative to $GOROOT (or $GOPATH).
func subdir() (pkgpath string, underGoRoot bool) {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	if root := runtime.GOROOT(); strings.HasPrefix(cwd, root) {
		subdir, err := filepath.Rel(root, cwd)
		if err != nil {
			log.Fatal(err)
		}
		return subdir, true
	}

	for _, p := range filepath.SplitList(build.Default.GOPATH) {
		if !strings.HasPrefix(cwd, p) {
			continue
		}
		subdir, err := filepath.Rel(p, cwd)
		if err == nil {
			return subdir, false
		}
	}
	return "/test", true
}
