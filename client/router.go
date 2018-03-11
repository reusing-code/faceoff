package main

import (
	"strconv"
	"strings"

	"github.com/go-humble/locstor"
	"github.com/gopherjs/gopherjs/js"
	"honnef.co/go/js/dom"
)

func route(path string, addToHistory bool) {
	if path == "" {
		path = dom.GetWindow().Location().Pathname
	}

	if path == "" || path == "/" {
		path = "/bracket"
	} else {
		stripped := strings.Replace(path, "/", "", -1)
		_, err := strconv.Atoi(stripped)
		if err == nil {
			locstor.SetItem("currentBracketKey", stripped)
			path = "/bracket"
		}
	}

	if addToHistory {
		history := js.Global.Get("history")
		history.Call("pushState", nil, "", path)
	}

	// Routes not requiring valid bracket
	if path == "/welcome" {
		go welcomeView()
		return
	} else if path == "/new" {
		go newBracketView()
		return
	}

	// Routes requiring valid current bracket
	go func() {
		_, err := getRosterFromServer()

		bracketValid := err == nil

		if !bracketValid {
			println("bracketInvalid")
			if addToHistory {
				history := js.Global.Get("history")
				history.Call("replaceState", nil, "", "/welcome")
			}
			route("/welcome", false)
			return
		}

		if path == "/admin" {
			go adminView()
			return
		} else if path == "/vote" {
			go votingView()
			return
		} else {
			go bracketView()
			return
		}
	}()
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
