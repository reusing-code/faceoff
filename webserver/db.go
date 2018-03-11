package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/reusing-code/faceoff"

	bolt "github.com/coreos/bbolt"
)

var db *bolt.DB

const bucketName = "BracketBucket"

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

func OpenDB() error {
	var err error
	db, err = bolt.Open("faceoff.db", 0644, nil)
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

func GetRoster(id string) (*faceoff.Roster, error) {
	value, err := GetValue(id)
	if value == nil || err != nil {
		return nil, fmt.Errorf("No key '%s' in DB", id)
	}

	result := &faceoff.Roster{}
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

func SetRoster(id string, roster *faceoff.Roster) error {
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
