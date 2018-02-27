package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/reusing-code/faceoff"

	"github.com/NYTimes/gziphandler"
)

func main() {
	_, err := faceoff.LoadTemplatesFromDisk()
	if err != nil {
		fmt.Println(err)
	}
	port := flag.Int("p", 8086, "port number")
	flag.Parse()
	idxHndlGz := gziphandler.GzipHandler(http.HandlerFunc(indexHandler))
	http.Handle("/", idxHndlGz)
	http.Handle("/static/", gziphandler.GzipHandler(http.StripPrefix("/static/", http.FileServer(http.Dir("static")))))
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
	if r.URL.Path == "/client.js" {
		jsHandler(w, r)
		return
	}
	http.ServeFile(w, r, "../client/index.html")
}
