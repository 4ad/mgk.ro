/*
Gotrace: go function tracer

Usage:
	gotrace [options] command [args]

Flags:

  -filter='.': only trace functions matching regexp; can be set multiple times
  -o="": print to file instead of stderr
  -ret=true: trace return from function
  -run=true: run the command
  -trace=true: enables tracing; false makes program print uprobes to output
  -leavetrace=false: leaves tracing on
*/
package main

// BUG(aram): Only one instance of the program can be run concurrently.
// BUG(aram): If this program crashes, it might leave tracing enabled.
// BUG(aram): After normal exit, this program will leave probes behind (but disabled).
// BUG(aram): This program will completely reset kernel tracing state.
// BUG(aram): This program will trace every instance of the specified program, including existing ones.
// BUG(aram): This program takes forever to exit.
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
	leaveOn   = flag.Bool("leavetrace", false, "leave tracing on")
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
	pipew *io.PipeWriter
	done = make(chan bool)
)

func cleanup() {
	if !*tracing {
		return
	}
	log.Println("disabling tracing... (this might take a while)")
	err := pipew.Close()
	err = os.Truncate(debugfs.UprobesEvents, 0)
	if *leaveOn {
		log.Println("leaving tracing enabled")
		return
	}
	err = debugfs.Disable(debugfs.UprobesEnable)
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
	i := 0
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
			i++
			if *traceRet {
				uprobes = append(uprobes, godebug.UretProbe(prg, &fn))
				i++
			}
		}
	}
	log.Printf("found %d probes", i)
}

func writeprobes() {
	f := out
	var err error
	if *tracing {
		f, err = os.Create(debugfs.UprobesEvents)
		if err != nil {
			log.Fatal(err)
		}
	}
	_, err = io.Copy(f, io.MultiReader(uprobes...))
	if err != nil {
		log.Fatal(err)
	}
	if *tracing {
		f.Close()
	}
}

func trace() {
	if !*tracing {
		return
	}
	err := os.Truncate(debugfs.Trace, 0)
	if err != nil {
		log.Fatal(err)
	}
	tracePipe, err := os.Open(debugfs.TracePipe)
	if err != nil {
		log.Fatal(err)
	}
	err = debugfs.Enable(debugfs.UprobesEnable)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("tracing...")
	r, w := io.Pipe()
	pipew = w
	go func() {
		io.Copy(w, tracePipe)
	}()
	go func() {
		io.Copy(out, r)
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
	cleanup()
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
