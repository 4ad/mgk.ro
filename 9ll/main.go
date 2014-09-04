/*
9ll: refactor Plan 9 linkers
	9ll [files ...]

9ll helps refactor Plan 9 linkers into liblink form used by Go.
*/
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"code.google.com/p/rsc/cc"

	_ "code.google.com/p/rbits/log"
)

var (
	inc = flag.String("I", "", "include directory")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: 9ll [flags] files ...\n")
		flag.PrintDefaults()
		os.Exit(1)
	}
	flag.Parse()
	args := flag.Args()
	if *inc != "" {
		cc.AddInclude(*inc)
	}
	if len(args) == 0 {
		flag.Usage()
	}
	var r []io.Reader
	files := args
	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			log.Fatal(err)
		}
		r = append(r, f)
		defer f.Close()
	}
	prog, err := cc.ReadMany(files, r)
	if err != nil {
		log.Fatal(err)
	}
	_ = prog
}
