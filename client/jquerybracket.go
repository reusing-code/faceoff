package main

import (
	"bytes"
	"strconv"

	"github.com/gopherjs/gopherjs/js"
	"github.com/reusing-code/faceoff"
)

func getBracketOptions(r *faceoff.Roster) *js.Object {
	// REALLY REALLY ugly way to do this...
	// data := js.Global.Get("Object").New()
	// data.Set("teams", []string{"Arsch", "Bratze", "Bongo", "Deppnase"})
	// data.Set("results", js.Global.Call("eval", "[ [[1,2], [3,4]],[[4,6], [2,1]] ]").Interface())
	teams := bytes.Buffer{}
	teams.WriteString("[")

	for _, currentMatch := range r.Rounds[0].Matches {
		teams.WriteString("[\"")
		teams.WriteString(currentMatch.Contenders[faceoff.A])
		teams.WriteString("\",\"")
		teams.WriteString(currentMatch.Contenders[faceoff.B])
		teams.WriteString("\"],")
	}

	teams.WriteString("]")

	results := bytes.Buffer{}
	results.WriteString("[")
	for _, currentRound := range r.Rounds {
		results.WriteString("[")
		for _, currentMatch := range currentRound.Matches {
			results.WriteString("[")
			results.WriteString(strconv.Itoa(currentMatch.Score[faceoff.A]))
			results.WriteString(",")
			results.WriteString(strconv.Itoa(currentMatch.Score[faceoff.B]))
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

	println(dataStr.String())

	data := js.Global.Call("eval", dataStr.String()).Interface()

	obj := js.Global.Get("Object").New()
	obj.Set("skipConsolationRound", true)
	obj.Set("teamWidth", 150)
	obj.Set("init", data)
	println(obj)
	return obj
}
