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
	"path"
	"runtime"
	"strings"

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

// symbols to start from
var start = []string{
	"span",
	"asmout",
	"chipfloat",
	"follow",
	"noops",
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
	var syms []*cc.Decl
	for _, v := range start {
		syms = append(syms, lookup(v))
	}
	subset := extract(syms...)

	if os.RemoveAll("liblink") != nil {
		log.Fatal(err)
	}
	if os.MkdirAll("liblink", 0775) != nil {
		log.Fatal(err)
	}
	file := make(map[string]*os.File)
	for _, v := range subset {
		if !strings.Contains(v.Span.String(), "7l") {
			continue
		}
		f, ok := file[v.Span.Start.File]
		if !ok {
			f, err = os.Create("liblink/" + path.Base(v.Span.Start.File))
			if err != nil {
				log.Fatal(err)
			}
			defer f.Close()
			file[v.Span.Start.File] = f
			f.WriteString("//+build ignore\n\n")
		}
		var pp cc.Printer
		pp.Print(v)
		f.Write(pp.Bytes())
		f.WriteString("\n\n")
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

// extract returns the recursive list of functions called by the fns
// x.Type.Is(cc.Func) must be true for fns, and will be true for subset.
func extract(fns ...*cc.Decl) (subset []*cc.Decl) {
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
	for _, v := range fns {
		r(v)
	}
	return
}
