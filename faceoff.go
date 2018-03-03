package faceoff

import (
	"bufio"
	"bytes"
	"errors"
	"math/rand"
	"strings"

	"github.com/google/uuid"
)

type Contender int

const (
	A    Contender = 0
	B    Contender = 1
	NONE Contender = -1
)

type Match struct {
	Contenders [2]string
	Score      [2]int
	Winner     Contender
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
	round := &Round{}

	round.Matches = generateMatches(partSlice)
	res.Rounds = append(res.Rounds, round)
	id, _ := uuid.New().MarshalBinary()
	res.UUID = id
	return res, nil

}

func NewMatch(contenderA string, contenderB string) *Match {
	m := &Match{Contenders: [2]string{contenderA, contenderB}, Score: [2]int{0, 0}, Winner: NONE}
	return m
}

func (m *Match) WinA() {
	m.Score[A]++
	m.checkWinner()
}

func (m *Match) WinB() {
	m.Score[B]++
	m.checkWinner()
}

func (m *Match) checkWinner() {
	if m.Score[A] > m.Score[B] {
		m.Winner = A
	} else if m.Score[A] < m.Score[B] {
		m.Winner = B
	} else {
		m.Winner = NONE
	}
}

type Round struct {
	Matches []*Match
}

type Roster struct {
	UUID   []byte
	Rounds []*Round
}

func (r *Roster) DeepCopy() *Roster {
	copy := &Roster{
		UUID:   r.UUID,
		Rounds: make([]*Round, 0),
	}

	for _, orgRound := range r.Rounds {
		copyRound := &Round{Matches: make([]*Match, 0)}

		for _, orgMatch := range orgRound.Matches {
			copyMatch := NewMatch(orgMatch.Contenders[A], orgMatch.Contenders[B])
			copyMatch.Score[A] = orgMatch.Score[A]
			copyMatch.Score[B] = orgMatch.Score[B]
			copyMatch.Winner = orgMatch.Winner
			copyRound.Matches = append(copyRound.Matches, copyMatch)
		}
		copy.Rounds = append(copy.Rounds, copyRound)
	}

	return copy
}

func (r *Roster) AdvanceRound() {
	currentRound := r.Rounds[len(r.Rounds)-1]
	if len(currentRound.Matches) < 2 {
		return
	}
	nextRound := &Round{}

	winners := make([]string, len(currentRound.Matches)/2)
	for _, currentMatch := range currentRound.Matches {
		checkWinner(currentMatch)
		winners = append(winners, currentMatch.Contenders[currentMatch.Winner])
	}

	nextRound.Matches = generateMatches(winners)
	r.Rounds = append(r.Rounds, nextRound)

	id, _ := uuid.New().MarshalBinary()
	r.UUID = id
}

func generateMatches(names []string) []*Match {
	l := len(names)
	if l%2 != 0 {
		panic("Number of names not divisible by 2")
	}
	res := make([]*Match, l/2)
	for i := 0; i < l; i++ {
		m := NewMatch(names[i], names[i+1])
		i++
		res = append(res, m)
	}
	return res
}

func checkWinner(m *Match) {
	if m.Score[A] > m.Score[B] {
		m.Winner = A
	} else if m.Score[B] > m.Score[A] {
		m.Winner = B
	} else {
		if rand.Intn(2) == 0 {
			m.Winner = A
		} else {
			m.Winner = B
		}
	}
}
