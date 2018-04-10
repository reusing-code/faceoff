package main

import (
	"strconv"
	"strings"

	"github.com/gopherjs/gopherjs/js"
	"github.com/reusing-code/faceoff/shared/contest"
	"honnef.co/go/js/dom"
)

type viewFunc func()
type viewFuncRoster func(bracket *contest.Contest)

type historyEntry struct {
	addToHistory bool
	path         string
}

func route(path string, addToHistory bool) {
	if path == "" {
		path = dom.GetWindow().Location().Pathname
	}

	components := strings.SplitN(path, "/", 3)
	if len(components) > 1 {
		_, err := strconv.Atoi(components[1])
		if err == nil {
			setCurrentBracket(components[1])
			if len(components) > 2 {
				path = "/" + components[2]
			} else {
				path = ""
			}
		}
	}

	if path == "" || path == "/" {
		path = "/bracket"
	}

	hist := &historyEntry{
		addToHistory: addToHistory,
		path:         path,
	}

	if path == "/welcome" {
		go routeWithOutBracket(welcomeView, hist)
		return
	} else if path == "/impressum" {
		go routeWithOutBracket(imprintView, hist)
		return
	} else if path == "/new" {
		go routeWithOutBracket(newBracketView, hist)
		return
	} else if path == "/list" {
		go routeWithOutBracket(listBracketView, hist)
		return
	} else if path == "/vote" {
		go routeWithBracket(votingView, hist)
		return
	} else {
		go routeWithBracket(bracketView, hist)
		return
	}

}

func routeWithBracket(view viewFuncRoster, hist *historyEntry) {
	roster, err := getRosterFromServer()

	bracketValid := err == nil

	if !bracketValid {
		route("/welcome", true)
		return
	}
	hist.handleHistoryEntry(true)
	view(roster)
}

func routeWithOutBracket(view viewFunc, hist *historyEntry) {
	hist.handleHistoryEntry(false)
	view()
}

func setActiveNavItem(id string) {
	d := dom.GetWindow().Document()
	nav := d.GetElementByID("navbarNav")
	navLinks := nav.GetElementsByClassName("nav-link")
	for _, link := range navLinks {
		link.Class().Remove("active")
	}

	d.GetElementByID(id).Class().Add("active")
}

func (hist *historyEntry) handleHistoryEntry(withKey bool) {
	if hist.addToHistory {
		newPath := hist.path
		if withKey {
			key := getCurrentBracketKey()
			if len(key) > 0 {
				newPath = "/" + key + hist.path
			}
		}
		history := js.Global.Get("history")
		history.Call("pushState", nil, "", newPath)
	}
}
