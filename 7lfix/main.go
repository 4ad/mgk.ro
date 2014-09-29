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
	"path"
	"runtime"
	"sort"
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
	"span":      true,
	"asmout":    true,
	"chipfloat": true,
	"follow":    true,
	"noops":     true,
	"listinit":  true,
}

// symbols to be renamed.
var rename = map[string]string{
	"span":      "span7",
	"chipfloat": "chipfloat7",
	"listinit":  "listinit7",
	"noops":     "addstacksplit",
}

var includes = `#include <u.h>
#include <libc.h>
#include <bio.h>
#include <link.h>
#include "../cmd/7l/7.out.h"
`

// symset is a set of symols.
type symset map[*cc.Decl]bool
// dependecies expresses forward or reverse dependencies between symbols.
type dependecies map[*cc.Decl]symset

// A prog is a program that need to be refactored.
type prog struct {
	*cc.Prog

	// global declarations, need slice for deterministic range and map
	// for quick lookup.
	symlist []*cc.Decl          
	symmap  symset

	symtab  map[string]*cc.Decl // symbol table
	forward dependecies         // a symbol uses other symbols
	reverse dependecies         // a symbol is used by other symbols
	filetab map[string]symset   // maps files to symbols
}

// symbols is a symbol table.
type symbols map[*cc.Decl][]*cc.Decl

var (
	deps    = symbols{} // deps is the dependency graph between symbols
	all     = symbols{} // all are all the symbols, including unwanted
	globals = symbols{} // globals mapping to functions that use them

	symsfile = map[string]map[*cc.Decl]bool{} // maps files to symbols
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

	prog := NewProg(parse())
	prog.extract(start)
	prog.print(iomap, "l.3")
	
/*
	all = symtab(prog)
	deps = dep(prog, all)
	globals = glob(prog, all)


	foo := NewProg(prog)
	print(foo.symlist, "zzz")

	// extract what we need.
	var syms []*cc.Decl
	for k := range start {
		syms = append(syms, deps.lookup(k))
	}
	subset := extract(syms...)
	print(subset, "l.0")

	// make everything static.
	symsfile = symfile(prog, all)
	static(subset, symsfile)
	print(subset, "l.1")

	// rename symbols.
	ren(prog, deps)
	print(subset, "l.2")

	diff()
*/
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

func NewProg(ccprog *cc.Prog) *prog {
	prog := &prog{Prog: ccprog}
	var curfunc *cc.Decl

	// fill global and symbol table
	prog.symmap = make(symset)
	prog.symtab = make(map[string]*cc.Decl)
	prog.forward = make(dependecies)
	prog.reverse = make(dependecies)
	var before = func(x cc.Syntax) {
		switch x := x.(type) {
		case *cc.Decl:
			if x.XOuter == nil && curfunc == nil {
				prog.forward[x] = make(symset)
				prog.reverse[x] = make(symset)
				prog.symlist = append(prog.symlist, x)
				prog.symmap[x] = true
				prog.symtab[x.Name] = x
			}
			if x.Type.Is(cc.Func) {
				curfunc = x
				return
			}
		}
	}
	var after = func(x cc.Syntax) {
		switch x := x.(type) {
		case *cc.Decl:
			if x.Type.Is(cc.Func) {
				curfunc = nil
				return
			}
		}
	}
	cc.Walk(prog.Prog,before, after)

	// calculate dependencies
	before = func(x cc.Syntax) {
		switch x := x.(type) {
		case *cc.Decl:
			if curfunc == nil {
				if x.Type.Is(cc.Func) && x.Body != nil {
					curfunc = x
					return
				}
			}
		case *cc.Expr:
			switch x.Op {
			case cc.Name:
				if curfunc == nil {
					return
				}
				sym, ok := prog.symtab[x.Text]
				if !ok {
					return
				}
				prog.forward[curfunc][sym] = true
				prog.reverse[sym][curfunc] = true
			case cc.Addr, cc.Call:
				if curfunc == nil {
					return
				}
				if x.Left == nil || x.Left.XDecl == nil {
					return
				}
				if _, ok := prog.symmap[x.Left.XDecl]; !ok {
					return // not a global symbol
				}
				prog.forward[curfunc][x.Left.XDecl] = true
				prog.reverse[x.Left.XDecl][curfunc] = true
			}
		}
	}
	cc.Walk(prog.Prog, before, after)

	// calculate prog.filetab
	prog.filetab = make(map[string]symset)
	for _, v := range prog.symlist {
		file := v.Span.Start.File
		if prog.filetab[file] == nil {
			prog.filetab[file] = make(symset)
		}
		prog.filetab[file][v] = true
	}
	return prog
}

// extract trims prog keeping only the symbols referenced by the start
// functions.
func (prog *prog) extract(start map[string]bool) {
	subset := make(symset)
	var r func(sym *cc.Decl)
	r = func(sym *cc.Decl) {
		if _, ok := subset[sym]; ok {
			return
		}
		subset[sym] = true
		for sym := range prog.forward[sym] {
			r(sym)
		}
	}
	for name := range start {
		sym, ok := prog.symtab[name]
		if !ok {
			log.Fatal("symbol %q not found", name)
		}
		r(sym)
	}
	prog.trim(subset)
}

// trim trims prog keeping only symbols in subset.
func (prog *prog) trim(subset symset) {
	var newsymlist []*cc.Decl
	newsymmap := make(symset)
	for _, sym := range prog.symlist {
		if _, ok := subset[sym]; !ok {
			continue
		}
		newsymlist = append(newsymlist, sym)
		newsymmap[sym] = true
	}
	prog.symlist = newsymlist
	prog.symmap = newsymmap
	for name, sym := range prog.symtab {
		if _, ok := subset[sym]; ok {
			continue
		}
		delete(prog.symtab, name)
	}
	for sym := range prog.forward {
		if _, ok := subset[sym]; ok {
			continue
		}
		delete(prog.forward, sym)
	}
	for sym, syms := range prog.reverse {
		if _, ok := subset[sym]; !ok {
			delete(prog.reverse, sym)
			continue
		}
		for sym := range syms {
			if _, ok := subset[sym]; ok {
				continue
			}
			delete(syms, sym)
		}
	}
	for _, syms := range prog.filetab {
		for sym := range syms {
			if _, ok := subset[sym]; ok {
				continue
			}
			delete(syms, sym)
		}
	}
}

// Print pretty prints prog writing the output into dir. FIle names are
// taked from iomap.
//
// BUG(aram): this correctly  prints only functions, not global variable
// declarations.
func (prog *prog) print(iomap map[string]string, dir string) {
	err := os.RemoveAll(dir)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0775); err != nil {
		log.Fatal(err)
	}
	file := make(map[string]*os.File)
	for _, v := range prog.symlist {
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
				f.WriteString("// From ")
				for _, from := range filenames() {
					if name == iomap[from] {
						f.WriteString(path.Base(from))
						f.WriteString(" ")
					}
				}
				f.WriteString("\n\n")
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
				f.WriteString("// From ")
				for _, from := range filenames() {
					if name == iomap[from] {
						f.WriteString(path.Base(from))
						f.WriteString(" ")
					}
				}
				f.WriteString("\n\n")
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

func filenames() []string {
	var files sort.StringSlice
	for from, _ := range iomap {
		files = append(files, from)
	}
	sort.Sort(sort.StringSlice(files))
	return files
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
		e, ok := x.(*cc.Expr)
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
	if err := ioutil.WriteFile("d01.patch", out, 0664); err != nil {
		log.Fatal(err)
	}
	out, _ = exec.Command("diff", "-urp", "l.1", "l.2").Output()
	if err := ioutil.WriteFile("d12.patch", out, 0664); err != nil {
		log.Fatal(err)
	}
}

// static ensures that every symbol that can be static, is.
//
// It patches fns in place.
func static(fns []*cc.Decl, syms map[string]map[*cc.Decl]bool) {
	for _, v := range fns {
		def := iomap[v.Span.Start.File]
		static := true
		for file, s := range syms {
			if iomap[file] == def {
				continue
			}
			if _, ok := s[v]; ok {
				static = false
			}
		}
		if static && !start[v.Name] {
			v.Storage = cc.Static
		}
	}
}
