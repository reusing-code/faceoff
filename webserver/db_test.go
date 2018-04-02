package main

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	"github.com/reusing-code/faceoff/shared/contest"
)

const DB_TEST_FILE = "test_db.db"

func setupDB(t *testing.T) {
	err := OpenDB(DB_TEST_FILE)
	if err != nil {
		t.Fatal(err)
	}
}

func tearDownDB(t *testing.T) {
	err := CloseDB()
	if err != nil {
		t.Error(err)
	}
	err = os.Remove(DB_TEST_FILE)
	if err != nil {
		t.Error(err)
	}
}

var openCloseTests = []struct {
	dbFilePath string
	validPath  bool
}{
	{"test_db.db", true},
	{"test_db/test_db.db", true},
	{"test_db/db/db/db/db/test_db.db", true},
	{"", false},
	{"test_db/", false},
	{"test_db///db/db/test_db2.db", true},
	{"invalid_file.db", false},
}

func TestOpenClose(t *testing.T) {
	defer os.RemoveAll("test_db/")
	defer os.Remove("test_db.db")
	ioutil.WriteFile("invalid_file.db", []byte("THIS IS NOT A VALID DB FILE"), 0644)
	defer os.Remove("invalid_file.db")

	for _, tt := range openCloseTests {
		err := OpenDB(tt.dbFilePath)
		// good cases: validPath && no error, !validPath && error
		// bad cases: validPath && error, !validPath && no error
		if err != nil && tt.validPath {
			t.Errorf("OpenDB(%q) => error: %q", tt.dbFilePath, err)
		}
		if err == nil && !tt.validPath {
			t.Errorf("OpenDB(%q) => no error", tt.dbFilePath)
		}
		CloseDB()
	}
}

func TestValue(t *testing.T) {
	setupDB(t)
	defer tearDownDB(t)

	v := []byte("testvalue. . . . ABC")
	key := "testkey"

	result, err := GetValue(key)
	if err != nil {
		t.Errorf("GetValue(%q) Error: %q", key, err)
	}
	if result != nil {
		t.Errorf("GetValue(%q) => %q, want nil", key, result)
	}

	err = SetValue(key, v)
	if err != nil {
		t.Fatalf("SetValue(%q, %q) Error: %q", key, string(v), err)
	}

	result, err = GetValue(key)
	if err != nil {
		t.Errorf("GetValue(%q) Error: %q", key, err)
	}
	if bytes.Compare(v, result) != 0 {
		t.Errorf("GetValue(%q) => %q, want %q", key, result, v)
	}

	err = SetValue("", v)
	if err == nil {
		t.Errorf("SetValue(%q, %q), want error", "", string(v))
	}

	result, err = GetValue("")
	if err != nil {
		t.Errorf("GetValue(%q) Error: %q", "", err)
	}
	if result != nil {
		t.Errorf("GetValue(%q) => %q, want nil", "", result)
	}

	err = SetValue("", nil)
	if err == nil {
		t.Errorf("SetValue(%q, nil), want error", "")
	}
}

var rosterTestData = []struct {
	key          string
	name         string
	participants []string
	private      bool
	closed       bool
}{
	{"id1", "Test Roster 1", []string{"A1", "A2", "A3", "A4"}, false, false},
	{"id2", "Test Roster 2", []string{"B1", "B2", "B3", "B4"}, false, false},
	{"id3", "Private Roster 1", []string{"C1", "C2"}, true, false},
	{"id4", "Private Roster 2", []string{"C1", "C2"}, true, false},
	{"id5", "Closed Test Roster 1", []string{"A1", "A2", "A3", "A4"}, false, true},
	{"id6", "Closed Private Test Roster 2", []string{"B1", "B2", "B3", "B4"}, true, false},
}

func TestRoster(t *testing.T) {
	setupDB(t)
	defer tearDownDB(t)

	for _, tt := range rosterTestData {
		r, err := contest.CreateRoster(tt.name, tt.participants, tt.private)
		if err != nil {
			t.Errorf("contest.CreateRoster(%q, ...) => error: %q", tt.name, err)
			continue
		}
		if tt.closed {
			r.ActiveRound = -1
		}
		err = SetRoster(tt.key, r)
		if err != nil {
			t.Errorf("SetRoster(%q, ...) => error: %q", tt.key, err)
			continue
		}
		err = SetRoster(GetScoreKey(tt.key), r)
		if err != nil {
			t.Errorf("SetRoster(%q, ...) => error: %q", GetScoreKey(tt.key), err)
			continue
		}
	}

	list := GetContestList()
	if len(list.Closed) != 1 {
		t.Errorf("len(list.Closed) => '%d', want '%d'", len(list.Closed), 1)
	}
	if len(list.Open) != 2 {
		t.Errorf("len(list.Open) => '%d', want '%d'", len(list.Open), 2)
	}

	err := SetRoster("", &contest.Roster{})
	if err == nil {
		t.Errorf("SetRoster(\"\", &contest.Roster{}) => no error, want error")
	}

	err = SetRoster("testkey", nil)
	if err == nil {
		t.Errorf("SetRoster(\"testkey\", nil) => no error, want error")
	}

	for _, tt := range rosterTestData {
		for _, key := range []string{tt.key, GetScoreKey(tt.key)} {
			r, err := GetRoster(key)
			if err != nil {
				t.Errorf("GetRoster(%q) => error: %q", key, err)
			}
			if r.Name != tt.name {
				t.Errorf("GetRoster(%q).name => %q, want %q", key, r.Name, tt.name)
			}
		}
	}
}
