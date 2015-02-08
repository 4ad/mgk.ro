/*
Package uprobes implements a small DSL that targets Linux Uprobes. You use it like this:

	e1 := uprobes.NewEvent("malloc_entry", "/bin/bash", 0x747d0).Stack("stk", 0).Register("size", "x0").S64()
	e2 := uprobes.NewEvent("malloc_return", "/bin/bash", 0x747d0).Return().Stack("", 0).RetVal("ret")

These will generate the following uprobe "code":

	p:malloc_entry /bin/bash:0x747d0 stk=$stack size=%x0:s64
	r:malloc_return /bin/bash:0x747d0 $stack ret=$retval

Events are io.Readers so you can do this:

	io.Copy(events, io.MultiReader(e1, e2))
*/
package uprobes // import "mgk.ro/uprobes"

// BUG(aram): Uprobes are Linux-specific.
// BUG(aram): Most of the code dealing with types should be generated.
// BUG(aram): It is not clear that the io.Reader based API adds any value.
// BUG(aram): There is no sanity checking and no errors of any kind.
// BUG(aram): Some names were made more Go-like and less Linux-like.

import (
	"fmt"
	"io"
	"strings"
)

const (
	Probe    = "p" // uprobe
	RetProbe = "r" // uretprobe
	ClrProbe = "-" // clear probe
)

// Event generates a new event, a.i. a line you write
// to the uprobe_events file. For more details see
// https://www.kernel.org/doc/Documentation/trace/uprobetracer.txt.
//
// You can add arguments to an event by calling methods such as e.Register,
// e.Address, etc. These method return a new *Event, so you can chain
// them together.
//
// An *Event is an io.Reader, so you can pass it to file I/O functions that
// copy to uprobe_events file, catenate multiple events together to make
// a new io.Reader, etc.
type Event struct {
	Kind      string // p, r, or -
	Group     string // group name, Linux will set empty group to "uprobes"
	Name      string // event name, a.i. EVENT in uprobetracer.txt
	Path      string // path to ELF file
	Offset    int    // offset where probe is inserted
	FetchArgs Args   // probe arguments

	r io.Reader
}

// NewEvent returns a new uprobe event with given settings. To add uprobe
// arguments you probably want to call methods like e.Register, e.Address,
// etc.
func NewEvent(name, path string, offset int) *Event {
	return &Event{
		Kind:   "p",
		Name:   name,
		Path:   path,
		Offset: offset,
	}
}

// String returns an Event in the format uprobe_events expects.
func (e *Event) String() string {
	if e.Kind == "-" {
		s := "-:"
		if e.Group != "" {
			s += e.Group + "/"
		}
		s += e.Name
		return s
	}
	s := e.Kind
	if e.Name != "" {
		s += ":"
		if e.Group != "" {
			s += e.Group + "/"
		}
		s += e.Name
	}
	s += " " + e.Path + ":"
	s += fmt.Sprintf("0x%x ", e.Offset)
	for _, v := range e.FetchArgs {
		s += v.String() + " "
	}
	return s
}

// Read implements io.Reader so you can just io.Copy to an opened
// uprobe_events file.
func (e *Event) Read(b []byte) (_ int, _ error) {
	if e.r == nil {
		e.r = strings.NewReader(e.String() + "\n")
	}
	return e.r.Read(b)
}

// Remove returns a new probe that will clear e.
func (e *Event) Remove() *Event {
	re := *e
	re.Kind = "-"
	return &re
}

// Return returns a new uretprobe corresponding to uprobe e.
func (e *Event) Return() *Event {
	re := *e
	re.Kind = "r"
	return &re
}

// Arg represents something to be fetched from memory. For more details
// see https://www.kernel.org/doc/Documentation/trace/uprobetracer.txt.
type Arg struct {
	Name  string       // name of the argument
	Type               // u16, s64, etc
	Value fmt.Stringer // the actual argument

	// offset from the actual argument, this is +|-offs(FETCHARG),
	// NOT @+OFFSET
	Offset uint64
}

// String return an argument in the format uprobe_events expects.
func (f *Arg) String() string {
	var s string
	if f.Name != "" {
		s += f.Name + "="
	}
	if f.Offset != 0 {
		s += fmt.Sprintf("%+d(", f.Offset)
	}
	s += f.Value.String()
	if f.Offset != 0 {
		s += ")"
	}
	s += f.Type.String()
	return s
}

// type of an argument, u16, s64, etc.
type Type int

const (
	TypeNone Type = iota
	TypeU8
	TypeU16
	TypeU32
	TypeU64
	TypeS8
	TypeS16
	TypeS32
	TypeS64
	TypeString
	TypeBitfield
)

// String returns the argument type in the format uprobe_events expects.
func (t Type) String() string {
	switch t {
	case TypeNone:
		return ""
	case TypeU8:
		return ":u8"
	case TypeU16:
		return ":u16"
	case TypeU32:
		return ":u32"
	case TypeU64:
		return ":u64"
	case TypeS8:
		return ":s8"
	case TypeS16:
		return ":s16"
	case TypeS32:
		return ":s32"
	case TypeS64:
		return ":s64"
	case TypeString:
		return ":string"
	case TypeBitfield:
		return ":bitfield"
	}
	panic("unreachable")
}

// Register represents a register fetch in the canonical format, e.g. rax,
// x1, etc.
type Register string

func (r Register) String() string {
	return "%" + string(r)
}

// Address represents a memory fetch from an absolute address.
type Address uint64

func (a Address) String() string {
	return fmt.Sprintf("@0x%x", uint64(a))
}

// Offset represents a memory fetch from an address relative to the file
// in Event.Path. It is not the same as Arg.Offset.
type Offset uint64

func (o Offset) String() string {
	return fmt.Sprintf("@+0x%x", uint64(o))
}

// Stack fetches the nth entry of stack.
type Stack int

func (s Stack) String() string {
	if s == 0 {
		return "$stack"
	}
	return fmt.Sprintf("$stack%d", int(s))
}

// RetVal fetches the return value assuming SystemV ABI. Only for return
// probes.
type RetVal int

func (r RetVal) String() string {
	return "$retval"
}

// Args is a collection of arguments. It exists to add methods that make it
// easy to add arguments. All the convenience methods (e.g. Args.Register,
// Args.Address) return a new *Args so that you can chain them together.
//
// The Args.U8, Args.Str, etc, methods operate on the last added Arg.
//
// It's fine to call all these methods on nil Args.
type Args []Arg

func (fa Args) Register(name, reg string) Args {
	return append(fa, Arg{Name: name, Value: Register(reg)})
}

func (e *Event) Register(name, reg string) *Event {
	e.FetchArgs = e.FetchArgs.Register(name, reg)
	return e
}

func (fa Args) RegisterOffset(name, reg string, off uint64) Args {
	return append(fa, Arg{Name: name, Value: Register(reg), Offset: off})
}

func (e *Event) RegisterOffset(name, reg string, off uint64) *Event {
	e.FetchArgs = e.FetchArgs.RegisterOffset(name, reg, off)
	return e
}

func (fa Args) Address(name string, addr uint64) Args {
	return append(fa, Arg{Name: name, Value: Address(addr)})
}

func (e *Event) Address(name string, addr uint64) *Event {
	e.FetchArgs = e.FetchArgs.Address(name, addr)
	return e
}

func (fa Args) AddressOffset(name string, addr, off uint64) Args {
	return append(fa, Arg{Name: name, Value: Address(addr), Offset: off})
}

func (e *Event) AddressOffset(name string, addr, off uint64) *Event {
	e.FetchArgs = e.FetchArgs.AddressOffset(name, addr, off)
	return e
}

func (fa Args) Stack(name string, N int) Args {
	return append(fa, Arg{Name: name, Value: Stack(N)})
}

func (e *Event) Stack(name string, N int) *Event {
	e.FetchArgs = e.FetchArgs.Stack(name, N)
	return e
}

func (fa Args) RetVal(name string) Args {
	return append(fa, Arg{Name: name, Value: RetVal(0)})
}

func (e *Event) RetVal(name string) *Event {
	e.FetchArgs = e.FetchArgs.RetVal(name)
	return e
}

func (fa Args) U8() Args {
	if len(fa) == 0 {
		return fa
	}
	fa[len(fa)-1].Type = TypeU8
	return fa
}

func (e *Event) U8() *Event {
	e.FetchArgs = e.FetchArgs.U8()
	return e
}

func (fa Args) U16() Args {
	if len(fa) == 0 {
		return fa
	}
	fa[len(fa)-1].Type = TypeU16
	return fa
}

func (e *Event) U16() *Event {
	e.FetchArgs = e.FetchArgs.U16()
	return e
}

func (fa Args) U32() Args {
	if len(fa) == 0 {
		return fa
	}
	fa[len(fa)-1].Type = TypeU32
	return fa
}

func (e *Event) U32() *Event {
	e.FetchArgs = e.FetchArgs.U32()
	return e
}

func (fa Args) U64() Args {
	if len(fa) == 0 {
		return fa
	}
	fa[len(fa)-1].Type = TypeU64
	return fa
}

func (e *Event) U64() *Event {
	e.FetchArgs = e.FetchArgs.U64()
	return e
}

func (fa Args) S8() Args {
	if len(fa) == 0 {
		return fa
	}
	fa[len(fa)-1].Type = TypeS8
	return fa
}

func (e *Event) S8() *Event {
	e.FetchArgs = e.FetchArgs.S8()
	return e
}

func (fa Args) S16() Args {
	if len(fa) == 0 {
		return fa
	}
	fa[len(fa)-1].Type = TypeS16
	return fa
}

func (e *Event) S16() *Event {
	e.FetchArgs = e.FetchArgs.S16()
	return e
}

func (fa Args) S32() Args {
	if len(fa) == 0 {
		return fa
	}
	fa[len(fa)-1].Type = TypeS32
	return fa
}

func (e *Event) S32() *Event {
	e.FetchArgs = e.FetchArgs.S32()
	return e
}

func (fa Args) S64() Args {
	if len(fa) == 0 {
		return fa
	}
	fa[len(fa)-1].Type = TypeS64
	return fa
}

func (e *Event) S64() *Event {
	e.FetchArgs = e.FetchArgs.S64()
	return e
}

func (fa Args) Str() Args {
	if len(fa) == 0 {
		return fa
	}
	fa[len(fa)-1].Type = TypeString
	return fa
}

func (e *Event) Str() *Event {
	e.FetchArgs = e.FetchArgs.Str()
	return e
}

func (fa Args) Bit() Args {
	if len(fa) == 0 {
		return fa
	}
	fa[len(fa)-1].Type = TypeBitfield
	return fa
}

func (e *Event) Bit() *Event {
	e.FetchArgs = e.FetchArgs.Bit()
	return e
}
