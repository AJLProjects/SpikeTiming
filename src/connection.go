package main

type Connection struct {
	From             *Neuron
	To               *Neuron
	IsExcitatory     bool
	Strength         float64
	IsAwaitingReward bool
	DebugLast        string

	LongTermStrength    float64
	LastUpdatedLongTerm float64
}

func (con *Connection) UpdateWeightsAndLTM(time float64) {

	var timePassed = time - con.LastUpdatedLongTerm

	var deltaStrength = con.LongTermStrength - con.Strength

	if timePassed > 1000 {
		con.Strength += deltaStrength - deltaStrength/1000
		con.LongTermStrength -= deltaStrength / 1000
	} else {

		//var ltmChange = (deltaStrength / 1000) * timePassed

	}

	con.LastUpdatedLongTerm = time
}
