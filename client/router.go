package main

import (
	"github.com/gopherjs/gopherjs/js"
	"honnef.co/go/js/dom"
)

func route(path string, addToHistory bool) {
	if path == "" {
		path = dom.GetWindow().Location().Pathname
	}

	if path == "" || path == "/" {
		path = "/bracket"
	}

	if addToHistory {
		history := js.Global.Get("history")
		history.Call("pushState", nil, "", path)
	}

	if path == "/admin" {
		go adminView()
	} else if path == "/vote" {
		go votingView()
	} else if path == "/new" {
		go newBracketView()
	} else {
		go bracketView()
	}
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
