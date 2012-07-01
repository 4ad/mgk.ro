// Permission to use, copy, modify, and/or distribute this software for
// any purpose is hereby granted, provided this notice appear in all copies.

/*
Package log sets the standard log package to print simple prog: message
messages. It doesn't export any symbol, it should be imported only for
side effects.
*/
package log

import (
	"log"
	"os"
	"path"
)

func init() {
	log.SetFlags(0)
	log.SetPrefix(path.Base(os.Args[0] + ": "))
}
