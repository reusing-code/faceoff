package main

import (
	"bytes"
	"encoding/json"
	"fmt"
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
