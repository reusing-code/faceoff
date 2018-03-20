package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gopherjs/websocket/websocketjs"

	"github.com/gopherjs/gopherjs/js"

	"github.com/go-humble/locstor"

	"github.com/reusing-code/faceoff"

	"honnef.co/go/js/dom"
)

var ts *faceoff.TemplateSet
var websocket *websocketjs.WebSocket

func main() {
	d := dom.GetWindow().Document()

	response, _ := http.Get("/templates")
	buf := &bytes.Buffer{}
	buf.ReadFrom(response.Body)
	response.Body.Close()
	var err error
	ts, err = faceoff.LoadTemplatesFromGob(buf.Bytes())
	if err != nil {
		d.GetElementByID("app").AppendChild(d.CreateTextNode("Error: " + err.Error()))
	}

	js.Global.Call("addEventListener", "popstate", func(event *js.Object) {
		route("", false)
	})

	route("", true)
}

func saveRoster(roster *faceoff.Roster) {
	b, err := json.Marshal(roster)
	if err != nil {
		panic(err)
	}
	locstor.SetItem("currentRoster", string(b))
}

func loadRoster() (*faceoff.Roster, error) {
	rosterStr, err := locstor.GetItem("currentRoster")
	if _, ok := err.(locstor.ItemNotFoundError); ok {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	result := &faceoff.Roster{}
	err = json.Unmarshal([]byte(rosterStr), result)
	return result, err

}

func getActiveVoteRoster(remoteRoster *faceoff.Roster) *faceoff.Roster {
	localRoster, err := loadRoster()
	if err != nil {
		localRoster = nil
	}

	currentRoster := remoteRoster
	if localRoster != nil {
		if bytes.Compare(localRoster.UUID, remoteRoster.UUID) == 0 {
			currentRoster = localRoster
		} else {
			locstor.RemoveItem("currentResultsTransmitted")
		}
	} else {
		locstor.RemoveItem("currentResultsTransmitted")
	}

	saveRoster(currentRoster)
	return currentRoster
}

func getRosterFromServer() (*faceoff.Roster, error) {
	r, err := http.Get(createParameterizedRequestURL("/roster.json"))
	if err != nil {
		return nil, err
	}
	if r.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("404")
	}
	result, err := faceoff.ParseRoster(r.Body)
	return result, err
}

func commitNewRoster(contestants []string) {
	marshalled, err := json.Marshal(contestants)
	if err != nil {
		println(err)
		return
	}
	r, err := http.Post(createParameterizedRequestURL("/commit-new-roster"), "POST", bytes.NewReader(marshalled))
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
	bracketCreatedView(contestants[0], buf.String())
}

func createParameterizedRequestURL(ressoure string) string {
	currentKey, err := locstor.GetItem("currentBracketKey")
	if err != nil {
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

func getNextMatch(roster *faceoff.Roster) *faceoff.Match {
	if roster.ActiveRound < 0 {
		return nil
	}
	r := roster.Rounds[roster.ActiveRound]
	for i, m := range r.Matches {
		if m.Winner == faceoff.NONE {
			m.Num = i
			return m
		}
	}
	return nil
}

func getBracketListFromServer() (*faceoff.ContestList, error) {

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
	list := &faceoff.ContestList{}
	err = json.Unmarshal(buf.Bytes(), list)
	return list, err
}
