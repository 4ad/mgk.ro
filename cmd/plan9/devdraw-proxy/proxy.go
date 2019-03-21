// Copyright (c) 2019 Aram Hăvărneanu <aram@mgk.ro>
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
devdraw-proxy: fake devdraw

This tool masquarades as devdraw for plan9port binaries when
DEVDRAW=devdraw-proxy. It relays the protocol to the devdraw server
specified by DEVDRAW_SERVER.
*/
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"

	_ "mgk.ro/log"
)

var usageString = "usage: DEVDRAW_SERVER=net!addr DEVDRAW=devdraw-proxy cmd\n"

func usage() {
	fmt.Fprint(os.Stderr, usageString)
	os.Exit(1)
}

func main() {
	flag.Usage = usage
	flag.Parse()

	conn, err := net.Dial(dialstring(os.Getenv("DEVDRAW_SERVER")))
	if err != nil {
		log.Fatal(err)
	}
	go proxy(os.Stdin, conn)
	proxy(conn, os.Stdout)
}

func proxy(dst io.Writer, src io.Reader) {
	_, err := io.Copy(dst, src)
	if err != nil {
		log.Fatal(err)
	}
}

func dialstring(s string) (net, addr string) {
	ss := strings.Split(s, "!")
	if len(ss) != 2 {
		log.Fatal("DEVDRAW_SERVER invalid or unset")
	}
	return ss[0], ss[1]
}