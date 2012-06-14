// Permission to use, copy, modify, and/or distribute this software for
// any purpose is hereby granted, provided this notice appear in all copies.

/*
Gostub creates a stub package that exports the same public interface as
the source package.

Usage:
	gostub pkg stubpkg

It works recursively and ignores non-Go files. Package names are not
changed. The file structure of the stub is the same as the original. Pkg
and stubpkg are directories, not import paths.
*/
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func usage() {
	fmt.Fprint(os.Stderr, "usage: gostub pkg stubpkg\n")
	os.Exit(1)
}

func main() {
	flag.Usage = usage
	flag.Parse()
	if flag.NArg() != 2 {
		usage()
	}

	err := filepath.Walk(flag.Arg(0), visitFile)
	if err != nil {
		fatal(err)
	}
}

func visitFile(path string, info os.FileInfo, verr error) (err error) {
	return
}

func isGoFile(f os.FileInfo) bool {
	// ignore non-Go files.
	name := f.Name()
	return !f.IsDir() && !strings.HasPrefix(name, ".") && strings.HasSuffix(name, ".go")
}

func logf(format string, args ...interface{}) {
	fmt.Fprint(os.Stderr, "gostub: ")
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

func log(args ...interface{}) { logf("%v", args...) }

func fatalf(format string, args ...interface{}) {
	logf(format, args...)
	os.Exit(2)
}

func fatal(args ...interface{}) { fatalf("%v", args...) }
