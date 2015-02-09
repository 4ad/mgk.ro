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

// Enable writes the string "1" to the file.
func Enable(name string) error {
	f, err := os.OpenFile(name, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	_, err = f.WriteString("1\n")
	return err
}

// Disable writes the string "0" to the file.
func Disable(name string) error {
	f, err := os.OpenFile(name, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	_, err = f.WriteString("0\n")
	return err
}
