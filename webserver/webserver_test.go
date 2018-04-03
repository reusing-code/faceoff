package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/reusing-code/faceoff/shared/templates"
)

func fixWorkingDir() {
	pwd, _ := os.Getwd()
	if path.Base(pwd) == "webserver" {
		os.Chdir("..")
	}
}

func unfixWorkingDir() {
	pwd, _ := os.Getwd()
	if path.Base(pwd) != "webserver" {
		os.Chdir("webserver")
	}
}

var expectedIndexContent = []struct {
	token string
}{
	{"<!doctype html>"},
	{"</html>"},
	{"client.js"},
}

func TestIndexHandler(t *testing.T) {
	fixWorkingDir()
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(indexHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
	body := rr.Body.String()
	for _, exp := range expectedIndexContent {
		if !strings.Contains(body, exp.token) {
			t.Errorf("handler returned unexpected body: want %v", exp.token)
		}
	}
}

func TestTemplateHandler(t *testing.T) {
	unfixWorkingDir()
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(templateHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusInternalServerError)
	}

	fixWorkingDir()

	rr = httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
	ts, err := templates.LoadTemplatesFromGob(rr.Body.Bytes())
	if err != nil {
		t.Fatal(err)
	}

	if len(ts.Templates) < 5 {
		t.Errorf("not enough templates : got %v want >5",
			len(ts.Templates))
	}
}

func TestRouter(t *testing.T) {
	fixWorkingDir()
	// this is not really testing a lot just if the router is functional
	router := CreateRouter()
	req, err := http.NewRequest("GET", "/THIS/IS/JUST/A/TEST", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
}
