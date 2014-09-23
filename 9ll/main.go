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

// deps is the dependency graph between symbols.
var deps = map[*cc.Decl][]*cc.Decl{}

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
	depGraph(prog)
	subset := append(extract(lookup("span")), extract(lookup("asmb"))...)
	for _, v := range subset {
		fmt.Printf("%s	%s\n", v.Name, v.GetSpan())
	}
}

func depGraph(prog *cc.Prog) {
	var curfunc *cc.Decl
	cc.Preorder(prog, func(x cc.Syntax) {
		switch x := x.(type) {
		case *cc.Decl:
			if x.Type.Is(cc.Func) {
				deps[x] = nil
				curfunc = x
			}
		case *cc.Expr:
			if x.Op == cc.Call {
				for _, v := range deps[curfunc] {
					if x.Left.XDecl == v {
						return
					}
				}
				deps[curfunc] = append(deps[curfunc], x.Left.XDecl)
			}
		}
	})
}

func lookup(name string) *cc.Decl {
	for s := range deps {
		if s.Name == name {
			return s
		}
	}
	return nil
}

func extract(f *cc.Decl) (subset []*cc.Decl) {
	var r func(f *cc.Decl)
	r = func(f *cc.Decl) {
		for _, s := range subset {
			if s == f {
				return
			}
		}
		subset = append(subset, f)
		for _, d := range deps[f] {
			r(d)
		}
	}
	r(f)
	return
}
