/*
Package godebug implements helper functions for debugging Go programs.
*/
package godebug // import "mgk.ro/godebug"

import (
	"debug/elf"
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
