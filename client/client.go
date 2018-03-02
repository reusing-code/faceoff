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
	d.GetElementByID("app").SetInnerHTML("<p>Baut das doch bitte!</p>")

	response, _ := http.Get("/templates")
	buf := &bytes.Buffer{}
	buf.ReadFrom(response.Body)
	response.Body.Close()
	var err error
	ts, err = faceoff.LoadTemplatesFromGob(buf.Bytes())
	if err != nil {
		d.GetElementByID("app").AppendChild(d.CreateTextNode("Error: " + err.Error()))
	}

	votingView()

}

func votingView() {
	rosterStr, err := locstor.GetItem("currentRoster")
	if err != nil {
		if _, ok := err.(locstor.ItemNotFoundError); ok {
			r, err := http.Get("/roster.json")
			if err != nil {
				panic(err)
			}
			b, err := ioutil.ReadAll(r.Body)
			if err != nil {
				panic(err)
			}
			rosterStr = string(b)
			locstor.SetItem("currentRoster", rosterStr)
		} else {
			panic(err)
		}
	}
	currentRoster = &faceoff.Roster{}
	json.Unmarshal([]byte(rosterStr), currentRoster)

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
	})
	btnB := d.GetElementByID("btn-contenderB").(*dom.HTMLButtonElement)
	btnB.AddEventListener("click", false, func(event dom.Event) {
		m.WinB()
		saveRoster()
	})
}

func saveRoster() {
	b, err := json.Marshal(currentRoster)
	if err != nil {
		panic(err)
	}
	locstor.SetItem("currentRoster", string(b))
	votingView()
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
