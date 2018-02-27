package main

import (
	"flag"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/NYTimes/gziphandler"
)

func main() {
	port := flag.Int("p", 8086, "port number")
	flag.Parse()
	idxHndl := http.HandlerFunc(indexHandler)
	idxHndlGz := gziphandler.GzipHandler(idxHndl)
	http.Handle("/", idxHndlGz)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(*port), nil))
}

func jsHandler(w http.ResponseWriter, r *http.Request) {
	if strings.Contains(r.URL.Path, "map") {
		http.ServeFile(w, r, "../client/client.js.map")
		return
	}
	http.ServeFile(w, r, "../client/client.js")
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if strings.Contains(r.URL.Path, "client.js") {
		jsHandler(w, r)
		return
	}
	http.ServeFile(w, r, "../client/index.html")
}
