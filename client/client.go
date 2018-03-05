package main

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/gopherjs/gopherjs/js"

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

	js.Global.Call("addEventListener", "popstate", func(event *js.Object) {
		route("", false)
		// if event.Get("state") == nil {
		// 	route("")
		// } else {
		// 	route(event.Get("state").String())
		// }
	})

	route("", true)
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
	result, err := faceoff.ParseRoster(r.Body)
	return result, err
}

func getScoreRosterFromServer() (*faceoff.Roster, error) {
	r, err := http.Get("/roster_score.json")
	if err != nil {
		return nil, err
	}
	result, err := faceoff.ParseRoster(r.Body)
	return result, err
}
