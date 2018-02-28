package main

import (
	"flag"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/reusing-code/faceoff"

	"github.com/NYTimes/gziphandler"
)

func main() {
	port := flag.Int("p", 8086, "port number")
	flag.Parse()
	idxHndlGz := gziphandler.GzipHandler(http.HandlerFunc(indexHandler))
	http.Handle("/", idxHndlGz)
	http.Handle("/static/", gziphandler.GzipHandler(http.StripPrefix("/static/", http.FileServer(http.Dir("static")))))
	http.HandleFunc("/templates", templateHandler)
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
	if r.URL.Path == "/client.js" || r.URL.Path == "/client.js.map" {
		jsHandler(w, r)
		return
	}
	http.ServeFile(w, r, "../client/index.html")
}

func templateHandler(w http.ResponseWriter, r *http.Request) {

	ts, err := faceoff.LoadTemplatesFromDisk()
	if err == nil {
		var gob []byte
		gob, err = ts.EncodeGob()
		if err == nil {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(gob)
			return
		}

	}

	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte("500 - Something bad happened!" + err.Error()))
	return

}
