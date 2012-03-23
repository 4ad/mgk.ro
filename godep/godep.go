// Permission to use, copy, modify, and/or distribute this software for
// any purpose is hereby granted, provided this notice appear in all copies.

// godep returns package dependency information.
package main

import (
	"flag"
	"fmt"
	"os"
	"go/build"
)

var (
	dot  = flag.Bool("dot", false, "print DOT language (GraphWiz)")
	png  = flag.String("png", "", "write graph to png file")
	tags = flag.String("tags", "", "additional built tags to consider")
)

var usageString = `usage: godep [options] [packages]

Godep prints dependency information for packages named by the import
paths. If no packages are specified, the current directory is assumed
to be the package. The special import path "all" expands to all package
directories found in all the GOPATH trees. The special import path "std"
is like all but expands to just the packages in the standard Go library.

For more about specifying packages, see 'go help packages'.

Options:`

func usage() {
	fmt.Fprintf(os.Stderr, "%s\n", usageString)
	flag.PrintDefaults()
	os.Exit(1)
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

func main() {
	flag.Usage = usage
	flag.Parse()
	
	args := flag.Args()
	for _, v := range args {
		switch {
		case v == "all":
			allDirs := build.Default.SrcDirs()
			args = append(args, allDirs...)
			continue
		case v == "std":
			ctxt := build.Default
			ctxt.GOPATH = ""
			stdDirs := ctxt.SrcDirs()[0]
			args = append(args, stdDirs)
			continue
		}
		bfs(v)
	}
}

// bfs does a breadth-first traversal of the package tree rooted at dir.
func bfs(dir string) {
	
}
