// Permission to use, copy, modify, and/or distribute this software for
// any purpose is hereby granted, provided this notice appear in all copies.

/*
Gofork is a tool to help manage forked packages.

Usage:
    gofork old new
    gofork -r 'old=new' [packages ...]

In its first invocation gofork creates a fork of the package tree
rooted at the old import path. Packages that wish to use the fork
should use the new import path. gofork potentially updates the
import paths of the packages in the tree in order to reference
the forked packages. Forked packages will be installed where go
get would install packages.

In its second form gofork will update the named packages to
change old import paths to the new fork. It works recursively,
if you forked:

    gofork net/http myhttp

and you wish to use myhttp and myhttp/httputil in godoc, it
suffices to:

    gofork -r 'net/http=myhttp' cmd/godoc

net/http will be changed to myhttp and net/http/httputil will be
changed to myhttp/httputil.
*/
package main

import (
	"flag"
	"fmt"
	"go/build"
	"os"
	"path/filepath"
)

var (
	flagR = flag.String("r", "", "update named packages to use the new import path")
)

var usageString = `usage:
    gofork old new
    gofork -r 'old=new' [packages ...]

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
	if flag.NArg() != 2 {
		usage()
	}
}

// srcDir returns the directory where the package with the named
// import path resides.
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
	fmt.Fprint(os.Stderr, "gofork: ")
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

func log(args ...interface{}) { logf("%v", args...) }

func fatalf(format string, args ...interface{}) {
	logf(format, args...)
	os.Exit(2)
}

func fatal(args ...interface{}) { fatalf("%v", args...) }
