// Permission to use, copy, modify, and/or distribute this software for
// any purpose is hereby granted, provided this notice appear in all copies.

package main

import (
	"flag"
	"fmt"
	"go/build"
	"os"
	"path/filepath"
)

var (
	flagR = flag.String("r", "", "")
)

func usage() {
	os.Exit(1)
}

func main() {
	flag.Usage = usage
	flag.Parse()
	if flag.NArg() != 2 {
		usage()
	}
}

// srcDir returns the directory where the package with the named
// import path resides. It is required for resolving local imports (ugh).
func srcDir(path string) string {
	// Check if it's a command in $GOROOT/src, like cmd/go.
	cmdpath := filepath.Join(build.Default.GOROOT, "src", path)
	// normally we'd use build.ImportDir, but it has a bug.
	fi, err := os.Stat(cmdpath)
	if err != nil || !fi.IsDir() {
		// A regular package in $GOROOT/src/pkg or in any $GOPATH/src.
		pkg, err := build.Import(path, "", build.FindOnly)
		if err != nil {
			fatal(err)
		}
		return pkg.Dir
	}
	return cmdpath
}

func logf(format string, args ...interface{}) {
	fmt.Fprint(os.Stderr, "godep: ")
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

func log(args ...interface{}) { logf("%v", args...) }

func fatalf(format string, args ...interface{}) {
	logf(format, args...)
	os.Exit(2)
}

func fatal(args ...interface{}) { fatalf("%v", args...) }
