/*
Package log sets the standard log package to print simple prog: message
messages. It doesn't export any symbol, it should be imported only for
side effects.
*/
package log // import "mgk.ro/log"

import (
	"log"
	"os"
	"path"
)

func init() {
	log.SetFlags(0)
	log.SetPrefix(path.Base(os.Args[0] + ": "))
}
