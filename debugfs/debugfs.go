/*
Package debugfs implements helper functions for accessing the Linux
debugfs (/sys/kernel/debug).
*/
package debugfs // import "mgk.ro/debugfs"

import (
	"os"
)

const (
	UprobesEvents = "/sys/kernel/debug/tracing/uprobe_events"
	UprobesEnable = "/sys/kernel/debug/tracing/events/uprobes/enable"
	Trace         = "/sys/kernel/debug/tracing/trace"
	TracePipe     = "/sys/kernel/debug/tracing/trace_pipe"
)

// MustOpenX exclusively opens the named file. It panics if it cant't open
// the file.
func MustOpenX(name string) *os.File {
	f, err := os.OpenFile(name, os.O_RDWR, 0)
	if err != nil {
		panic(err)
	}
	return f
}

// Enable writes the string "1" to the file.
func Enable(f *os.File) error {
	_, err := f.WriteString("1")
	return err
}

// Disable writes the string "0" to the file.
func Disable(f *os.File) error {
	_, err := f.WriteString("0")
	return err
}
