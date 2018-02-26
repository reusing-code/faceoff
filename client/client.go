package main

import (
	"bytes"
	"html/template"

	"honnef.co/go/js/dom"
)

func main() {
	d := dom.GetWindow().Document()
	d.GetElementByID("app").SetInnerHTML("<p>Baut das doch bitte!</p>")

	t := template.New("base")
	t = template.Must(t.ParseFiles("templates/layout/base.tmpl.html", "templates/matchvote.tmpl.html"))

	buf := &bytes.Buffer{}
	err := t.Execute(buf, nil)
	if err != nil {
		println(err.Error())
	}
	d.GetElementByID("app").SetInnerHTML(buf.String())
}
