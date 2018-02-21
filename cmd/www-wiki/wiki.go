// Copyright (c) 2018 Aram Hăvărneanu <aram@mgk.ro>
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

// www-wiki is a small server that provides a mediawiki instalation.
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/handlers"

	_ "mgk.ro/log"
	"mgk.ro/www/mediawiki"
)

func main() {
	h := mediawiki.New("/opt/pkg/share/mediawiki", "/w/", "/opt/pkg/libexec/cgi-bin/php")

	http.Handle("/w/", handlers.CombinedLoggingHandler(os.Stdout, h))
	log.Fatal(http.ListenAndServe(":8080", nil))
}
