package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"strconv"

	"github.com/reusing-code/faceoff/webserver/websockets"

	"github.com/gorilla/mux"

	"github.com/reusing-code/faceoff/shared/contest"
	"github.com/reusing-code/faceoff/shared/templates"
)

func main() {
	port := flag.Int("p", 8086, "port number")
	flag.Parse()

	err := OpenDB("db/faceoff.db")
	if err != nil {
		panic(err)
	}
	router := CreateRouter()
	http.Handle("/", router)
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(*port), nil))
}

func CreateRouter() *mux.Router {

	router := mux.NewRouter()
	xhr := router.PathPrefix("/xhr/{key:[0-9]+}").Subrouter()
	xhr.HandleFunc("/roster.json", rosterHandler)
	xhr.HandleFunc("/submit-vote", voteHandler)
	xhr.HandleFunc("/advance-round", roundAdvanceHandler)
	xhr.HandleFunc("/commit-new-roster", newRosterHandler)

	websockets.RegisterRoutes(router.PathPrefix("ws").Subrouter())

	router.HandleFunc("/rosterlist.json", rosterListHandler)
	router.HandleFunc("/templates", templateHandler)
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	router.PathPrefix("/").HandlerFunc(indexHandler)

	return router
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "static/index.html")
}

func templateHandler(w http.ResponseWriter, r *http.Request) {
	ts, err := templates.LoadTemplatesFromDisk()
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

func handleBadRequest(w http.ResponseWriter, message string) {
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte("400 - " + message))
}

func voteHandler(w http.ResponseWriter, r *http.Request) {
	key := mux.Vars(r)["key"]
	voteRoster, err := contest.ParseRoster(r.Body)
	if err != nil {
		return
	}
	scoreRoster, err := GetRoster(GetScoreKey(key))
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
		SetRoster(GetScoreKey(key), scoreRoster)
		SetRoster(key, roster)
		websockets.TriggerUpdate(key)
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
	scoreRoster, err := GetRoster(GetScoreKey(key))
	if err != nil {
		return
	}
	if bytes.Compare(id, scoreRoster.UUID) == 0 {
		scoreRoster.AdvanceRound()
		SetRoster(key, scoreRoster)
		SetRoster(GetScoreKey(key), scoreRoster)
		websockets.TriggerUpdate(key)
	}
}

func newRosterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		handleBadRequest(w, "/commit-new-roster called with "+r.Method+". Ignoring")
		return
	}
	b := &bytes.Buffer{}
	b.ReadFrom(r.Body)
	r.Body.Close()

	roster := &contest.Roster{}
	err := json.Unmarshal(b.Bytes(), roster)
	if err != nil {
		handleBadRequest(w, "Bad data in /commit-new-roster: "+err.Error())
		return
	}

	key := CreateKey()
	SetRoster(key, roster)
	SetRoster(GetScoreKey(key), roster)

	w.Write([]byte(key))

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
