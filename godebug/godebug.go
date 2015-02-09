/*
Package godebug implements helper functions for debugging Go programs.
*/
package godebug // import "mgk.ro/godebug"

import (
	"debug/elf"
	"debug/gosym"
	"fmt"
	"os/exec"
	"path/filepath"
)

// ProgLoadAddr returns the program load address. It's useful to calculate
// file offset from VA for uprobes.
func ProgLoadAddr(f *elf.File) uint64 {
	for _, p := range f.Progs {
		if p.Type == elf.PT_LOAD && p.Flags == elf.PF_X|elf.PF_R {
			return p.Vaddr
		}
	}
	panic("program load address not found")
}

// Prog is a representation of the debugged program.
type Prog struct {
	*elf.File
	*gosym.Table

	load uint64
}

func NewProg(cmd *exec.Cmd) (*Prog, error) {
	file := cmd.Path
	if !filepath.IsAbs(file) {
		file = cmd.Dir + cmd.Path
	}
	f, err := elf.Open(file)
	if err != nil {
		return nil, err
	}
	symdat, err := f.Section(".gosymtab").Data()
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("reading %s gosymtab: %v", file, err)
	}
	pclndat, err := f.Section(".gopclntab").Data()
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("reading %s gopclntab: %v", file, err)
	}

	pcln := gosym.NewLineTable(pclndat, f.Section(".text").Addr)
	tab, err := gosym.NewTable(symdat, pcln)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("parsing %s gosymtab: %v", file, err)
	}
	prg := &Prog{
		File: f,
		Table: tab,
		load: ProgLoadAddr(f),
	}
	return prg, nil
}

// FuncOffset returns the offset of the named function in the memory
// image. This offset is used by uprobes.
func (p *Prog) FuncOffset(name string) uint64 {
	fn := p.LookupFunc(name)
	if fn == nil {
		panic("can't find function " + name)
	}
	return FuncOffset(fn, p.load)
}

// FuncOffset returns the offset of the function in the memory
// image. This offset is used by uprobes.
func FuncOffset(fn *gosym.Func, load uint64) uint64 {
	return fn.Entry - load
}
