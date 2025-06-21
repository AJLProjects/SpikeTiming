package main

type Connection struct {
	From                *Neuron
	To                  *Neuron
	IsExcitatory        bool
	Strength            float64
	IsAwaitingReward    bool
	MagnitudeMultiplier float64 // this multiplied with the Strength(0 to 1) is the final value
	DebugLast           string
}
