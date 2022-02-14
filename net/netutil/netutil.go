/*
Package netutil implements some network I/O utility functions.
*/
package netutil // import "mgk.ro/net/netutil"

import (
	"errors"
	"net"
	"strings"
)

// SplitDialString takes a Plan 9 dialstring like tcp!golang.org!http
// and returns the constituent elements in a form useful to net.Dial.
func SplitDialString(s string) (net, addr string, err error) {
	ss := strings.Split(s, "!")
	if len(ss) == 3 {
		return ss[0], ss[1] + ":" + ss[2], nil
	}
	if len(ss) != 2 {
		return "", "", errors.New("invalid dialstring")
	}
	return ss[0], ss[1], nil
}

// Dial is like net.Dial, but takes a Plan 9 dial string as its
// argument.
func Dial(dialstring string) (conn net.Conn, err error) {
	netw, addr, err := SplitDialString(dialstring)
	if err != nil {
		return nil, err
	}
	return net.Dial(netw, addr)
}
