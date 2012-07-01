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
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	_ "code.google.com/p/rbits/log"
)

var (
	fflag   = flag.Int("f", 0, "struct field verbosity.")
	pkgdir  string // user supplied.
	stubdir string // user supplied.
)

func usage() {
	fmt.Fprint(os.Stderr, "usage: gostub [options] pkg stubpkg\n")
	flag.PrintDefaults()
	os.Exit(1)
}

func main() {
	flag.Usage = usage
	flag.Parse()
	if flag.NArg() != 2 {
		usage()
	}
	if *fflag < 0 || 2 < *fflag {
		usage()
	}

	pkgdir = flag.Arg(0)
	stubdir = flag.Arg(1)
	err := filepath.Walk(pkgdir, visitFile)
	if err != nil {
		log.Fatal(err)
	}
}

func visitFile(path string, f os.FileInfo, verr error) (err error) {
	if verr != nil {
		return verr
	}
	target := stubdir + path[len(pkgdir):]
	if f.IsDir() {
		return os.MkdirAll(target, f.Mode())
	}
	if isGoFile(f) {
		return copyGoStubs(path, target)
	}
	return copyFile(path, target)
}

func copyFile(src, dst string) (err error) {
	from, err := os.Open(src)
	if err != nil {
		return
	}
	defer from.Close()

	to, err := os.Create(dst)
	if err != nil {
		return
	}
	defer to.Close()

	_, err = io.Copy(to, from)
	return
}

func isGoFile(f os.FileInfo) bool {
	// ignore non-Go files.
	name := f.Name()
	return !f.IsDir() && !strings.HasPrefix(name, ".") && strings.HasSuffix(name, ".go")
}

func copyGoStubs(src, dst string) (err error) {
	fset, fast, err := compileGoFile(src)
	if err != nil {
		return
	}
	myFilterFile(fast, myExportFilter, true)
	return printGoFile(dst, fset, fast)
}

func compileGoFile(file string) (fset *token.FileSet, fast *ast.File, err error) {
	fset = token.NewFileSet()
	fast, err = parser.ParseFile(fset, file, nil, parser.ParseComments)
	if err != nil {
		return
	}
	return
}

func printGoFile(file string, fset *token.FileSet, fast *ast.File) (err error) {
	f, err := os.Create(file)
	if err != nil {
		return
	}
	defer f.Close()
	return printer.Fprint(f, fset, fast)
}
