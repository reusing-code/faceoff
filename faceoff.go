package faceoff

type contender int

const (
	a    contender = 0
	b    contender = 1
	none contender = -1
)

type match struct {
	contenders [2]string
	score      [2]int
	winner     contender
}

func newMatch(contenderA string, contenderB string) *match {
	m := &match{contenders: [2]string{contenderA, contenderB}, score: [2]int{0, 0}, winner: none}
	return m
}

func (m *match) WinA() {
	m.score[a]++
}

func (m *match) WinB() {
	m.score[b]++
}
