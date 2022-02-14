/*
devdraw-proxy: fake devdraw
	DEVDRAW_SERVER=net!addr DEVDRAW=devdraw-proxy cmd

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

	"mgk.ro/net/netutil"
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
