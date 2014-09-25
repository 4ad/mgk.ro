/*
7lfix: refactor arm64 Plan 9 linker
	7lfix [files ...]

7lfix helps refactor Plan 9 linkers into liblink form used by Go.
*/
package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"code.google.com/p/rsc/cc"

	_ "code.google.com/p/rbits/log"
)

const (
	theChar = 7
	ld      = "7l"
	lddir   = "/src/cmd/" + ld
)

// iomap maps each input file to its corresponding output file.
// Unknown files go to zzz.c.
// missing *.h pstate.c main.c.
var iomap = map[string]string{
	"dyn.c":    "asm7.c",
	"sub.c":    "xxx.c",
	"mod.c":    "xxx.c",
	"list.c":   "list7.c",
	"noop.c":   "obj7.c",
	"elf.c":    "xxx.c",
	"pass.c":   "obj7.c",
	"pobj.c":   "xxx.c",
	"asm.c":    "asm7.c",
	"optab.c":  "asm7.c",
	"obj.c":    "obj7.c",
	"span.c":   "asm7.c",
	"asmout.c": "asm7.c",
}

// symbols to start from
var start = map[string]bool{
	"span": true,
	"asmout": true,
	"chipfloat": true,
	"follow": true,
	"noops": true,
	"listinit": true,
}

// symbols to be renamed.
var rename = map[string]string{
	"span":      "span7",
	"chipfloat": "chipfloat7",
	"listinit":  "listinit7",
}

var includes = `#include <u.h>
#include <libc.h>
#include <bio.h>
#include <link.h>
#include "../cmd/7l/7.out.h"
`

// symbols is a symbol table.
type symbols map[*cc.Decl][]*cc.Decl

var (
	deps = symbols{} // deps is the dependency graph between wanted symbols
	all  = symbols{} // all are all the symbols, for quick lookup

	symsfile = map[string]map[*cc.Decl]bool{} // maps files to symbols used
)

// replace unqualified names in iomap with full paths.
func init() {
	for k, v := range iomap {
		iomap[runtime.GOROOT()+lddir+"/"+k] = v
		delete(iomap, k)
	}
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: 7lfix\n")
		fmt.Fprintf(os.Stderr, "\tfiles are from $GOOROT"+lddir)
		os.Exit(1)
	}
	if flag.NArg() != 0 {
		flag.Usage()
	}
	prog := parse()
	all = symtab(prog)
	deps = dep(prog, all)
	var syms []*cc.Decl
	for k := range start {
		syms = append(syms, deps.lookup(k))
	}
	subset := extract(syms...)
	print(subset, "l.0")
	symsfile = symfile(prog, all)
	static(subset, symsfile)
	print(subset, "l.1")
	ren(prog, deps)
	print(subset, "l.2")
	diff()
}

// parse opens and parses all input files, and returns the result as
// a *cc.Prog.
func parse() *cc.Prog {
	var r []io.Reader
	var files []string
	for name, _ := range iomap {
		f, err := os.Open(name)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		r = append(r, f)
		files = append(files, name)
	}
	prog, err := cc.ReadMany(files, r)
	if err != nil {
		log.Fatal(err)
	}
	return prog
}

// print pretty prints fns (for which x.Type.Is(cc.Func) must be true)
// into dir.
func print(fns []*cc.Decl, dir string) {
	err := os.RemoveAll(dir)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0775); err != nil {
		log.Fatal(err)
	}
	file := make(map[string]*os.File)
	for _, v := range fns {
		if !strings.Contains(v.Span.String(), ld) {
			continue
		}
		name, ok := iomap[v.Span.Start.File]
		if !ok {
			if strings.Contains(v.Span.Start.File, ".h") {
				name = "l.h"
			} else {
				name = "zzz.c"
			}
		}
		f, ok := file[name]
		if !ok {
			// fmt.Printf("%v:	%v\n", dir + "/" + name, file)
			f, err = os.Create(dir + "/" + name)
			if err != nil {
				log.Fatal(err)
			}
			defer f.Close()
			file[name] = f
			f.WriteString("//+build ignore\n\n")
			if strings.Contains(v.Span.Start.File, ".c") {
				f.WriteString(includes)
				f.WriteString("\n")
			}
		}
		var pp cc.Printer
		pp.Print(v)
		f.Write(pp.Bytes())
		f.WriteString("\n\n")
	}
}

func dep(prog *cc.Prog, all symbols) symbols {
	var deps = symbols{}
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
			// Using a name for a function address.
			case cc.Name:
				xfn := all.lookup(x.Text)
				if xfn == nil {
					return
				}
				for _, v := range deps[curfunc] {
					if xfn == v {
						return
					}
				}
				deps[curfunc] = append(deps[curfunc], xfn)
			// Take a function's address.
			case cc.Addr:
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
	return deps
}

// symtab creates a symbol table.
func symtab(prog *cc.Prog) symbols {
	s := symbols{}
	cc.Preorder(prog, func(x cc.Syntax) {
		d, ok := x.(*cc.Decl)
		if !ok {
			return
		}
		if !d.Type.Is(cc.Func) {
			return
		}
		s[d] = nil
	})
	return s
}

func (st symbols) lookup(name string) *cc.Decl {
	for s := range st {
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

// ren renames some symbols for liblink. This is a trivial operation and
// it's easier to do by hand than to automate here. We do it here however
// to learn more about rsc/cc and as a proof of concept.
//
// Rename is in place because it's a PITA otherwise.
func ren(prog *cc.Prog, syms symbols) {
	for s := range syms {
		if _, ok := rename[s.Name]; !ok {
			continue
		}
		s.Name = rename[s.Name]
	}
	cc.Preorder(prog, func(x cc.Syntax) {
		e, ok :=  x.(*cc.Expr)
		if !ok {
			return
		}
		if e.Op != cc.Name {
			return
		}
		if _, ok := rename[e.Text]; !ok {
			return
		}
		e.Text = rename[e.Text]
	})
}

// diff generates diffs between transformations, so we can see what we
// are doing.
func diff() {
	out, _ := exec.Command("diff", "-urp", "l.0", "l.1").Output()
	if err := ioutil.WriteFile("d01.patch", out, 0665); err != nil {
		log.Fatal(err)
	}
	out, _ = exec.Command("diff", "-urp", "l.1", "l.2").Output()
	if err := ioutil.WriteFile("d12.patch", out, 0665); err != nil {
		log.Fatal(err)
	}
}

func symfile(prog *cc.Prog, all symbols) map[string]map[*cc.Decl]bool {
	syms := map[string]map[*cc.Decl]bool{}
	var curfunc *cc.Decl
	cc.Preorder(prog, func(x cc.Syntax) {
		var xfn *cc.Decl
		switch x := x.(type) {
		case *cc.Decl:
			if x.Type.Is(cc.Func) {
				curfunc = x
			}
		case *cc.Expr:
			switch x.Op {
			// Using a name for a function address.
			case cc.Name:
				xfn = all.lookup(x.Text)
			// Take a function's address.
			case cc.Addr:
				if x.Left == nil || x.Left.XDecl == nil {
					return
				}
				if !x.Left.XDecl.Type.Is(cc.Func) {
					return
				}
				xfn = x.Left.XDecl
			// Direct function call.
			case cc.Call:
				xfn = x.Left.XDecl
			}
		}
		if xfn == nil {
			return
		}
		file := curfunc.Span.Start.File
		if _, ok := syms[file]; !ok {
			syms[file] = make(map[*cc.Decl]bool)
		}
		syms[file][xfn] = true
	})
	return syms
}

// static ensures that every symbol that can be static, is.
//
// It patches fns in place.
func static(fns []*cc.Decl, syms map[string]map[*cc.Decl]bool) {
	for _, v := range fns {
		def := v.Span.Start.File
		static := true
		for file, s := range syms {
			if file == def {
				continue
			}
			if _, ok := s[v]; ok {
				static = false
			}
		}
		_, entry := start[v.Name]
		if static && !entry {
			v.Storage = cc.Static
		}
	}
}
