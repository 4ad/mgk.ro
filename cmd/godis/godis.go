// Copyright (c) 2016 Aram Hăvărneanu <aram@mgk.ro>
//
// Permission to use, copy, modify, and distribute this software for any
// purpose with or without fee is hereby granted, provided that the above
// copyright notice and this permission notice appear in all copies.
//
// THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
// WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
// MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
// ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
// WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
// ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
// OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.

/*
Godis: disassemble -S output from the Go gc toolchain.

Godis reads standard input, scans for the dump from -S, and uses
the tool to disassemble the bytes.

Options:
    -tool	name of the tool to use (default sparc64-none-elf-objdump)
    -flags	flags to pass to the tool (default "-EB -m sparc:v9 -b binary -D")
    -hex	print the bytes in hexadecimal (extract only the dump from the output)
    -binary	print the raw bytes

The file containing the raw bytes is passed as the last argument to
the tool, after the flags.
*/
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

var usageString = `usage: go tool compile -S file.go | gcdis [[-tool toolname] [-flags flags]] | [-hex | -binary]
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
