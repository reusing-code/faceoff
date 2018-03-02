package main

import (
	"bytes"
	"encoding/json"
	"html/template"
	"io/ioutil"
	"net/http"

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

	r := currentRoster.Rounds[len(currentRoster.Rounds)-1]
	for _, m := range r.Matches {
		if m.Winner == faceoff.NONE {
			showMatch(m)
			break
		}
	}
}

func showMatch(m *faceoff.Match) {
	t := template.New("base")
	t = template.Must(t.Parse(ts.Templates["layout/base"]))
	t = template.Must(t.Parse(ts.Templates["matchvote"]))

	buf := &bytes.Buffer{}
	err := t.Execute(buf, m)
	if err != nil {
		println(err.Error())
	}
	d := dom.GetWindow().Document()
	d.GetElementByID("app").SetInnerHTML(buf.String())
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
