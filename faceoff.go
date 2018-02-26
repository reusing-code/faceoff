package faceoff

type contender int

const (
	a    contender = 0
	b    contender = 1
	none contender = -1
)

type Match struct {
	contenders [2]string
	score      [2]int
	winner     contender
}

func NewMatch(contenderA string, contenderB string) *Match {
	m := &Match{contenders: [2]string{contenderA, contenderB}, score: [2]int{0, 0}, winner: none}
	return m
}

func (m *Match) WinA() {
	m.score[a]++
}

func (m *Match) WinB() {
	m.score[b]++
}
