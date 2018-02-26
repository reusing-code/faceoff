package main

import (
	"log"
	"net/http"
	"strings"
)

func main() {
	http.HandleFunc("/", indexHandler)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	log.Fatal(http.ListenAndServe(":8086", nil))
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
