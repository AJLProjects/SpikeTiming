package main

import "math"

type FractionDecayValue struct {
	Value                float64
	DecayPerTimestep     float64
	LastCycleProgression float64
}

func (v *FractionDecayValue) GetAndUpdate(pregression float64) float64 {
	var cyclesSince = pregression - v.LastCycleProgression

	var newValue = v.Value * math.Pow(v.DecayPerTimestep, cyclesSince)

	v.Value = newValue
	v.LastCycleProgression = pregression

	return newValue
}
