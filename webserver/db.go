package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"math/rand"
	"os"
	pathpkg "path"
	"strconv"
	"strings"
	"time"

	"github.com/reusing-code/faceoff/shared/contest"

	bolt "github.com/coreos/bbolt"
)

var db *bolt.DB

const bucketName = "BracketBucket"

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

func OpenDB(path string) error {
	var err error
	dir, _ := pathpkg.Split(path)
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		return err
	}
	db, err = bolt.Open(path, 0644, nil)
	if err != nil {
		return err
	}
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		return nil
	})

	return err
}

func CloseDB() error {
	return db.Close()
}

func GetRoster(id string) (*contest.Roster, error) {
	value, err := GetValue(id)
	if value == nil || err != nil {
		return nil, fmt.Errorf("No key '%s' in DB", id)
	}

	result := &contest.Roster{}
	dec := gob.NewDecoder(bytes.NewReader(value))
	err = dec.Decode(result)
	return result, err
}

func GetValue(id string) ([]byte, error) {
	key := []byte(id)
	var value []byte
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		value = b.Get([]byte(key))
		return nil
	})
	return value, err
}

func SetRoster(id string, roster *contest.Roster) error {
	buf := &bytes.Buffer{}
	enc := gob.NewEncoder(buf)
	err := enc.Encode(roster)
	if err != nil {
		return err
	}
	value := buf.Bytes()
	key := []byte(id)

	err = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		err := b.Put(key, value)
		return err
	})
	return err
}

func CreateKey() string {
	var rnd int
	for rnd < 10000000 {
		rnd = rand.Intn(100000000)
	}
	id := strconv.Itoa(rnd)
	value, _ := GetValue(id)
	if value != nil {
		return CreateKey()
	}
	return id
}

func GetContestList() *contest.ContestList {
	list := &contest.ContestList{
		Open:   make([]contest.ContestDescription, 0),
		Closed: make([]contest.ContestDescription, 0),
	}

	db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		b.ForEach(func(k, v []byte) error {
			if k != nil && v != nil {
				key := string(k)
				if strings.HasSuffix(key, "_score") {
					return nil
				}
				r := &contest.Roster{}
				dec := gob.NewDecoder(bytes.NewReader(v))
				err := dec.Decode(r)
				if err != nil {
					return err
				}
				if r.Private {
					return nil
				}
				desc := contest.ContestDescription{
					Key:  key,
					Name: r.Name,
				}
				if r.ActiveRound < 0 {
					list.Closed = append(list.Closed, desc)
				} else {
					list.Open = append(list.Open, desc)
				}
			}
			return nil
		})
		return nil
	})
	return list
}
