package main

import (
	"bytes"
	"html/template"
	"net/http"
	"strings"

	"github.com/go-humble/locstor"
	"github.com/gopherjs/gopherjs/js"
	"github.com/reusing-code/faceoff"
	"honnef.co/go/js/dom"
)

type matchViewData struct {
	ContenderA string
	ContenderB string
	RoundNum   int
	MatchNum   int
}

func adminView() {
	remoteRoster, err := getRosterFromServer()
	if err != nil {
		panic(err)
	}

	currentRound := len(remoteRoster.Rounds) - 1
	contenderCount := len(remoteRoster.Rounds[currentRound].Matches) * 2

	data := struct {
		Round          int
		ContenderCount int
	}{
		currentRound + 1,
		contenderCount,
	}

	renderTemplate("admin", data)
	d := dom.GetWindow().Document()
	btnA := d.GetElementByID("btn-advance-round").(*dom.HTMLButtonElement)
	if contenderCount > 2 {
		btnA.AddEventListener("click", false, func(event dom.Event) {
			go http.Post("/advance-round", "POST", bytes.NewReader(remoteRoster.UUID))
			route("/bracket", true)
		})
	} else {
		btnA.Disabled = true
	}
}

func bracketView() {
	scoreRoster, err := getScoreRosterFromServer()
	if err != nil {
		panic(err)
	}

	renderTemplate("bracket", nil)

	js.Global.Call("jQuery", "#bracket").Call("bracket", getBracketOptions(scoreRoster))

	btnA := dom.GetWindow().Document().GetElementByID("btn-vote").(*dom.HTMLButtonElement)
	btnA.AddEventListener("click", false, func(event dom.Event) {
		route("/vote", true)
	})
	btnB := dom.GetWindow().Document().GetElementByID("btn-bracket").(*dom.HTMLButtonElement)
	btnB.AddEventListener("click", false, func(event dom.Event) {
		route("/bracket", false)
	})

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
		} else {
			locstor.RemoveItem("currentResultsTransmitted")
		}
	} else {
		locstor.RemoveItem("currentResultsTransmitted")
	}

	saveRoster()

	matchShown := false
	r := currentRoster.Rounds[len(currentRoster.Rounds)-1]
	for i, m := range r.Matches {
		if m.Winner == faceoff.NONE {
			data := matchViewData{
				ContenderA: m.Contenders[faceoff.A],
				ContenderB: m.Contenders[faceoff.B],
				RoundNum:   len(currentRoster.Rounds),
				MatchNum:   i + 1,
			}
			showMatch(data, m)
			matchShown = true
			break
		}
	}
	if !matchShown {
		showVotingFinished()
	}

}

func showMatch(data matchViewData, m *faceoff.Match) {
	renderTemplate("matchvote", data)
	d := dom.GetWindow().Document()
	btnA := d.GetElementByID("btn-contenderA").(*dom.HTMLButtonElement)
	btnA.AddEventListener("click", false, func(event dom.Event) {
		m.WinA()
		saveRoster()
		route("/vote", false)
	})
	btnB := d.GetElementByID("btn-contenderB").(*dom.HTMLButtonElement)
	btnB.AddEventListener("click", false, func(event dom.Event) {
		m.WinB()
		saveRoster()
		route("/vote", false)
	})
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

	btnA := dom.GetWindow().Document().GetElementByID("btn-bracket").(*dom.HTMLButtonElement)
	btnA.AddEventListener("click", false, func(event dom.Event) {
		route("/bracket", true)
	})
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

	bracket := d.GetElementByID("bracket-link").(*dom.HTMLAnchorElement)
	bracket.AddEventListener("click", false, func(event dom.Event) {
		event.PreventDefault()
		route("/bracket", true)
	})
	vote := d.GetElementByID("vote-link").(*dom.HTMLAnchorElement)
	vote.AddEventListener("click", false, func(event dom.Event) {
		event.PreventDefault()
		route("/vote", true)
	})
	admin := d.GetElementByID("admin-link").(*dom.HTMLAnchorElement)
	admin.AddEventListener("click", false, func(event dom.Event) {
		event.PreventDefault()
		route("/admin", true)
	})

}
