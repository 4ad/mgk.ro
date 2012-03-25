// Permission to use, copy, modify, and/or distribute this software for
// any purpose is hereby granted, provided this notice appear in all copies.

package main

import (
	"flag"
	"fmt"
	"os"
)

var (
	flagD = flag.Bool("d", false, "also print directories")
	flagT = flag.Bool("t", false, "print Unix time")
)

var usageString=`usage: lsr [-dt] [name ...]

For each directory argument, lsr recursively lists the contents of the
directory; for each file argument, lsr repeats its name and any other
information requested. When no argument is given, the current directory
is listed.

Options:`

func usage() {
	fmt.Fprintf(os.Stderr, "%s\n", usageString)
	flag.PrintDefaults()
	os.Exit(1)
}

func main() {
	flag.Usage = usage
	flag.Parse()
}
