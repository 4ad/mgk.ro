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

// filemap maps each input file to its corresponding output file.
// Unknown files go to zzz.c.
// missing *.h pstate.c main.c.
var filemap = map[string]string{
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

// using these symbols needs a LSym *cursym parameter.
var needcursym = map[string]bool{
	"curtext": true,
	"firstp": true,
}

var includes = `#include <u.h>
#include <libc.h>
#include <bio.h>
#include <link.h>
#include "../cmd/7l/7.out.h"
`

var linksrc = `#include <u.h>
#include <libc.h>
#include <bio.h>
#include <link.h>`

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

// Linkprog is a parsed link.h with a symbol table.
type linkprog struct {
	*cc.Prog
	fields map[string]*cc.Decl // symbol table
}

// replace unqualified names in filemap with full paths.
func init() {
	for k, v := range filemap {
		filemap[runtime.GOROOT()+lddir+"/"+k] = v
		delete(filemap, k)
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

	prog := NewProg(parse(filemap))
	lprog := NewLinkprog(linksrc)

	prog.extract(start)
	prog.print(filemap, "l.0")
	prog.static(start, filemap)
	prog.print(filemap, "l.1")
	prog.rename(rename)
	prog.print(filemap, "l.2")
	prog.addcursym(needcursym)
	prog.print(filemap, "l.3")
	prog.addctxt(lprog)
	prog.print(filemap, "l.4")
	diff()
}

// parse opens and parses all input files, and returns the result as
// a *cc.Prog.
func parse(filemap map[string]string) *cc.Prog {
	var r []io.Reader
	var files []string
	for name, _ := range filemap {
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

func NewLinkprog(src string) *linkprog {
	p, err := cc.Read("virtual", strings.NewReader(src))
	if err != nil {
		log.Fatal(err)
	}
	lp := &linkprog{Prog: p, fields: make(map[string]*cc.Decl)}

	var stack = []*cc.Decl{nil}
	var tos *cc.Decl
	var before = func(x cc.Syntax) {
		decl, ok := x.(*cc.Decl)
		if !ok {
			return
		}
		tos = stack[len(stack)-1]
		stack = append(stack, decl)
		if tos != nil && tos.Name == "Link" && tos.Type.Kind == cc.Struct {
			lp.fields[decl.Name] = decl
		}
	}
	var after = func(x cc.Syntax) {
		_, ok := x.(*cc.Decl)
		if !ok {
			return
		}
		tos, stack = stack[len(stack)-1], stack[:len(stack)-1]
	}
	cc.Walk(lp.Prog, before, after)
	return lp
}

func NewProg(ccprog *cc.Prog) *prog {
	prog := &prog{Prog: ccprog}
	var curfunc *cc.Decl

	// compute symlist, symmap, symtab.
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
	cc.Walk(prog.Prog, before, after)

	// compute dependencies.
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

	// compute prog.filetab.
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

// Print pretty prints prog, writing the output into dir. File names
// are taken from filemap.
//
// BUG(aram): this function can print correctly only functions, 
// not global variable declarations.
func (prog *prog) print(filemap map[string]string, dir string) {
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
		name, ok := filemap[v.Span.Start.File]
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
					if name == filemap[from] {
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
	for from, _ := range filemap {
		files = append(files, from)
	}
	sort.Sort(sort.StringSlice(files))
	return files
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
	out, _ = exec.Command("diff", "-urp", "l.2", "l.3").Output()
	if err := ioutil.WriteFile("d23.patch", out, 0664); err != nil {
		log.Fatal(err)
	}
	out, _ = exec.Command("diff", "-urp", "l.3", "l.4").Output()
	if err := ioutil.WriteFile("d34.patch", out, 0664); err != nil {
		log.Fatal(err)
	}
	out, _ = exec.Command("diff", "-urp", "l.0", "l.4").Output()
	if err := ioutil.WriteFile("d04.patch", out, 0664); err != nil {
		log.Fatal(err)
	}
}

// static ensures that every symbol that can be static, is.
func (prog *prog) static(start map[string]bool, filemap map[string]string) {
	for _, sym := range prog.symlist {
		dfile := filemap[sym.Span.Start.File]
		static := true
		for file, syms := range prog.filetab {
			if filemap[file] == dfile {
				continue
			}
			if _, ok := syms[sym]; ok {
				static = false
			}
		}
		if static && !start[sym.Name] {
			sym.Storage = cc.Static
		}
	}
}

// rename renames prog symbols according to newnames.
func (prog *prog) rename(newnames map[string]string) {
	for _, sym := range prog.symlist {
		if _, ok := newnames[sym.Name]; !ok {
			continue
		}
		sym.Name = newnames[sym.Name]
	}
	for old, new := range newnames {
		sym, ok := prog.symtab[old]
		if !ok {
			continue
		}
		delete(prog.symtab, old)
		prog.symtab[new] = sym
	}
	cc.Preorder(prog.Prog, func(x cc.Syntax) {
		expr, ok := x.(*cc.Expr)
		if !ok {
			return
		}
		if expr.Op != cc.Name {
			return
		}
		if _, ok := newnames[expr.Text]; !ok {
			return
		}
		expr.Text = newnames[expr.Text]
	})
}

// addctxt adds Link *ctxt parameters to functions and rewrites code to use
// the parameter.
func (prog *prog) addctxt(lprog *linkprog) {
	var funcs = make(symset) // functions requiring Link *ctxt parameter
	// fix expressions involving Link *ctxt
	var curfunc *cc.Decl
	cc.Preorder(prog.Prog, func(x cc.Syntax) {
		decl, ok := x.(*cc.Decl)
		if ok {
			if decl.Type.Is(cc.Func) && decl.Body != nil {
				curfunc = decl
			}
			return
		}
		expr, ok := x.(*cc.Expr)
		if !ok {
			return
		}
		switch expr.Op {
		case cc.Name:
			sym, ok := prog.symtab[expr.Text]
			if !ok {
				return
			}
			if lprog.fields[sym.Name] != nil {
				// hack: we only replace the name, not the expression.
				expr.Text = "ctxt->" + expr.Text
				funcs[curfunc] = true
			}
		case cc.Addr, cc.Call:
			if expr.Left == nil || expr.Left.XDecl == nil {
				return
			}
			if _, ok := prog.symmap[expr.Left.XDecl]; !ok {
				return // not a global symbol
			}
			if lprog.fields[expr.Left.XDecl.Name] != nil {
				// hack: we only replace the name, not the expression.
				expr.Text = "ctxt->" + expr.Text
				funcs[curfunc] = true
			}
		}
	})
	// if a function requires a Link *ctxt, its callees require it too.
	var r func(sym *cc.Decl)
	r = func(sym *cc.Decl) {
		if _, ok := funcs[sym]; ok {
			return
		}
		funcs[sym] = true
		for sym := range prog.reverse[sym] {
			r(sym)
		}
	}
	for sym := range funcs {
		r(sym)
	}
	// fix definitions of functions now using Link *ctxt
	for sym := range funcs {
		arg0 := &cc.Decl{
			Name: "ctxt",
			Type: &cc.Type{
				Kind: cc.Ptr,
				Base: &cc.Type{
					Name: "Link",
					Kind: cc.TypedefType, // not even a lie
				},
			},
		}
		if sym.Type.Decls[0].Type.Is(cc.Void) {
			sym.Type.Decls = []*cc.Decl{arg0}
			continue
		}
		sym.Type.Decls = append([]*cc.Decl{arg0}, sym.Type.Decls...)
	}
	// patch call sites of every function now taking a Link *ctxt.
	cc.Preorder(prog.Prog, func(x cc.Syntax) {
		expr, ok := x.(*cc.Expr)
		if !ok {
			return
		}
		if expr.Op != cc.Call {
			return
		}
		if _, ok := prog.symmap[expr.Left.XDecl]; !ok {
			return
		}
		if _, ok := funcs[expr.Left.XDecl]; !ok {
			return
		}
		expr0 := &cc.Expr{
			Op: cc.Name,
			Text: "ctxt",
		}
		expr.List = append([]*cc.Expr{expr0}, expr.List...)
	})
}

// addcursym adds LSym *cursym parameters to functions and rewrites code
// to use the parameter.
func (prog *prog) addcursym(needcursym map[string]bool) {
	// patch function definitions
	var funcs = make(symset)
	for symname := range needcursym {
		sym, ok := prog.symtab[symname]
		if !ok {
			log.Fatalf("symbol %q not found")
		}
		for sym := range prog.reverse[sym] {
			// we don't need to patch diag, liblink diag is different.
			if sym.Name == "diag" {
				continue
			}
			funcs[sym] = true
		}
	}
	// if a function requires a LSym *cursym, its callees require it too.
	var r func(sym *cc.Decl)
	r = func(sym *cc.Decl) {
		if _, ok := funcs[sym]; ok {
			return
		}
		// we don't need to patch diag, liblink diag is different.
		if sym.Name == "diag" {
			return
		}
		funcs[sym] = true
		for sym := range prog.reverse[sym] {
			r(sym)
		}
	}
	for sym := range funcs {
		r(sym)
	}
	// fix definitions of functions now using Link *ctxt
	for sym := range funcs {
		arg0 := &cc.Decl{
			Name: "cursym",
			Type: &cc.Type{
				Kind: cc.Ptr,
				Base: &cc.Type{
					Name: "LSym",
					Kind: cc.TypedefType, // not even a lie
				},
			},
		}
		if sym.Type.Decls[0].Type.Is(cc.Void) {
			sym.Type.Decls = []*cc.Decl{arg0}
			continue
		}
		sym.Type.Decls = append([]*cc.Decl{arg0}, sym.Type.Decls...)
	}
	// patch call sites of every function now taking a LSym *cursym.
	cc.Preorder(prog.Prog, func(x cc.Syntax) {
		expr, ok := x.(*cc.Expr)
		if !ok {
			return
		}
		if expr.Op != cc.Call {
			return
		}
		if _, ok := prog.symmap[expr.Left.XDecl]; !ok {
			return
		}
		if _, ok := funcs[expr.Left.XDecl]; !ok {
			return
		}
		expr0 := &cc.Expr{
			Op: cc.Name,
			Text: "cursym",
		}
		expr.List = append([]*cc.Expr{expr0}, expr.List...)
	})
	// fix expressions involving LSym *cursym.
	var curfunc *cc.Decl
	cc.Preorder(prog.Prog, func(x cc.Syntax) {
		decl, ok := x.(*cc.Decl)
		if ok {
			if decl.Type.Is(cc.Func) && decl.Body != nil {
				curfunc = decl
			}
			return
		}
		expr, ok := x.(*cc.Expr)
		if !ok {
			return
		}
		switch expr.Op {
		case cc.Name:
			sym, ok := prog.symtab[expr.Text]
			if !ok {
				return
			}
			if needcursym[sym.Name] {
				// hack: we only replace the name, not the expression.
				expr.Text = "cursym->text"
			}
		case cc.Addr, cc.Call:
			if expr.Left == nil || expr.Left.XDecl == nil {
				return
			}
			if _, ok := prog.symmap[expr.Left.XDecl]; !ok {
				return // not a global symbol
			}
			if needcursym[expr.Left.XDecl.Name] {
				// hack: we only replace the name, not the expression.
				expr.Text = "cursym->text"
			}
		}
	})
}
