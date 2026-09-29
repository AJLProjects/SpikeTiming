package main

import "math"

type Connection struct {
	From             *Neuron
	To               *Neuron
	IsExcitatory     bool
	Strength         float64
	IsAwaitingReward bool
	DebugLast        string

	LastUpdateTime float64
}

func (con *Connection) UpdateWeightsSince(time float64) {

	if time == con.LastUpdateTime {
		return
	}

	var cyclesSince = time - con.LastUpdateTime

	con.Strength *= math.Pow(0.999, cyclesSince)

	con.LastUpdateTime = time
}
