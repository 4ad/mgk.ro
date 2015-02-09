/*
Gotrace: go function tracer
*/
package main

// BUG(aram): Only one instance of the program can be run concurrently.
// BUG(aram): If this program crashes, it might leave tracing enabled.
// BUG(aram): After normal exit, this program will leave probes behind (but disabled).
// BUG(aram): This program will completely reset kernel tracing state.
// BUG(aram): This program will trace every instance of the specified program, including existing ones.
// BUG(aram): This program uses uprobes, which are Linux-specific and lack functionality.
// BUG(aram): This program would not be necessary if Linux had DTrace and Go supported DTrace better.

import (
	"fmt"
	"flag"
	"log"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"regexp"

	"mgk.ro/debugfs"
	"mgk.ro/godebug"
	_ "mgk.ro/log"
)

var (
	ofile     = flag.String("o", "", "print to file instead of stderr")
	run       = flag.Bool("run", true, "run the command")
	traceRet  = flag.Bool("ret", true, "trace return from function")
	tracing   = flag.Bool("trace", true, "enables tracing; false makes program print uprobes to output")
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
	fmt.Fprintf(os.Stderr, "Flags:\n")
	flag.PrintDefaults()
	os.Exit(2)
}

var (
	out = os.Stderr
	cmd *exec.Cmd
	prg *godebug.Prog
	uprobes []io.Reader
	eventsFile *os.File
	enableFile *os.File
	done = make(chan bool)
)

func cleanup() {
	if !*tracing {
		return
	}
	out.Close()
	err := debugfs.Disable(enableFile)
	if err != nil {
		log.Fatal(err)
	}
}

func flags() {
	flag.Usage = Usage
	flag.Parse()
	if flag.NArg() == 0 {
		flag.Usage()
	}
	if len(filter) == 0 {
		filter = append(filter, ".")
	}
	if *ofile != "" {
		f, err := os.Create(*ofile)
		if err != nil {
			log.Fatal(err)
		}
		out = f
	}
}

func signals() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	go func() {
		<-c
		cleanup()
	}()
}

func command() {
	cmd = exec.Command(flag.Arg(0), flag.Args()[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
}

func findprobes() {
	prg, err := godebug.NewProg(cmd)
	if err != nil {
		log.Fatal(err)
	}
	for _, fn := range prg.Funcs {
		for _, regex := range filter {
			matched, err := regexp.MatchString(regex, fn.Name)
			if err != nil {
				log.Fatal(err)
			}
			if !matched {
				continue
			}
			uprobes = append(uprobes, godebug.Uprobe(prg, &fn))
			if *traceRet {
				uprobes = append(uprobes, godebug.UretProbe(prg, &fn))
			}
		}
	}
}

func writeprobes() {
	if *tracing {
		eventsFile = debugfs.MustOpenX(debugfs.UprobesEvents)
		err := eventsFile.Truncate(0)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		eventsFile = out
	}
	_, err := io.Copy(eventsFile, io.MultiReader(uprobes...))
	if err != nil {
		log.Fatal(err)
	}
}

func trace() {
	if !*tracing {
		return
	}
	traceFile := debugfs.MustOpenX(debugfs.Trace)
	err := traceFile.Truncate(0)
	traceFile.Close()
	tracePipe := debugfs.MustOpenX(debugfs.TracePipe)
	if err != nil {
		log.Fatal(err)
	}
	enableFile = debugfs.MustOpenX(debugfs.UprobesEnable)
	err = debugfs.Enable(enableFile)
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		io.Copy(out, tracePipe)
		close(done)
	}()
}

func runcmd() {
	if !*run {
		return
	}
	err := cmd.Run()
	if err != nil {
		cleanup()
		log.Fatal(err)
	}
}

func main() {
	defer cleanup()
	flags()
	signals()
	command()
	findprobes()
	writeprobes()
	trace()
	runcmd()
	if *tracing {
		<-done
	}
}
