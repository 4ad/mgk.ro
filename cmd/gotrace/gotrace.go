package main

import (
	"debug/elf"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"syscall"

	_ "mgk.ro/log"
)

var (
	follow    = flag.Bool("follow", true, "follow children")
	indent    = flag.Bool("indent", false, "indent callees; assumes -ret")
	output    = flag.String("o", "", "print to file instead of stderr")
	pid       = flag.Int("p", 0, "examine process with pid")
	printArgs = flag.Bool("args", true, "print arguments and return value")
	printTime = flag.Bool("t", false, "print timestamps")
	printCpu  = flag.Bool("cpu", false, "print CPU number")
	traceRet  = flag.Bool("ret", true, "trace return from function")
)

var filter MultiFlag

func init() {
	flag.Var(&filter, "filter", "only trace functions matching regexp; can be set multiple times")
}

// MultiFlag allows setting a value multiple times to collect a list,
// as in -I=dir1 -I=dir2.

type MultiFlag []string

func (m *MultiFlag) String() string {
	return `'.'`
}

func (m *MultiFlag) Set(val string) error {
	(*m) = append(*m, val)
	return nil
}

func Usage() {
	fmt.Fprintf(os.Stderr, "usage: gotrace [options] command [args]\n")
	fmt.Fprintf(os.Stderr, "       gotrace [options] -p pid\n")
	fmt.Fprintf(os.Stderr, "Flags:\n")
	flag.PrintDefaults()
	os.Exit(2)
}

func flags() {
	flag.Usage = Usage
	flag.Parse()
	if flag.NArg() == 0 && *pid == 0 || flag.NArg() != 0 && *pid != 0 {
		flag.Usage()
	}
	if !*traceRet && *indent {
		flag.Usage()
	}
	if len(filter) == 0 {
		filter = append(filter, ".")
	}
}

var funcs = map[string]uint64{} // name -> address

func findFuncs(cmd string) {
	path, err := exec.LookPath(cmd)
	if err != nil {
		log.Fatal(err)
	}
	f, err := elf.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	syms, err := f.Symbols()
	if err != nil {
		log.Fatalf("processing %q: %v", path, err)
	}
	if len(syms) == 0 {
		log.Fatalf("processing %q: symbol table is empty", path)
	}
	for _, s := range syms {
		switch s.Section {
		case elf.SHN_UNDEF:
		case elf.SHN_COMMON:
			break
		default:
			i := int(s.Section)
			if i < 0 || i >= len(f.Sections) {
				break
			}
			sect := f.Sections[i]
			switch sect.Flags & (elf.SHF_WRITE | elf.SHF_ALLOC | elf.SHF_EXECINSTR) {
			case elf.SHF_ALLOC | elf.SHF_EXECINSTR:
				for _, v := range filter {
					matched, err := regexp.MatchString(v, s.Name)
					if err != nil {
						log.Fatal(err)
					}
					if matched {
						funcs[s.Name] = s.Value
					}
				}
			}
		}
	}
}

func startProcess() int {
	path, err := exec.LookPath(flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}
	runtime.LockOSThread()
	attr := &os.ProcAttr {
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
		Sys: &syscall.SysProcAttr{Ptrace: true},
	}
	proc, err := os.StartProcess(path, flag.Args(), attr)
	if err != nil {
		log.Fatal(err)
	}
	return proc.Pid
}

var breakpoints = map[uint64][4]byte{}
var reverse = map[uint64]string{}

func addBreakPoint(addr uint64) {
	var out [4]byte
	_, err := syscall.PtracePeekText(*pid, uintptr(addr), out[:])
	if err != nil {
		log.Fatal(err)
	}
	breakpoints[addr] = out

	_, err = syscall.PtracePokeText(*pid, uintptr(addr), []byte{0xD4, 0x2, 0, 0})
	if err != nil {
		log.Fatal(err)
	}
}

func addBreakPoints() {
	for name, addr := range funcs {
		addBreakPoint(addr)
		reverse[addr] = name
		log.Printf("added breakpoint at 0x%x (%s)", addr, name)
	}
}

func traceProcess(pid int) {
	var wstat syscall.WaitStatus
	_, err := syscall.Wait4(pid, &wstat, 0, nil)
	if err != nil {
		log.Fatal(err)
	}

	// TODO(aram): only if follow == true?
	err = syscall.PtraceSetOptions(pid, syscall.PTRACE_O_TRACECLONE)
	if err != nil {
		log.Fatal(err)
	}

	addBreakPoints()

	i := 0
	for wstat.Stopped() {	
		regs := syscall.PtraceRegs{}
		if err := syscall.PtraceGetRegs(pid, &regs); err != nil {
			log.Fatal(err)
		}
		log.Printf("PC=0x%x: %s", regs.PC(), reverse[regs.PC()])
		i++
		if i >= 10 {
			log.Fatal("exit")
		}
		orig := breakpoints[regs.PC()]
		_, err := syscall.PtracePokeText(pid, uintptr(regs.PC()), orig[:])
		if err != nil {
			log.Fatal(err)
		}
		regs.SetPC(regs.PC() - 4)

		err = syscall.PtraceCont(pid, 0)
		if err != nil {
			log.Fatal(err)
		}

		_, err = syscall.Wait4(pid, &wstat, 0, nil)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func main() {
	flags()
	if *pid != 0 {
		log.Fatal("attaching to processes is not implemented yet")
	} else {
		findFuncs(flag.Arg(0))
		*pid = startProcess()
	}
	traceProcess(*pid)
}
