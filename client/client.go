package main

import (
	"bytes"
	"encoding/json"
	"html/template"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/go-humble/locstor"

	"github.com/reusing-code/faceoff"

	"honnef.co/go/js/dom"
)

var ts *faceoff.TemplateSet
var currentRoster *faceoff.Roster

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

	go votingView()

}

func votingView() {
	remoteRoster, err := getRosterFromServer()
	if err != nil {
		panic(err)
	}
	localRoster, err := loadRoster()
	if err != nil {
		println(err)
		localRoster = nil
	}

	currentRoster = remoteRoster
	if localRoster != nil {
		if bytes.Compare(localRoster.UUID, remoteRoster.UUID) == 0 {
			currentRoster = localRoster
		}
	}

	saveRoster()

	matchShown := false
	r := currentRoster.Rounds[len(currentRoster.Rounds)-1]
	for _, m := range r.Matches {
		if m.Winner == faceoff.NONE {
			showMatch(m)
			matchShown = true
			break
		}
	}
	if !matchShown {
		showVotingFinished()
	}

}

func showMatch(m *faceoff.Match) {
	renderTemplate("matchvote", m)
	d := dom.GetWindow().Document()
	btnA := d.GetElementByID("btn-contenderA").(*dom.HTMLButtonElement)
	btnA.AddEventListener("click", false, func(event dom.Event) {
		m.WinA()
		saveRoster()
		go votingView()
	})
	btnB := d.GetElementByID("btn-contenderB").(*dom.HTMLButtonElement)
	btnB.AddEventListener("click", false, func(event dom.Event) {
		m.WinB()
		saveRoster()
		go votingView()
	})
}

func saveRoster() {
	b, err := json.Marshal(currentRoster)
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

func getRosterFromServer() (*faceoff.Roster, error) {
	r, err := http.Get("/roster.json")
	if err != nil {
		return nil, err
	}
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	result := &faceoff.Roster{}
	err = json.Unmarshal(b, result)
	return result, err
}

func showVotingFinished() {
	renderTemplate("finishedvote", nil)

	_, err := locstor.GetItem("currentResultsTransmitted")
	if err != nil {
		if _, ok := err.(locstor.ItemNotFoundError); ok {
			roster, err := locstor.GetItem("currentRoster")
			if err != nil {
				panic(err)
			}
			r, err := http.Post("submit-vote", "application/json", strings.NewReader(roster))
			if err != nil {
				panic(err)
			}
			if r.StatusCode >= 200 && r.StatusCode < 300 {
				locstor.SetItem("currentResultsTransmitted", "TRUE")
			}
		}
	}
}

func renderTemplate(templateName string, data interface{}) {
	t := template.New("base")
	t = template.Must(t.Parse(ts.Templates["layout/base"]))
	t = template.Must(t.Parse(ts.Templates[templateName]))

	buf := &bytes.Buffer{}
	err := t.Execute(buf, data)
	if err != nil {
		println(err.Error())
	}
	d := dom.GetWindow().Document()
	d.GetElementByID("app").SetInnerHTML(buf.String())
}
