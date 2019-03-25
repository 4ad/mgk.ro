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

This program is not intended to be called directly by the user, but
by plan9port graphical programs.
*/
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"mgk.ro/cmd/net/netutil"
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

	conn, err := netutil.Dial(os.Getenv("DEVDRAW_SERVER"))
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
