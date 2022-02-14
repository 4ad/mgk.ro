/*
Lsr: list recursively.
	lsr [-d | -fd] [-l] [-a] [name ...]

For each directory argument, lsr recursively lists the contents of
the directory; for each file argument, lsr repeats its name. When
no argument is given, the current directory is listed. Hidden files
are ignored by default.

Options:
    -d  only print directories
    -fd print both files and directories
    -l  list in long format; name mode mtime size
    -a print hidden files or directories
*/
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "mgk.ro/log"
)

var (
	flagD  = flag.Bool("d", false, "only print directories")
	flagFD = flag.Bool("fd", false, "print both files and directories")
	flagL  = flag.Bool("l", false, "long format")
	flagA  = flag.Bool("a", false, "print hidden files")
)

var usageString = `usage: lsr [-d | -fd] [-l] [-a] [name ...]
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
	if !*flagA {
		ok, err := filepath.Match(".?*", filepath.Base(path))
		if ok {
			return nil
		}
		if err != nil {
			return err
		}
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
	if !*flagA && f.IsDir() {
		ok, err := filepath.Match(".?*", filepath.Base(path))
		if ok {
			return filepath.SkipDir
		}
		if err != nil {
			return err
		}
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
