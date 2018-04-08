package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"reflect"
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

	roster, _ := contest.CreateRoster("TestRoster", []string{"A", "TestNameB", "C", "D"}, false)
	b, _ := json.Marshal(roster)
	req, err = http.NewRequest("POST", "/", bytes.NewBuffer(b))
	if err != nil {
		t.Fatal(err)
	}
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if status, expetedStatus := rr.Code, http.StatusOK; status != expetedStatus {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, expetedStatus)
	}
	newKey := rr.Body.String()
	if len(newKey) < 3 || len(newKey) > 15 {
		t.Errorf("Unexpected new key %q", newKey)
	}
}

func TestRosterListHandler(t *testing.T) {
	setupDB(t)
	defer tearDownDB(t)

	roster, _ := contest.CreateRoster("TestRoster", []string{"A", "TestNameB", "C", "D"}, false)
	SetRoster("123", roster)

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(rosterListHandler)

	handler.ServeHTTP(rr, req)

	expecetdList := GetContestList()
	var receivedList contest.ContestList

	err = json.Unmarshal(rr.Body.Bytes(), &receivedList)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(expecetdList, &receivedList) {
		t.Errorf("received list not equal to expected list")
	}
}

func TestVoteHandler(t *testing.T) {
	setupDB(t)
	defer tearDownDB(t)

	roster, _ := contest.CreateRoster("TestRoster", []string{"A", "TestNameB", "C", "D"}, false)
	SetRoster("123", roster)
	SetRoster(GetScoreKey("123"), roster)

	SetRoster("789", roster) // no score roster

	roster.Rounds[0].Matches[0].WinB()
	roster.Rounds[0].Matches[1].WinA()

	voteJson, _ := json.Marshal(roster)

	r := mux.NewRouter()
	r.HandleFunc("/xhr/{key:[0-9]+}", http.HandlerFunc(voteHandler))

	ts := httptest.NewServer(r)
	defer ts.Close()

	tt := []struct {
		key        string
		data       []byte
		statuscode int
	}{
		{"123", voteJson, http.StatusOK},
		{"123", []byte("foobar"), http.StatusBadRequest},
		{"456", voteJson, http.StatusNotFound},
		{"789", voteJson, http.StatusNotFound},
	}

	for _, tc := range tt {
		url := ts.URL + "/xhr/" + tc.key
		resp, err := http.Post(url, "application/json", bytes.NewBuffer(tc.data))
		if err != nil {
			t.Fatal(err)
		}

		if status := resp.StatusCode; status != tc.statuscode {
			t.Errorf("wrong status code: got %d want %d", status, tc.statuscode)
			continue
		}
	}
}

func TestAdvanceRoundHandler(t *testing.T) {
	setupDB(t)
	defer tearDownDB(t)

	roster, _ := contest.CreateRoster("TestRoster", []string{"A", "TestNameB", "C", "D"}, false)
	roster.Rounds[0].Matches[0].WinB()
	roster.Rounds[0].Matches[1].WinA()
	SetRoster("123", roster)
	SetRoster(GetScoreKey("123"), roster)

	r := mux.NewRouter()
	r.HandleFunc("/xhr/{key:[0-9]+}", http.HandlerFunc(roundAdvanceHandler))

	ts := httptest.NewServer(r)
	defer ts.Close()

	tt := []struct {
		key        string
		data       []byte
		statuscode int
	}{
		{"123", roster.UUID, http.StatusOK},
		{"123", []byte("foobar"), http.StatusBadRequest},
		{"456", roster.UUID, http.StatusNotFound},
	}

	for _, tc := range tt {
		url := ts.URL + "/xhr/" + tc.key
		resp, err := http.Post(url, "application/json", bytes.NewBuffer(tc.data))
		if err != nil {
			t.Fatal(err)
		}

		if status := resp.StatusCode; status != tc.statuscode {
			t.Errorf("wrong status code: got %d want %d", status, tc.statuscode)
			continue
		}
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
