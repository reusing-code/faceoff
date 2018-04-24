package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/go-humble/locstor"

	"github.com/gopherjs/websocket/websocketjs"

	"github.com/gopherjs/gopherjs/js"

	"github.com/reusing-code/faceoff/shared/contest"
	"github.com/reusing-code/faceoff/shared/templates"

	"honnef.co/go/js/dom"
)

var ts *templates.TemplateSet
var websocket *websocketjs.WebSocket
var websocketSate SocketState = Inactive

type clientContest struct {
	*contest.Contest
	SubmittedVoteRound int
}

type SocketState int

const (
	Inactive SocketState = iota
	Active
	Error
	Offline
)

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

	dom.GetWindow().AddEventListener("offline", false, func(ev dom.Event) {
		if websocket != nil {
			websocket.Close()
			websocket = nil
		}
		websocketSate = Offline
		renderWebsocketState()
	})

	if getCurrentBracketKey() != "" {
		createWebsocket()
	}

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

func getCurrentLocalContest() (*clientContest, error) {
	key := getCurrentBracketKey()
	return getLocalContest(key)
}

func getLocalContest(key string) (*clientContest, error) {
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
	localContest, err := getCurrentLocalContest()
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
	key := getCurrentBracketKey()
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
	oldKey := getCurrentBracketKey()
	if oldKey == key {
		//nothing to do
		return
	}
	if key == "" {
		locstor.RemoveItem("currentBracketKey")
	} else {
		locstor.SetItem("currentBracketKey", key)
	}
	closeWebsocket()
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
	localContest, err := getCurrentLocalContest()

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

func addParticipatingContests(list *contest.ContestList) {
	processedContests := make(map[string]contest.ContestDescription)

	localContests := getAllLocalContests()
	list.Participating = localContests
	for _, loc := range localContests {
		processedContests[loc.Key] = loc
	}

	tmpOpen := list.Open[:0]
	for _, open := range list.Open {
		if _, ok := processedContests[open.Key]; !ok {
			// bracket not in local list
			tmpOpen = append(tmpOpen, open)
		}
	}
	list.Open = tmpOpen

	sort.Slice(list.Participating, func(i, j int) bool {
		return list.Participating[i].TimeStamp > list.Participating[j].TimeStamp
	})
}

func getAllLocalContests() []contest.ContestDescription {
	result := make([]contest.ContestDescription, 0)
	len, _ := locstor.Length()
	for i := 0; i < len; i++ {
		key, _ := locstor.Key(strconv.Itoa(i))
		_, err := strconv.Atoi(key)
		if err != nil {
			continue
		}
		con, err := getLocalContest(key)
		if err != nil {
			continue
		}
		if con.ActiveRound < 0 {
			continue
		}
		result = append(result, contest.ContestDescription{Key: key, Name: con.Name,
			IsAdmin: con.AdminKey != "", TimeStamp: con.CreatedTimeStamp})
	}
	return result
}

func createWebsocket() {
	if websocket != nil && websocketSate == Active {
		return
	}
	if getCurrentBracketKey() == "" {
		return
	}
	var err error
	websocket, err = websocketjs.New(getWebsocketURL())
	if err != nil {
		println(err)
		return
	}
	websocketSate = Active
	websocket.AddEventListener("message", false, func(ev *js.Object) {
		data := ev.Get("data").Interface().(string)
		if data == "refresh" {
			if strings.Contains(dom.GetWindow().Location().Pathname, "/bracket") {
				route("/bracket", false)
			}
		}
	})
	websocket.AddEventListener("close", false, func(ev *js.Object) {
		println("Websocket closed")
		websocketSate = Inactive
		websocket = nil
		renderWebsocketState()
	})
	websocket.AddEventListener("error", false, func(ev *js.Object) {
		println("Websocket error")
		websocketSate = Error
		websocket = nil
		renderWebsocketState()
	})
}

func closeWebsocket() {
	if websocket != nil {
		websocket.Close()
		websocket = nil
	}
	websocketSate = Inactive
}
