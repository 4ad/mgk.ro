package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	_ "mgk.ro/log"
)

var (
	flagTool   = flag.String("tool", "sparc64-none-elf-objdump", "disassembler name")
	flagFlags  = flag.String("flags", "-EB -m sparc:v9 -b binary -D", "flags to pass to the tool")
	flagHex    = flag.Bool("hex", false, "print the bytes in hexadecimal")
	flagBinary = flag.Bool("binary", false, "print the binary")
)

var usageString = `usage: gcdis [[-tool toolname] [-flags flags]] | [-hex | -binary]
Options:
`

func usage() {
	fmt.Fprint(os.Stderr, usageString)
	flag.PrintDefaults()
	os.Exit(1)
}

var bytesRE = regexp.MustCompile(`(^|[[:space:]])(([a-f0-9][a-f0-9]) ([a-f0-9][a-f0-9]) ([a-f0-9][a-f0-9]) ([a-f0-9][a-f0-9]))`)

func main() {
	flag.Usage = usage
	flag.Parse()

	var buf []byte
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		for _, v := range bytesRE.FindAllStringSubmatch(scanner.Text(), -1) {
			for _, vv := range v[3:7] {
				b, _ := strconv.ParseUint(vv, 16, 8)
				buf = append(buf, byte(b))
			}
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatal("reading standard input:", err)
	}
	if *flagHex {
		fmt.Printf("% x\n", buf)
		return
	}
	if *flagBinary {
		os.Stdout.Write(buf)
	}
	tmpf, err := ioutil.TempFile("", "obj")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(tmpf.Name())
	_, err = tmpf.Write(buf)
	if err != nil {
		log.Fatal(err)
	}
	flags := strings.Split(*flagFlags, " ")
	flags = append(flags, tmpf.Name())
	out, err := exec.Command(*flagTool, flags...).Output()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s", out)
}
