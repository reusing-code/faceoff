package faceoff

import (
	"bytes"
	"encoding/gob"
	"fmt"
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
	os.Chdir("..")
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
		ts.Templates[name] = string(b)
		fmt.Println(name)
		return nil
	})

	if err != nil {
		return ts, err
	}
	os.Chdir("webserver")
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
