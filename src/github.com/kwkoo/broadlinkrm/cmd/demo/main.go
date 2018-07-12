package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/kwkoo/broadlinkrm"
)

var broadlink broadlinkrm.Broadlink
var code string

func handler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	w.Header().Set("Content-type", "text/plain")
	if strings.HasPrefix(path, "/learn") {
		data, err := broadlink.Learn("")
		if err != nil {
			fmt.Fprintf(w, "Error: %v", err)
			return
		}
		fmt.Fprintln(w, data)
		code = data
		return
	}

	if len(code) == 0 {
		fmt.Fprintln(w, "Error: have not learned code")
		return
	}

	broadlink.Execute("", code)
	fmt.Fprintln(w, "OK")
}

func main() {
	broadlink = broadlinkrm.NewBroadlink()
	err := broadlink.Discover()
	if err != nil {
		log.Fatal(err)
	}

	port := 8080
	flag.IntVar(&port, "port", 8080, "HTTP listener port")
	flag.Parse()

	log.Print("Listening on port ", port)
	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}
