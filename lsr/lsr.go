// Permission to use, copy, modify, and/or distribute this software for
// any purpose is hereby granted, provided this notice appear in all copies.

/*
Lsr: list recursively.
	lsr [-d | -fd] [name ...]

For each directory argument, lsr recursively lists the contents of the
directory; for each file argument, lsr repeats its name. When no argument
is given, the current directory is listed.

Options:
	-d  only print directories
	-fd	print both files and directories
	-l  list in long format; name mode mtime size
*/
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "code.google.com/p/rbits/log"
)

var (
	flagD  = flag.Bool("d", false, "only print directories")
	flagFD = flag.Bool("fd", false, "print both files and directories")
	flagL  = flag.Bool("l", false, "long format")
)

var usageString = `usage: lsr [-d | -fd | -l] [name ...]
Options:
`

func usage() {
	fmt.Fprint(os.Stderr, usageString)
	flag.PrintDefaults()
	os.Exit(1)
}

func prname(path string, f os.FileInfo) error {
	if f.IsDir() && path[len(path)-1] != '/' {
		path = path + "/"
	}
	if *flagL {
		mode := f.Mode() & 0x1FF
		size := f.Size()
		mtime := f.ModTime().Unix()
		fmt.Printf("%s %o %v %v\n", path, mode, mtime, size)
		return nil
	}
	fmt.Println(path)
	return nil
}

func pr(path string, f os.FileInfo, err error) error {
	if err != nil {
		log.Println(err)
		return nil
	}
	if *flagFD {
		return prname(path, f)
	}
	if *flagD && f.IsDir() {
		return prname(path, f)
	}
	if !*flagD && !f.IsDir() {
		return prname(path, f)
	}
	return nil
}

func main() {
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() == 0 {
		filepath.Walk(".", pr)
		return
	}
	for _, v := range flag.Args() {
		filepath.Walk(v, pr)
	}
}
