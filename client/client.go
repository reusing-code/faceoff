package main

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/reusing-code/faceoff"

	"honnef.co/go/js/dom"
)

func main() {
	d := dom.GetWindow().Document()
	d.GetElementByID("app").SetInnerHTML("<p>Baut das doch bitte!</p>")

	// t := template.New("base")
	// t = template.Must(t.ParseFiles("templates/layout/base.tmpl.html", "templates/matchvote.tmpl.html"))

	// buf := &bytes.Buffer{}
	// err := t.Execute(buf, nil)
	// if err != nil {
	// 	println(err.Error())
	// }
	// d.GetElementByID("app").SetInnerHTML(buf.String())
	response, _ := http.Get("/templates")
	buf := &bytes.Buffer{}
	buf.ReadFrom(response.Body)
	response.Body.Close()
	ts, err := faceoff.LoadTemplatesFromGob(buf.Bytes())
	if err != nil {
		d.GetElementByID("app").AppendChild(d.CreateTextNode("Error: " + err.Error()))
	}
	b, _ := json.Marshal(ts)
	text := d.CreateTextNode(string(b))
	d.GetElementByID("app").AppendChild(text)

	json.Unmarshal(b, ts)

}
