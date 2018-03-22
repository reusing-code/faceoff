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

	"github.com/gorilla/mux"

	"github.com/reusing-code/faceoff"
)

func main() {
	port := flag.Int("p", 8086, "port number")
	flag.Parse()

	OpenDB()

	router := mux.NewRouter()
	xhr := router.PathPrefix("/xhr/{key:[0-9]+}").Subrouter()
	xhr.HandleFunc("/roster.json", rosterHandler)
	xhr.HandleFunc("/submit-vote", voteHandler)
	xhr.HandleFunc("/advance-round", roundAdvanceHandler)
	xhr.HandleFunc("/commit-new-roster", newRosterHandler)

	router.HandleFunc("/rosterlist.json", rosterListHandler)
	router.HandleFunc("/ws/{key:[0-9]+}", ServeWs)
	router.HandleFunc("/templates", templateHandler)
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	router.PathPrefix("/").HandlerFunc(indexHandler)

	http.Handle("/", router)
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
	r, err := faceoff.CreateRosterRaw("Default", b)
	if err != nil {
		panic(err)
	}
	return r
}

func rosterHandler(w http.ResponseWriter, r *http.Request) {
	key := mux.Vars(r)["key"]
	roster, err := GetRoster(key)
	if err != nil {
		handleNotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	b, err := json.Marshal(roster)
	if err != nil {
		handleError(w, err)
		return
	}
	_, err = w.Write(b)
	if err != nil {
		handleError(w, err)
		return
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
	key := mux.Vars(r)["key"]
	voteRoster, err := faceoff.ParseRoster(r.Body)
	if err != nil {
		return
	}
	scoreRoster, err := GetRoster(scoreKey(key))
	if err != nil {
		return
	}
	roster, err := GetRoster(key)
	if err != nil {
		return
	}
	if bytes.Compare(voteRoster.UUID, scoreRoster.UUID) == 0 {
		scoreRoster.AddVotes(voteRoster)
		roster.CurrentVotes++
		SetRoster(scoreKey(key), scoreRoster)
		SetRoster(key, roster)
		TriggerUpdate(key)
	}
}

func roundAdvanceHandler(w http.ResponseWriter, r *http.Request) {
	key := mux.Vars(r)["key"]
	if r.Method != "POST" {
		println("/advance-round called with " + r.Method + ". Ignoring")
		return
	}
	b := &bytes.Buffer{}
	b.ReadFrom(r.Body)
	id := b.Bytes()
	r.Body.Close()
	scoreRoster, err := GetRoster(scoreKey(key))
	if err != nil {
		return
	}
	if bytes.Compare(id, scoreRoster.UUID) == 0 {
		scoreRoster.AdvanceRound()
		SetRoster(key, scoreRoster)
		SetRoster(scoreKey(key), scoreRoster)
		TriggerUpdate(key)
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
	if len(participants) < 1 {
		println("Bad data in /commit-new-roster: slice empty")
		return
	}

	roster, err := faceoff.CreateRoster(participants[0], participants[1:])
	if err != nil {
		println("Bad data in /commit-new-roster: " + err.Error())
		return
	}
	key := CreateKey()
	SetRoster(key, roster)
	SetRoster(scoreKey(key), roster)

	w.Write([]byte(key))

}

func scoreKey(key string) string {
	return key + "_score"
}

func rosterListHandler(w http.ResponseWriter, r *http.Request) {
	list := GetContestList()
	w.Header().Set("Content-Type", "application/json")
	b, err := json.Marshal(list)
	if err != nil {
		handleError(w, err)
		return
	}
	_, err = w.Write(b)
	if err != nil {
		handleError(w, err)
		return
	}
}
