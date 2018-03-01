package faceoff

import (
	"bufio"
	"bytes"
	"errors"
	"strings"
)

type contender int

const (
	a    contender = 0
	b    contender = 1
	none contender = -1
)

type Match struct {
	Contenders [2]string
	Score      [2]int
	Winner     contender
}

func CreateRoster(participants []byte) (*Roster, error) {
	buf := bufio.NewScanner(bytes.NewReader(participants))
	partSlice := make([]string, 0, 16)
	for buf.Scan() {
		name := strings.TrimSpace(buf.Text())
		if len(name) > 1 {
			partSlice = append(partSlice, name)
		}
	}
	l := len(partSlice)
	// very ugly, but good enough for now
	if l != 2 && l != 4 && l != 8 && l != 16 && l != 32 {
		return nil, errors.New("Unsupported participant number")
	}

	res := &Roster{Rounds: make([]*Round, 0)}
	round := &Round{Matches: make([]*Match, 0)}

	for i := 0; i < l; i++ {
		m := NewMatch(partSlice[i], partSlice[i+1])
		i++
		round.Matches = append(round.Matches, m)
	}
	res.Rounds = append(res.Rounds, round)

	return res, nil

}

func NewMatch(contenderA string, contenderB string) *Match {
	m := &Match{Contenders: [2]string{contenderA, contenderB}, Score: [2]int{0, 0}, Winner: none}
	return m
}

func (m *Match) WinA() {
	m.Score[a]++
}

func (m *Match) WinB() {
	m.Score[b]++
}

type Round struct {
	Matches []*Match
}

type Roster struct {
	Rounds []*Round
}
