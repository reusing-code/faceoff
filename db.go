package faceoff

import (
	"bytes"
	"encoding/gob"
	"fmt"

	bolt "github.com/coreos/bbolt"
)

var db *bolt.DB

const bucketName = "BracketBucket"

func OpenDB() error {
	var err error
	db, err = bolt.Open("my.db", 0644, nil)
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

func GetRoster(id string) (*Roster, error) {
	key := []byte(id)
	var value []byte
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		value = b.Get([]byte(key))
		return nil
	})
	if value == nil || err != nil {
		return nil, fmt.Errorf("No key '%s' in DB", id)
	}

	result := &Roster{}
	dec := gob.NewDecoder(bytes.NewReader(value))
	err = dec.Decode(result)
	return result, err
}

func SetRoster(id string, roster *Roster) error {
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
