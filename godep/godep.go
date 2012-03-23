// Permission to use, copy, modify, and/or distribute this software for
// any purpose is hereby granted, provided this notice appear in all copies.

// godep returns package dependency information.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"

//	"go/build"
)

var (
	dot  = flag.Bool("dot", false, "print DOT language (GraphWiz)")
	png  = flag.String("png", "", "write graph to png file")
	tags = flag.String("tags", "", "additional built tags to consider")
)

var packages []string

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

// golist runs 'go list args' and assigns packages the result.
func golist(args ...string) {
	args = append([]string{"list"}, args...)
	cmd := exec.Command("go", args...)
	cmd.Stderr = os.Stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fatal(err)
	}
	r := bufio.NewReader(stdout)
	
	if err = cmd.Start(); err != nil {
		fatal(err)
	}
	for {
		pkg, _, err := r.ReadLine()
		if err != nil {
			break
		}
		packages = append(packages, string(pkg))
	}
	if err = cmd.Wait(); err != nil {
		os.Exit(1)
	}
	return
}

func main() {
	flag.Usage = usage
	flag.Parse()

	golist(flag.Args()...)
	for _, v := range packages {
		fmt.Println(v)
	}
}
