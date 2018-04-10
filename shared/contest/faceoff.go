package contest

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"math/rand"
	"strconv"
	"time"
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
	Num        int
}

type Round struct {
	Matches []*Match
}

type Contest struct {
	Name         string
	Rounds       []*Round
	CurrentVotes int
	ActiveRound  int
	Private      bool
	AdminKey     string
}

type ContestDescription struct {
	Key  string
	Name string
}

type ContestList struct {
	Open   []ContestDescription
	Closed []ContestDescription
}

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

func CreateRoster(name string, participants []string, private bool) (*Contest, error) {
	l := len(participants)
	// very ugly, but good enough for now
	if l != 2 && l != 4 && l != 8 && l != 16 && l != 32 {
		return nil, errors.New("Unsupported participant number")
	}

	res := &Contest{Rounds: make([]*Round, 0), Name: name}
	round := &Round{}

	round.Matches = generateMatches(participants)
	res.Rounds = append(res.Rounds, round)
	res.AdminKey = createAdminKey()
	res.Private = private
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

func (r *Contest) DeepCopy() *Contest {
	copy := &Contest{
		AdminKey:     r.AdminKey,
		Rounds:       make([]*Round, 0),
		CurrentVotes: r.CurrentVotes,
		ActiveRound:  r.ActiveRound,
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

func (r *Contest) AdvanceRound() {
	if r.ActiveRound < 0 {
		return
	}
	currentRound := r.Rounds[r.ActiveRound]
	if len(currentRound.Matches) < 1 {
		return
	}
	nextRound := &Round{}

	winners := make([]string, 0, len(currentRound.Matches)/2)
	for _, currentMatch := range currentRound.Matches {
		checkWinner(currentMatch)
		winners = append(winners, currentMatch.Contenders[currentMatch.Winner])
	}

	if len(winners) > 1 {
		nextRound.Matches = generateMatches(winners)
		r.Rounds = append(r.Rounds, nextRound)
		r.ActiveRound++

	} else {
		r.ActiveRound = -1
	}

	r.CurrentVotes = 0
}

func generateMatches(names []string) []*Match {
	l := len(names)
	if l%2 != 0 {
		panic("Number of names not divisible by 2")
	}
	res := make([]*Match, 0, l/2)
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
			m.Score[A]++
		} else {
			m.Winner = B
			m.Score[B]++
		}
	}
}

func ParseRoster(r io.ReadCloser) (*Contest, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	r.Close()
	result := &Contest{}
	err = json.Unmarshal(b, result)
	return result, err
}

func (r *Contest) AddVotes(vote *Contest) {
	currentRound := r.Rounds[len(r.Rounds)-1]
	voteRound := vote.Rounds[len(r.Rounds)-1]

	for i, voteMatch := range voteRound.Matches {
		match := currentRound.Matches[i]
		if voteMatch.Winner != NONE {
			match.Score[voteMatch.Winner]++
		}
	}
}

func (r *Contest) RemovePrivateData() {
	r.AdminKey = ""

	if r.ActiveRound >= 0 {
		currentRound := r.Rounds[r.ActiveRound]

		for i, voteMatch := range currentRound.Matches {
			match := currentRound.Matches[i]
			voteMatch.Winner = NONE
			match.Score = [2]int{0, 0}
		}
	}
}

var minKey = 10000000
var maxKey = 100000000

func createAdminKey() string {
	var rnd int
	for rnd < minKey {
		rnd = rand.Intn(maxKey)
	}
	id := strconv.Itoa(rnd)
	return "A" + id
}
