package main

import (
	"bytes"
	"strconv"

	"github.com/gopherjs/gopherjs/js"
	"github.com/reusing-code/faceoff/shared/contest"
)

func getBracketOptions(r *contest.Roster) *js.Object {
	// REALLY REALLY ugly way to do this...
	teams := bytes.Buffer{}
	teams.WriteString("[")

	for _, currentMatch := range r.Rounds[0].Matches {
		teams.WriteString("[\"")
		teams.WriteString(currentMatch.Contenders[contest.A])
		teams.WriteString("\",\"")
		teams.WriteString(currentMatch.Contenders[contest.B])
		teams.WriteString("\"],")
	}

	teams.WriteString("]")

	results := bytes.Buffer{}
	results.WriteString("[")
	for _, currentRound := range r.Rounds {
		results.WriteString("[")
		for _, currentMatch := range currentRound.Matches {
			results.WriteString("[")
			results.WriteString(strconv.Itoa(currentMatch.Score[contest.A]))
			results.WriteString(",")
			results.WriteString(strconv.Itoa(currentMatch.Score[contest.B]))
			results.WriteString("],")
		}
		results.WriteString("],")
	}
	results.WriteString("]")

	dataStr := bytes.Buffer{}
	dataStr.WriteString("({\"teams\": ")
	dataStr.Write(teams.Bytes())
	dataStr.WriteString(",\"results\": ")
	dataStr.Write(results.Bytes())
	dataStr.WriteString("})")

	data := js.Global.Call("eval", dataStr.String()).Interface()

	obj := js.Global.Get("Object").New()
	obj.Set("skipConsolationRound", true)
	obj.Set("teamWidth", 150)
	obj.Set("init", data)
	return obj
}
