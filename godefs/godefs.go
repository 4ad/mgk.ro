// Permission to use, copy, modify, and/or distribute this software for
// any purpose is hereby granted, provided this notice appear in all copies.

/*
Godefs prints a top-down description of the top-level type definitions
for packages named by the import paths.

Usage:
	godefs [packages]

By default it prints type definitions for the package in the current
directory. For more about specifying packages, see 'go help packages'.\
*/
package main

import (
	"bufio"
	"flag"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	bldCtxt = build.Default
	pkgs    []string // user supplied.
)

func usage() {
	fmt.Fprint(os.Stderr, "usage: godefs [packages]\n")
	os.Exit(1)
}

func main() {
	flag.Usage = usage
	flag.Parse()

	golist(flag.Args()...) // finds packages to work with.
	for _, pkg := range pkgs {
		fset := token.NewFileSet()
		asts, err := parser.ParseDir(fset, srcDir(pkg), isGoFile, parser.ParseComments)
		if err != nil {
			fatal(err)
		}
		for _, ast := range asts {
			// BUG(aram): exclude _test packages.
			recordDefs(ast, fset)
		}
	}
}

func isGoFile(f os.FileInfo) bool {
	// ignore non-Go files
	name := f.Name()
	return !f.IsDir() && !strings.HasPrefix(name, ".") && strings.HasSuffix(name, ".go")
}

// recordDefs traverses the ast and records the type definitions.
func recordDefs(pkg *ast.Package, fset *token.FileSet) {
	f := ast.MergePackageFiles(pkg, ast.FilterImportDuplicates|ast.FilterFuncDuplicates)
	for _, v := range f.Decls {
		if decl, ok := v.(*ast.GenDecl); ok {
			if decl.Tok == token.TYPE {
				pos := fset.Position(decl.Pos())
				fmt.Printf("//line %v:%v\n", pos.Filename, pos.Line)
				printer.Fprint(os.Stdout, fset, decl)
				fmt.Printf("\n\n")
			}
		}
	}
}

// golist runs 'go list args' and assigns the result to pkgs.
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
		pkgs = append(pkgs, string(pkg))
	}
	if err = cmd.Wait(); err != nil {
		os.Exit(1)
	}
	return
}

// srcDir returns the directory where the package with the named
// import path resides.
func srcDir(path string) string {
	// Check if it's a command in $GOROOT/src, like cmd/go.
	// Normally we'd use build.ImportDir, but it has a bug (3428),
	// so we check for the existence of the corresponding directory
	// in $GOROOT.
	cmdpath := filepath.Join(bldCtxt.GOROOT, "src", path)
	fi, err := os.Stat(cmdpath)
	if err == nil && fi.IsDir() {
		return cmdpath // found a command.
	}

	// A regular package in $GOROOT/src/pkg or in any $GOPATH/src.
	pkg, err := bldCtxt.Import(path, "", build.FindOnly)
	if err != nil {
		fatal(err)
	}
	return pkg.Dir
}

func logf(format string, args ...interface{}) {
	fmt.Fprint(os.Stderr, "godefs: ")
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

func log(args ...interface{}) { logf("%v", args...) }

func fatalf(format string, args ...interface{}) {
	logf(format, args...)
	os.Exit(2)
}

func fatal(args ...interface{}) { fatalf("%v", args...) }
