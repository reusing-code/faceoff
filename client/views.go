package main

import (
	"bytes"
	"html/template"
	"math/rand"
	"net/http"
	"strconv"
	"strings"

	"github.com/gopherjs/gopherjs/js"
	"github.com/reusing-code/faceoff/shared/contest"
	"honnef.co/go/js/dom"
)

type matchViewData struct {
	ContenderA string
	ContenderB string
	RoundNum   int
	MatchNum   int
}

func bracketView(remoteRoster *contest.Contest) {
	activeRoster := getActiveVoteRoster(remoteRoster)

	isAdmin := activeRoster.AdminKey != ""

	m := getNextMatch(activeRoster)
	data := struct {
		Name             string
		CloseRoundActive bool
		VoteActive       bool
		BracketClosed    bool
		CurrentVotes     int
		IsAdmin          bool
	}{
		Name:             activeRoster.Name,
		CloseRoundActive: m == nil && remoteRoster.ActiveRound >= 0 && isAdmin,
		VoteActive:       m != nil,
		BracketClosed:    remoteRoster.ActiveRound < 0,
		CurrentVotes:     remoteRoster.CurrentVotes,
		IsAdmin:          isAdmin,
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
				http.Post(createParameterizedXHRRequestURL("/advance-round"), "POST", strings.NewReader(activeRoster.AdminKey))
				route("/bracket", false)
			}()
			btnClose.(*dom.HTMLButtonElement).Disabled = true
		})
	}

	btnRefresh := dom.GetWindow().Document().GetElementByID("btn-bracket")
	if btnRefresh != nil {
		btnRefresh.AddEventListener("click", false, func(event dom.Event) {
			route("/bracket", false)
		})
	}
}

func votingView(remoteRoster *contest.Contest) {
	currentRoster := getActiveVoteRoster(remoteRoster)

	m := getNextMatch(currentRoster)
	if m == nil {
		showVotingFinished()
		return
	}
	data := matchViewData{
		ContenderA: m.Contenders[contest.A],
		ContenderB: m.Contenders[contest.B],
		RoundNum:   remoteRoster.ActiveRound + 1,
		MatchNum:   m.Num + 1,
	}
	showMatch(currentRoster, data, m)
}

func showMatch(cont *clientContest, data matchViewData, m *contest.Match) {
	renderTemplate("matchvote", data)
	setActiveNavItem("vote-link")
	d := dom.GetWindow().Document()
	btnA := d.GetElementByID("btn-contenderA").(*dom.HTMLButtonElement)
	btnA.AddEventListener("click", false, func(event dom.Event) {
		m.WinA()
		saveLocalContest(cont)
		route("/vote", false)
	})
	btnB := d.GetElementByID("btn-contenderB").(*dom.HTMLButtonElement)
	btnB.AddEventListener("click", false, func(event dom.Event) {
		m.WinB()
		saveLocalContest(cont)
		route("/vote", false)
	})
}

func showVotingFinished() {
	renderTemplate("finishedvote", nil)
	setActiveNavItem("vote-link")

	submitVote()

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
		contestants := make([]string, count)
		name := d.GetElementByID("name-input").(*dom.HTMLInputElement).Value
		for i, input := range d.GetElementsByClassName("contestant-input") {
			contestants[i] = input.(*dom.HTMLInputElement).Value
		}
		if d.GetElementByID("randomize-input").(*dom.HTMLInputElement).Checked {
			rand.Shuffle(count, func(i, j int) {
				contestants[i], contestants[j] = contestants[j], contestants[i]
			})
		}
		private := false
		if d.GetElementByID("radio-private").(*dom.HTMLInputElement).Checked {
			private = true
		}
		roster, err := contest.CreateRoster(name, contestants, private)
		if err != nil {
			println(err)
			return
		}

		go commitNewRoster(roster)
	})
}

func bracketCreatedView(newID string) {
	cont, _ := getCurrentLocalContest()
	url := dom.GetWindow().Location().Origin + "/" + newID
	data := struct {
		Name     string
		ID       string
		URL      string
		AdminKey string
	}{
		Name:     cont.Name,
		ID:       newID,
		URL:      url,
		AdminKey: cont.AdminKey,
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

	addParticipatingContests(data)

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
	t = template.Must(t.Parse(ts.Templates["layout/fixedposition"]))
	t = template.Must(t.Parse(ts.Templates[templateName]))

	buf := &bytes.Buffer{}
	err := t.Execute(buf, data)
	if err != nil {
		println(err.Error())
	}
	d := dom.GetWindow().Document()
	d.GetElementByID("app").SetInnerHTML(buf.String())

	renderWebsocketState()

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

func renderWebsocketState() {
	t := template.New("wsstate")
	templ := "inactive"
	switch websocketSate {
	case Active:
		templ = "active"
	case Offline:
		templ = "offline"
	case Error:
		templ = "error"
	default:
		templ = "inactive"
	}
	t = template.Must(t.Parse(ts.Templates["snippets/ws-state/"+templ]))

	buf := &bytes.Buffer{}
	err := t.Execute(buf, nil)
	if err != nil {
		println(err.Error())
	}

	dom.GetWindow().Document().GetElementByID("ws-state-content").SetInnerHTML(buf.String())
}
