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
	"runtime"

	"code.google.com/p/rsc/cc"

	_ "code.google.com/p/rbits/log"
)

// missing *.h pstate.c main.c.
var files = []string{
	"dyn.c",
	"sub.c",
	"mod.c",
	"list.c",
	"noop.c",
	"elf.c",
	"pass.c",
	"pobj.c",
	"asm.c",
	"optab.c",
	"obj.c",
	"span.c",
	"asmout.c",
}

// deps is the dependency graph between symbols.
var deps = map[*cc.Decl][]*cc.Decl{}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: 9ll [files] ...\n")
		fmt.Fprintf(os.Stderr, "\tdefault files are from $GOOROT/src/cmd/7l")
		os.Exit(1)
	}
	flag.Parse()
	args := flag.Args()
	if len(args) != 0 {
		files = args
	} else {
		for k, v := range files {
			files[k] = runtime.GOROOT() + "/src/cmd/7l/" + v
		}
	}
	var r []io.Reader
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
			// Potentially indirect function dependency by taking a
			// function's address, which can be both &foo and foo.
			case cc.Addr, cc.Name:
				if x.Left == nil || x.Left.XDecl == nil {
					return
				}
				if !x.Left.XDecl.Type.Is(cc.Func) {
					return
				}
				for _, v := range deps[curfunc] {
					if x.Left.XDecl == v {
						return
					}
				}
				deps[curfunc] = append(deps[curfunc], x.Left.XDecl)
				fmt.Println(x.Left.XDecl.Name)
			// Direct function call.
			case cc.Call:
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
