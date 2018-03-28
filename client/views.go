package main

import (
	"bytes"
	"html/template"
	"math/rand"
	"net/http"
	"strconv"
	"strings"

	"github.com/gopherjs/websocket/websocketjs"

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

func adminView(remoteRoster *faceoff.Roster) {
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
				http.Post(createParameterizedXHRRequestURL("/advance-round"), "POST", bytes.NewReader(remoteRoster.UUID))
				route("/bracket", true)
			}()
		})
	}

	btnNew := d.GetElementByID("btn-new-tournament").(*dom.HTMLButtonElement)
	btnNew.AddEventListener("click", false, func(event dom.Event) {
		route("/new", true)
	})
}

func bracketView(remoteRoster *faceoff.Roster) {
	activeRoster := getActiveVoteRoster(remoteRoster)
	m := getNextMatch(activeRoster)
	data := struct {
		Name             string
		CloseRoundActive bool
		VoteActive       bool
		BracketClosed    bool
		CurrentVotes     int
	}{
		Name:             activeRoster.Name,
		CloseRoundActive: m == nil && remoteRoster.ActiveRound >= 0,
		VoteActive:       m != nil,
		BracketClosed:    remoteRoster.ActiveRound < 0,
		CurrentVotes:     remoteRoster.CurrentVotes,
	}
	renderTemplate("bracket", data)
	setActiveNavItem("bracket-link")

	js.Global.Call("jQuery", "#bracket").Call("bracket", getBracketOptions(remoteRoster))

	btnA := dom.GetWindow().Document().GetElementByID("btn-vote")
	if btnA != nil {
		btnA.AddEventListener("click", false, func(event dom.Event) {
			route("/vote", true)
		})
	}

	btnClose := dom.GetWindow().Document().GetElementByID("btn-close-vote")
	if btnClose != nil {
		btnClose.AddEventListener("click", false, func(event dom.Event) {
			result := dom.GetWindow().Confirm("Beendet die Runde für alle! Fortfahren?")
			if !result {
				return
			}
			go func() {
				http.Post(createParameterizedXHRRequestURL("/advance-round"), "POST", bytes.NewReader(remoteRoster.UUID))
				route("/bracket", false)
			}()
		})
	}

	btnRefresh := dom.GetWindow().Document().GetElementByID("btn-bracket")
	if btnRefresh != nil {
		btnRefresh.AddEventListener("click", false, func(event dom.Event) {
			route("/bracket", false)
		})
		btnRefresh.(*dom.HTMLButtonElement).Disabled = true
	}

	var err error
	if websocket == nil {
		websocket, err = websocketjs.New(getWebsocketURL())
		if err != nil {
			println(err)
			return
		}
		websocket.AddEventListener("message", false, func(ev *js.Object) {
			data := ev.Get("data").Interface().(string)
			if data == "refresh" {
				if dom.GetWindow().Location().Pathname == "/bracket" {
					route("/bracket", false)
				}
			}
		})
		websocket.AddEventListener("close", false, func(ev *js.Object) {
			btnRefresh := dom.GetWindow().Document().GetElementByID("btn-bracket")
			if btnRefresh != nil {
				btnRefresh.(*dom.HTMLButtonElement).Disabled = false
			}
		})
		websocket.AddEventListener("error", false, func(ev *js.Object) {
			btnRefresh := dom.GetWindow().Document().GetElementByID("btn-bracket")
			if btnRefresh != nil {
				btnRefresh.(*dom.HTMLButtonElement).Disabled = false
			}
		})
	}
}

func votingView(remoteRoster *faceoff.Roster) {
	currentRoster := getActiveVoteRoster(remoteRoster)

	m := getNextMatch(currentRoster)
	if m == nil {
		showVotingFinished()
		return
	}
	data := matchViewData{
		ContenderA: m.Contenders[faceoff.A],
		ContenderB: m.Contenders[faceoff.B],
		RoundNum:   remoteRoster.ActiveRound + 1,
		MatchNum:   m.Num + 1,
	}
	showMatch(currentRoster, data, m)
}

func showMatch(roster *faceoff.Roster, data matchViewData, m *faceoff.Match) {
	renderTemplate("matchvote", data)
	setActiveNavItem("vote-link")
	d := dom.GetWindow().Document()
	btnA := d.GetElementByID("btn-contenderA").(*dom.HTMLButtonElement)
	btnA.AddEventListener("click", false, func(event dom.Event) {
		m.WinA()
		saveRoster(roster)
		route("/vote", false)
	})
	btnB := d.GetElementByID("btn-contenderB").(*dom.HTMLButtonElement)
	btnB.AddEventListener("click", false, func(event dom.Event) {
		m.WinB()
		saveRoster(roster)
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
			r, err := http.Post(createParameterizedXHRRequestURL("/submit-vote"), "application/json", strings.NewReader(roster))
			if err != nil {
				panic(err)
			}
			if r.StatusCode >= 200 && r.StatusCode < 300 {
				locstor.SetItem("currentResultsTransmitted", "TRUE")
			}
		}
	}

	btnA := dom.GetWindow().Document().GetElementByID("btn-bracket")
	btnA.AddEventListener("click", false, func(event dom.Event) {
		route("/bracket", true)
	})
}

func newBracketView() {
	renderTemplate("newbracket", nil)
	setActiveNavItem("new-link")
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

	oldNameInput := d.GetElementByID("name-input")
	oldName := ""
	if oldNameInput != nil {
		oldName = oldNameInput.(*dom.HTMLInputElement).Value
	}
	oldInputElements := d.GetElementsByClassName("contestant-input")
	oldValues := make([]string, 0, len(oldInputElements))
	for _, ele := range oldInputElements {
		oldValues = append(oldValues, ele.(*dom.HTMLInputElement).Value)
	}

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

	newNameInput := d.GetElementByID("name-input")
	if newNameInput != nil {
		newNameInput.(*dom.HTMLInputElement).Value = oldName
	}
	newInputElements := d.GetElementsByClassName("contestant-input")
	for i, ele := range newInputElements {
		if i < len(oldValues) {
			ele.(*dom.HTMLInputElement).Value = oldValues[i]
		}
	}

	d.GetElementByID("form-contestant-names").AddEventListener("submit", false, func(event dom.Event) {
		event.PreventDefault()
		contestants := make([]string, count+1)
		contestants[0] = d.GetElementByID("name-input").(*dom.HTMLInputElement).Value
		for i, input := range d.GetElementsByClassName("contestant-input") {
			contestants[i+1] = input.(*dom.HTMLInputElement).Value
		}
		if d.GetElementByID("randomize-input").(*dom.HTMLInputElement).Checked {
			rand.Shuffle(count, func(i, j int) {
				contestants[i+1], contestants[j+1] = contestants[j+1], contestants[i+1]
			})
		}
		go commitNewRoster(contestants)
	})
}

func bracketCreatedView(name string, newID string) {
	setCurrentBracket(newID)

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
	setCurrentBracket("")
	renderTemplate("welcome", nil)
	d := dom.GetWindow().Document()
	d.GetElementByID("button-new-bracket").(*dom.HTMLButtonElement).AddEventListener("click", false, func(event dom.Event) {
		route("/new", true)
	})

	d.GetElementByID("button-submit-key").(*dom.HTMLButtonElement).AddEventListener("click", false, func(event dom.Event) {
		event.PreventDefault()
		key := d.GetElementByID("input-key").(*dom.HTMLInputElement).Value
		key = strings.TrimSpace(key)
		setCurrentBracket(key)
		route("/bracket", true)
	})

}

func imprintView() {
	renderTemplate("imprint", nil)
}

func listBracketView() {
	data, err := getBracketListFromServer()
	if err != nil {
		println(err)
		return
	}
	renderTemplate("bracketlist", data)
	setActiveNavItem("list-link")

	handler := func(ev dom.Event) {
		ev.PreventDefault()
		key := ev.Target().ID()
		setCurrentBracket(key)
		route("/bracket", true)
	}
	listItems := dom.GetWindow().Document().GetElementsByClassName("bracket-list-item")
	for _, button := range listItems {
		button.AddEventListener("click", false, handler)
	}
}

func renderTemplate(templateName string, data interface{}) {
	t := template.New("base")
	t = template.Must(t.Parse(ts.Templates["layout/base"]))
	t = template.Must(t.Parse(ts.Templates["layout/footer"]))
	t = template.Must(t.Parse(ts.Templates[templateName]))

	buf := &bytes.Buffer{}
	err := t.Execute(buf, data)
	if err != nil {
		println(err.Error())
	}
	d := dom.GetWindow().Document()
	d.GetElementByID("app").SetInnerHTML(buf.String())

	brand := d.GetElementByID("navbar-brand").(*dom.HTMLAnchorElement)
	brand.AddEventListener("click", false, func(event dom.Event) {
		event.PreventDefault()
		route("/welcome", true)
	})
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
	new := d.GetElementByID("new-link").(*dom.HTMLAnchorElement)
	new.AddEventListener("click", false, func(event dom.Event) {
		event.PreventDefault()
		route("/new", true)
	})
	list := d.GetElementByID("list-link").(*dom.HTMLAnchorElement)
	list.AddEventListener("click", false, func(event dom.Event) {
		event.PreventDefault()
		route("/list", true)
	})
	imprint := d.GetElementByID("a-imprint").(*dom.HTMLAnchorElement)
	imprint.AddEventListener("click", false, func(event dom.Event) {
		event.PreventDefault()
		route("/impressum", true)
	})
	contact := d.GetElementByID("a-contact").(*dom.HTMLAnchorElement)
	contact.AddEventListener("click", false, func(event dom.Event) {
		event.PreventDefault()
		route("/impressum", true)
	})

}
