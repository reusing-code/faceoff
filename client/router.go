package main

import (
	"strconv"
	"strings"

	"github.com/reusing-code/faceoff"

	"github.com/gopherjs/gopherjs/js"
	"honnef.co/go/js/dom"
)

type viewFunc func(bracket *faceoff.Roster)

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

	if addToHistory {
		key := getCurrentBracketKey()
		newPath := path
		if len(key) > 0 {
			newPath = "/" + key + path
		}
		history := js.Global.Get("history")
		history.Call("pushState", nil, "", newPath)
	}

	if path == "/welcome" {
		go welcomeView()
		return
	} else if path == "/new" {
		go newBracketView()
		return
	} else if path == "/list" {
		go listBracketView()
		return
	} else if path == "/admin" {
		go routeWithBracket(adminView)
		return
	} else if path == "/vote" {
		go routeWithBracket(votingView)
		return
	} else {
		go routeWithBracket(bracketView)
		return
	}

}

func routeWithBracket(view viewFunc) {
	roster, err := getRosterFromServer()

	bracketValid := err == nil

	if !bracketValid {
		route("/welcome", true)
		return
	}

	view(roster)
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
