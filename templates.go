package faceoff

import (
	"bytes"
	"encoding/gob"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type TemplateSet struct {
	Templates map[string]string
}

func LoadTemplatesFromDisk() (*TemplateSet, error) {
	ts := &TemplateSet{}
	ts.Templates = make(map[string]string)
	err := filepath.Walk("templates", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		name := strings.TrimPrefix(path, "templates/")
		name = strings.Split(name, ".")[0]
		b, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		b = configureRawTemplateFile(b)
		ts.Templates[name] = string(b)
		return nil
	})

	if err != nil {
		return ts, err
	}
	return ts, nil
}

func (ts *TemplateSet) EncodeGob() ([]byte, error) {
	buf := &bytes.Buffer{}
	enc := gob.NewEncoder(buf)
	err := enc.Encode(ts)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func LoadTemplatesFromGob(b []byte) (*TemplateSet, error) {
	ts := &TemplateSet{}
	dec := gob.NewDecoder(bytes.NewReader(b))
	err := dec.Decode(ts)
	return ts, err
}

func configureRawTemplateFile(b []byte) []byte {
	version, err := ioutil.ReadFile("version.txt")
	if err != nil {
		version = []byte("")
	}

	result := strings.Replace(string(b), "@version@", string(version), -1)
	return []byte(result)
}
