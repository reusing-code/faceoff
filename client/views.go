package main

import (
	"bytes"
	"html/template"
	"math/rand"
	"net/http"
	"strconv"
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

	contenderCount := 0
	if remoteRoster.ActiveRound >= 0 {
		contenderCount = len(remoteRoster.Rounds[remoteRoster.ActiveRound].Matches) * 2
	}

	data := struct {
		Round          int
		ContenderCount int
	}{
		remoteRoster.ActiveRound + 1,
		contenderCount,
	}

	renderTemplate("admin", data)

	setActiveNavItem("admin-link")

	d := dom.GetWindow().Document()

	if remoteRoster.ActiveRound >= 0 {
		btnA := d.GetElementByID("btn-advance-round").(*dom.HTMLButtonElement)
		btnA.AddEventListener("click", false, func(event dom.Event) {
			go func() {
				http.Post(createParameterizedRequestURL("/advance-round"), "POST", bytes.NewReader(remoteRoster.UUID))
				route("/bracket", true)
			}()
		})
	}

	btnNew := d.GetElementByID("btn-new-tournament").(*dom.HTMLButtonElement)
	btnNew.AddEventListener("click", false, func(event dom.Event) {
		route("/new", true)
	})
}

func bracketView() {
	scoreRoster, err := getRosterFromServer()
	if err != nil {
		panic(err)
	}

	renderTemplate("bracket", scoreRoster)
	setActiveNavItem("bracket-link")

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
	setActiveNavItem("vote-link")
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
	setActiveNavItem("vote-link")

	_, err := locstor.GetItem("currentResultsTransmitted")
	if err != nil {
		if _, ok := err.(locstor.ItemNotFoundError); ok {
			roster, err := locstor.GetItem("currentRoster")
			if err != nil {
				panic(err)
			}
			r, err := http.Post(createParameterizedRequestURL("submit-vote"), "application/json", strings.NewReader(roster))
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

func newBracketView() {
	renderTemplate("newbracket", nil)
	handler := func(ev dom.Event) {
		count, _ := strconv.Atoi(ev.Target().InnerHTML())
		showContestantInputs(count)
	}
	contestantButtons := dom.GetWindow().Document().GetElementsByClassName("btn-contestant-number")
	for _, button := range contestantButtons {
		button.AddEventListener("click", false, handler)
	}
}

func showContestantInputs(count int) {
	d := dom.GetWindow().Document()
	formDiv := d.GetElementByID("contestant-input-elements").(*dom.HTMLDivElement)

	t := template.New("contestant-input")
	t = template.Must(t.Parse(ts.Templates["snippets/contestant-input"]))

	nums := make([]int, count)
	for i := 0; i < count; i++ {
		nums[i] = i + 1
	}
	data := struct {
		Nums []int
	}{
		nums,
	}

	buf := &bytes.Buffer{}
	err := t.Execute(buf, data)
	if err != nil {
		println(err.Error())
	}

	formDiv.SetInnerHTML(buf.String())
	d.GetElementByID("form-contestant-names").AddEventListener("submit", false, func(event dom.Event) {
		event.PreventDefault()
		result := dom.GetWindow().Confirm("Erstellen eines neuen Wettbewerb ersetzt den aktuellen! Fortfahren?")
		if !result {
			return
		}
		contestants := make([]string, count+1)
		contestants[0] = d.GetElementByID("name-input").(*dom.HTMLInputElement).Value
		for i, input := range d.GetElementsByClassName("contestant-input") {
			contestants[i+1] = input.(*dom.HTMLInputElement).Value
		}
		if d.GetElementByID("randomize-input").(*dom.HTMLInputElement).Checked {
			rand.Shuffle(count, func(i, j int) {
				contestants[i], contestants[j] = contestants[j], contestants[i]
			})
		}
		go commitNewRoster(contestants)
	})
}

func bracketCreatedView(name string, newID string) {
	locstor.SetItem("currentBracketKey", newID)

	url := dom.GetWindow().Location().Origin + "/" + newID
	data := struct {
		Name string
		ID   string
		URL  string
	}{
		Name: name,
		ID:   newID,
		URL:  url,
	}
	renderTemplate("bracketcreated", data)
	d := dom.GetWindow().Document()
	btncpy := d.GetElementByID("btn-cpy").(*dom.HTMLButtonElement)
	btncpy.AddEventListener("click", false, func(event dom.Event) {
		dummy := d.GetElementByID("url-cpy-dummy").(*dom.HTMLInputElement)
		dummy.Class().Remove("invisible")
		dummy.Select()
		d.Underlying().Call("execCommand", "Copy")
		dummy.Class().Add("invisible")
	})

	btncontest := d.GetElementByID("btn-goto-bracket").(*dom.HTMLButtonElement)
	btncontest.AddEventListener("click", false, func(event dom.Event) {
		route("/bracket", true)
	})

}

func welcomeView() {
	locstor.RemoveItem("currentBracketKey")
	renderTemplate("welcome", nil)
	d := dom.GetWindow().Document()
	d.GetElementByID("button-new-bracket").(*dom.HTMLButtonElement).AddEventListener("click", false, func(event dom.Event) {
		route("/new", true)
	})

	d.GetElementByID("button-submit-key").(*dom.HTMLButtonElement).AddEventListener("click", false, func(event dom.Event) {
		event.PreventDefault()
		key := d.GetElementByID("input-key").(*dom.HTMLInputElement).Value
		key = strings.TrimSpace(key)
		locstor.SetItem("currentBracketKey", key)
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
