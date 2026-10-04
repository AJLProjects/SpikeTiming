package main

import "math"

// Distance based baseline: connections relax towards a base strength that falls off with distance
// Being rewarded can push a connection above its base, but far connections have a lower base and are pulled back faster
//   dist 1  -> base ~0.18, relax 0.99888 per cycle
//   dist 8  -> base ~0.07, relax 0.998 per cycle
//   dist 17 -> base ~0.02, relax 0.99688 per cycle
const BASE_STRENGTH_NEAR = 0.2       // Base strength at distance 0
const BASE_STRENGTH_FALLOFF = 8.0    // Base strength drops by ~63% every this many squares
const STRENGTH_RELAX_RATE = 0.001    // Fraction of the gap to the base closed per cycle at distance 0
const STRENGTH_RELAX_DIST_MULT = 1.0 // How much faster far connections relax, relax rate is multiplied by 1 + dist/FALLOFF * this

const TO_LONG_TERM_RATE = 0.0001 // Per cycle lerp of the long term strength towards the strength (slow)
const FROM_LONG_TERM_RATE = 0.01 // Per cycle lerp of the strength towards the long term strength (fast)

type Connection struct {
	From             *Neuron
	To               *Neuron
	IsExcitatory     bool
	Strength         float64
	LongTermStrength float64 // Slowly follows the strength, and the strength is pulled back towards it
	BaseStrength     float64 // Value the strength relaxes towards when not being rewarded or punished
	RelaxDecay       float64 // Per cycle multiplier applied to the gap between the strength and the base
	IsAwaitingReward bool
	DebugLast        string

	LastUpdateTime float64
}

// SetDistanceBase sets the base strength and relax speed from the distance between the two neurons
func (con *Connection) SetDistanceBase(dist float64) {
	con.BaseStrength = BASE_STRENGTH_NEAR * math.Exp(-dist/BASE_STRENGTH_FALLOFF)
	con.RelaxDecay = 1.0 - STRENGTH_RELAX_RATE*(1.0+STRENGTH_RELAX_DIST_MULT*dist/BASE_STRENGTH_FALLOFF)
}

func (con *Connection) UpdateWeightsSince(time float64) {

	if time == con.LastUpdateTime {
		return
	}

	var cyclesSince = time - con.LastUpdateTime

	// Exponentially relax towards the base, above the base it decays down, below it recovers up
	con.Strength = con.BaseStrength + (con.Strength-con.BaseStrength)*math.Pow(con.RelaxDecay, cyclesSince)

	// Strength and long term strength lerp into each other, the long term one moves much slower
	strength, longTerm := con.Strength, con.LongTermStrength
	con.Strength = Lerp(strength, longTerm, math.Min(FROM_LONG_TERM_RATE*cyclesSince, 1.0))
	con.LongTermStrength = Lerp(longTerm, strength, math.Min(TO_LONG_TERM_RATE*cyclesSince, 1.0))

	con.LastUpdateTime = time
}
