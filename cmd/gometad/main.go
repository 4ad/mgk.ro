/*
Gometad: go get vanity redirector

Gometad serves HTML meta tags over HTTP in order to implement mgk.ro vanity domain redirect to https://github.com/4ad/mgk.ro.git.

It also serves files in /var/www, because why not.
*/
package main

import (
	"fmt"
	"log"
	"net/http"
)

var meta = `<meta name="go-import" content="mgk.ro git https://github.com/4ad/mgk.ro.git">`

func main() {
	http.Handle("/pub/", http.StripPrefix("/pub/", http.FileServer(http.Dir("/var/www"))))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, meta)
	})
	// go log.Fatal(http.ListenAndServeTLS(":443", "/opt/mgk.ro/ssl-unified.crt", "/opt/mgk.ro/ssl.key", nil))
	log.Fatal(http.ListenAndServe(":80", nil))
}
