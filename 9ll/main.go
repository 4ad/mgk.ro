/*
9ll: refactor Plan 9 linkers
	9ll -I includedir [files ...]

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
			// A function declaration. Function prototypes in the
			// middle of a function would probably break our plans,
			// but we hope for the best.
			if x.Type.Is(cc.Func) {
				deps[x] = nil
				curfunc = x
			}
		case *cc.Expr:
			switch x.Op {
			case cc.Addr:
				// We take the address of something; if it's a function,
				// record it.
				if x.Left != nil && x.Left.XDecl != nil {
					if !x.Left.XDecl.Type.Is(cc.Func) {
						return
					}
					for _, v := range deps[curfunc] {
						if x.Left.XDecl == v {
							return
						}
					}
					deps[curfunc] = append(deps[curfunc], x.Left.XDecl)
				}
			case cc.Call:
				// Direct function call. We miss indirect function
				// calls, but that's ok. The plan is to record
				// geting the address of a function too.
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

// extract returns the recursive list of functions called by f
// x.Type.Is(cc.Func) must be true for f, and will be true for subset.
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
