/*
Gometad: go get vanity redirector

Gometad serves HTML meta tags over HTTP in order to implement mgk.ro vanity domain redirect to https://github.com/4ad/mgk.ro.git.
*/
package main

import (
	"fmt"
	"log"
	"net/http"
)

var meta = `<meta name="go-import" content="mgk.ro git https://github.com/4ad/mgk.ro.git">`

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, meta)
	})
	log.Fatal(http.ListenAndServe(":80", nil))
}
