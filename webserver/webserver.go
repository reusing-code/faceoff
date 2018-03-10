package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/reusing-code/faceoff"

	"github.com/NYTimes/gziphandler"
)

const rosterKey = "testKey"
const scoreKey = "scoreKey"

func main() {
	port := flag.Int("p", 8086, "port number")
	flag.Parse()

	OpenDB()

	currentRoster := createRoster("values.txt")

	SetRoster(rosterKey, currentRoster)
	SetRoster(scoreKey, currentRoster)

	idxHndlGz := gziphandler.GzipHandler(http.HandlerFunc(indexHandler))
	http.Handle("/", idxHndlGz)
	http.Handle("/static/", gziphandler.GzipHandler(http.StripPrefix("/static/", http.FileServer(http.Dir("static")))))
	http.HandleFunc("/templates", templateHandler)
	http.HandleFunc("/roster.json", rosterHandler)
	http.HandleFunc("/submit-vote", voteHandler)
	http.HandleFunc("/advance-round", roundAdvanceHandler)
	http.HandleFunc("/commit-new-roster", newRosterHandler)

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

	handleError(w, err)
	return

}

func createRoster(filename string) *faceoff.Roster {
	f, err := os.Open(filename)
	defer f.Close()
	if err != nil {
		panic(err)
	}
	b, err := ioutil.ReadAll(f)
	if err != nil {
		panic(err)
	}
	r, err := faceoff.CreateRosterRaw(b)
	if err != nil {
		panic(err)
	}
	return r
}

func rosterHandler(w http.ResponseWriter, r *http.Request) {
	roster, err := GetRoster(rosterKey)
	if err != nil {
		handleNotFound(w, r)
	}
	w.Header().Set("Content-Type", "application/json")
	b, err := json.Marshal(roster)
	if err != nil {
		handleError(w, err)
	}
	_, err = w.Write(b)
	if err != nil {
		handleError(w, err)
	}
}

func handleError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte("500 - Something bad happened! " + err.Error()))
}

func handleNotFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("404 - " + r.URL.Path))
}

func voteHandler(w http.ResponseWriter, r *http.Request) {
	voteRoster, err := faceoff.ParseRoster(r.Body)
	if err != nil {
		return
	}
	scoreRoster, err := GetRoster(scoreKey)
	if err != nil {
		return
	}
	if bytes.Compare(voteRoster.UUID, scoreRoster.UUID) == 0 {
		scoreRoster.AddVotes(voteRoster)
		scoreRoster.CurrentVotes++
		SetRoster(scoreKey, scoreRoster)
	}
}

func roundAdvanceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		println("/advance-round called with " + r.Method + ". Ignoring")
		return
	}
	b := &bytes.Buffer{}
	b.ReadFrom(r.Body)
	id := b.Bytes()
	r.Body.Close()
	scoreRoster, err := GetRoster(scoreKey)
	if err != nil {
		return
	}
	if bytes.Compare(id, scoreRoster.UUID) == 0 {
		scoreRoster.AdvanceRound()
		SetRoster(rosterKey, scoreRoster)
		SetRoster(scoreKey, scoreRoster)
	}
}

func newRosterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		println("/commit-new-roster called with " + r.Method + ". Ignoring")
		return
	}
	b := &bytes.Buffer{}
	b.ReadFrom(r.Body)
	r.Body.Close()

	participants := make([]string, 0)
	json.Unmarshal(b.Bytes(), &participants)

	roster, err := faceoff.CreateRoster(participants)
	if err != nil {
		println("Bad data in /commit-new-roster: " + err.Error())
		return
	}
	SetRoster(rosterKey, roster)
	SetRoster(scoreKey, roster)

}
