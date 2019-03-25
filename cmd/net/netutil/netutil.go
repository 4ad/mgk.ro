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
Package netutil implements some network I/O utility functions.
*/
package netutil // import "mgk.ro/cmd/net/netutil"

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
