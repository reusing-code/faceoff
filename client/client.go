package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gopherjs/websocket/websocketjs"

	"github.com/gopherjs/gopherjs/js"

	"github.com/go-humble/locstor"

	"github.com/reusing-code/faceoff/shared/contest"
	"github.com/reusing-code/faceoff/shared/templates"

	"honnef.co/go/js/dom"
)

var ts *templates.TemplateSet
var websocket *websocketjs.WebSocket

type clientContest struct {
	*contest.Contest
	SubmittedVoteRound int
}

func main() {
	d := dom.GetWindow().Document()

	response, _ := http.Get("/templates")
	buf := &bytes.Buffer{}
	buf.ReadFrom(response.Body)
	response.Body.Close()
	var err error
	ts, err = templates.LoadTemplatesFromGob(buf.Bytes())
	if err != nil {
		d.GetElementByID("app").AppendChild(d.CreateTextNode("Error: " + err.Error()))
	}

	js.Global.Call("addEventListener", "popstate", func(event *js.Object) {
		route("", false)
	})

	route("", true)
}

func saveLocalContest(contest *clientContest) {
	b, err := json.Marshal(contest)
	if err != nil {
		panic(err)
	}
	err = locstor.SetItem(getCurrentBracketKey(), string(b))
	if err != nil {
		panic(err)
	}
}

func getLocalContest() (*clientContest, error) {
	key := getCurrentBracketKey()
	rosterStr, err := locstor.GetItem(key)
	if _, ok := err.(locstor.ItemNotFoundError); ok {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	result := &clientContest{}
	err = json.Unmarshal([]byte(rosterStr), result)
	return result, err

}

func getActiveVoteRoster(remoteRoster *contest.Contest) *clientContest {
	localContest, err := getLocalContest()
	if err != nil {
		localContest = nil
	}
	adminKey := ""
	if localContest != nil {
		if remoteRoster.ActiveRound == localContest.ActiveRound {
			return localContest
		}
		adminKey = localContest.AdminKey
	}
	contest := &clientContest{Contest: remoteRoster, SubmittedVoteRound: -1}
	contest.AdminKey = adminKey
	saveLocalContest(contest)
	return contest
}

func getRosterFromServer() (*contest.Contest, error) {
	r, err := http.Get(createParameterizedXHRRequestURL("/roster.json"))
	if err != nil {
		return nil, err
	}
	if r.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("404")
	}
	result, err := contest.ParseRoster(r.Body)
	return result, err
}

func commitNewRoster(roster *contest.Contest) {
	marshalled, err := json.Marshal(roster)
	if err != nil {
		println(err)
		return
	}
	r, err := http.Post(createParameterizedXHRRequestURL("/commit-new-roster"), "POST", bytes.NewReader(marshalled))
	if err != nil {
		println(err)
		return
	}

	if r.StatusCode != http.StatusOK {
		println("commitNewRoster: unsuccsessful reply")
		return
	}
	buf := &bytes.Buffer{}
	buf.ReadFrom(r.Body)
	r.Body.Close()
	setCurrentBracket(buf.String())
	saveLocalContest(&clientContest{Contest: roster, SubmittedVoteRound: -1})
	bracketCreatedView(roster.Name)
}

func createParameterizedXHRRequestURL(ressoure string) string {
	currentKey := getCurrentBracketKey()
	if currentKey == "" {
		currentKey = "0"
	}
	return "/xhr/" + currentKey + ressoure
}

func getWebsocketURL() string {
	key, _ := locstor.GetItem("currentBracketKey")
	buf := bytes.Buffer{}
	if dom.GetWindow().Location().Protocol == "https:" {
		buf.WriteString("wss://")
	} else {
		buf.WriteString("ws://")
	}
	buf.WriteString(dom.GetWindow().Location().Hostname)
	buf.WriteString(":")
	buf.WriteString(dom.GetWindow().Location().Port)
	buf.WriteString("/ws/")
	buf.WriteString(key)
	return buf.String()
}

func setCurrentBracket(key string) {
	if key == "" {
		locstor.RemoveItem("currentBracketKey")
	} else {
		locstor.SetItem("currentBracketKey", key)
	}
	if websocket != nil {
		websocket.Close()
		websocket = nil
	}
}

func getCurrentBracketKey() string {
	key, err := locstor.GetItem("currentBracketKey")
	if err != nil {
		return ""
	}
	return key
}

func getNextMatch(cont *clientContest) *contest.Match {
	if cont.ActiveRound < 0 {
		return nil
	}
	r := cont.Rounds[cont.ActiveRound]
	for i, m := range r.Matches {
		if m.Winner == contest.NONE {
			m.Num = i
			return m
		}
	}
	return nil
}

func getBracketListFromServer() (*contest.ContestList, error) {

	r, err := http.Get("/rosterlist.json")
	if err != nil {
		return nil, err
	}
	if r.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("404")
	}
	buf := &bytes.Buffer{}
	buf.ReadFrom(r.Body)
	r.Body.Close()
	list := &contest.ContestList{}
	err = json.Unmarshal(buf.Bytes(), list)
	return list, err
}

func submitVote() {
	localContest, err := getLocalContest()

	if err != nil {
		return
	}

	if localContest.SubmittedVoteRound < localContest.ActiveRound {
		b, err := json.Marshal(localContest.Contest)
		if err != nil {
			panic(err)
		}
		r, err := http.Post(createParameterizedXHRRequestURL("/submit-vote"), "application/json", bytes.NewBuffer(b))
		if err != nil {
			panic(err)
		}
		if r.StatusCode >= 200 && r.StatusCode < 300 {
			localContest.SubmittedVoteRound = localContest.ActiveRound
			saveLocalContest(localContest)
		}
	}

}
