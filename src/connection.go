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

// Connections are packed small (40 bytes) and stored by value, each neuron's outgoing connections sit side by side in Brain.Connections
// so firing a neuron walks one contiguous block of memory rather than chasing a pointer per connection
// Strength and LongTermStrength stay float64 as they accumulate tiny changes, the fixed per connection values are float32, neurons are referenced by index into Brain.Neurons
// There is no IsExcitatory, all of a neuron's outgoing connections share the source neuron's type
type Connection struct {
	To               uint32 // Index into Brain.Neurons
	LastUpdateTime   uint32 // Simulation time only moves in whole cycles, so it fits in a uint32
	Strength         float64
	LongTermStrength float64 // Slowly follows the strength, and the strength is pulled back towards it
	BaseStrength     float32 // Set once at setup, so float32 loses nothing. Value the strength relaxes towards when not being rewarded or punished
	LogRelaxDecay    float32 // Log of the per cycle multiplier applied to the gap between the strength and the base, stored as a log so it can use math.Exp rather than math.Pow
	IsAwaitingReward bool
}

// IncomingConnection points back at a connection that lives in another neuron's outgoing block
type IncomingConnection struct {
	Conn uint32 // Index into Brain.Connections
	From uint32 // Index into Brain.Neurons of the sending neuron
}

// SetDistanceBase sets the base strength and relax speed from the distance between the two neurons
func (con *Connection) SetDistanceBase(dist float64) {
	con.BaseStrength = float32(BASE_STRENGTH_NEAR * math.Exp(-dist/BASE_STRENGTH_FALLOFF))
	con.LogRelaxDecay = float32(math.Log(1.0 - STRENGTH_RELAX_RATE*(1.0+STRENGTH_RELAX_DIST_MULT*dist/BASE_STRENGTH_FALLOFF)))
}

func (con *Connection) UpdateWeightsSince(time float64) {

	now := uint32(time)
	if now == con.LastUpdateTime {
		return
	}

	var cyclesSince = float64(now - con.LastUpdateTime)

	// Exponentially relax towards the base, above the base it decays down, below it recovers up
	base := float64(con.BaseStrength)
	strength := base + (con.Strength-base)*math.Exp(cyclesSince*float64(con.LogRelaxDecay))

	// Strength and long term strength lerp into each other, the long term one moves much slower
	longTerm := con.LongTermStrength
	con.Strength = Lerp(strength, longTerm, math.Min(FROM_LONG_TERM_RATE*cyclesSince, 1.0))
	con.LongTermStrength = Lerp(longTerm, strength, math.Min(TO_LONG_TERM_RATE*cyclesSince, 1.0))

	con.LastUpdateTime = now
}
