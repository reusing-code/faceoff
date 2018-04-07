package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/reusing-code/faceoff/shared/contest"

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

func TestRosterHandler(t *testing.T) {
	setupDB(t)
	defer tearDownDB(t)

	roster, _ := contest.CreateRoster("TestRoster", []string{"A", "TestNameB", "C", "D"}, false)
	SetRoster("123", roster)

	err := SetValue("456", []byte("notarosternotaroster"))
	if err != nil {
		t.Fatal(err)
	}

	r := mux.NewRouter()
	r.HandleFunc("/xhr/{key:[0-9]+}", http.HandlerFunc(rosterHandler))

	ts := httptest.NewServer(r)
	defer ts.Close()

	tt := []struct {
		key        string
		statuscode int
	}{
		{"555", http.StatusNotFound},
		{"123", http.StatusOK},
		{"456", http.StatusNotFound},
	}

	for _, tc := range tt {
		url := ts.URL + "/xhr/" + tc.key
		resp, err := http.Get(url)
		if err != nil {
			t.Fatal(err)
		}

		if status := resp.StatusCode; status != tc.statuscode {
			t.Errorf("wrong status code: got %d want %d", status, tc.statuscode)
			continue
		}
		if tc.statuscode == http.StatusOK {
			buf := &bytes.Buffer{}
			buf.ReadFrom(resp.Body)
			resp.Body.Close()
			body := buf.String()
			if expString := "TestNameB"; !strings.Contains(body, expString) {
				t.Errorf("Response body does not contain %q", expString)
			}
		}
	}

}

func TestNewRosterHandler(t *testing.T) {
	setupDB(t)
	defer tearDownDB(t)
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(newRosterHandler)

	handler.ServeHTTP(rr, req)

	if status, expetedStatus := rr.Code, http.StatusBadRequest; status != expetedStatus {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, expetedStatus)
	}
	buf := bytes.NewBufferString("notaroster")
	req, err = http.NewRequest("POST", "/", buf)
	if err != nil {
		t.Fatal(err)
	}
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if status, expetedStatus := rr.Code, http.StatusBadRequest; status != expetedStatus {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, expetedStatus)
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
