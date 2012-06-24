// Permission to use, copy, modify, and/or distribute this software for
// any purpose is hereby granted, provided this notice appear in all copies.

/*
Goembed reads each file in sequence and writes on the standard output
Go source code that wraps the file's contents in a []byte. If no file
is given, goembed reads from the standard input.
*/
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
)

func usage() {
	fmt.Fprint(os.Stderr, "usage: goembed [files]\n")
	os.Exit(1)
}

func main() {
	flag.Usage = usage
	flag.Parse()

	files := flag.Args()
	if len(files) == 0 {
		files = append(files, os.Stdin.Name())
	}
	for _, name := range files {
		b, err := ioutil.ReadFile(name)
		if err != nil {
			fmt.Fprint(os.Stderr, "goembed: %v\n", err)
			os.Exit(2)
		}
		fmt.Printf("var %s = %#v\n\n", path.Base(name), b)
	}
}
