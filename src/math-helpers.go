package main

import (
	"math"
	"math/rand"
)

func PunishValue(val float64, fraction float64) float64 {

	/*if fraction < 0 { // REMOVE ON FINAL
		panic("Expcted pos")
	}*/

	return math.Max(val-val*fraction, 0.00001)
}

func RewardValue(val float64, fraction float64, max float64) float64 {

	/*if fraction < 0 { // REMOVE ON FINAL
		panic("Expcted pos")
	}*/

	var remaining = max - val
	return val + remaining*fraction
}

func PosOnlyRandMultiplier(size float64) float64 {
	return 1.0 + (rand.ExpFloat64() * size)
}

func Lerp(from float64, to float64, timestep float64) float64 {
	return from + (to-from)*timestep
}

// math.Pow is slow and simulation time only moves in whole cycles, so decays like rate^cycles are looked up from a table instead
// Anything outside the table (fractional or very long gaps) falls back to exp(cycles * log(rate)), which is still faster than math.Pow
const POW_TABLE_SIZE = 4096

type PowTable struct {
	base    float64
	logBase float64
	values  [POW_TABLE_SIZE]float64
}

func NewPowTable(base float64) *PowTable {
	t := &PowTable{}
	t.build(base)
	return t
}

func (t *PowTable) build(base float64) {
	t.base = base
	t.logBase = math.Log(base)
	value := 1.0
	for i := range t.values {
		t.values[i] = value
		value *= base
	}
}

// Pow returns base^exponent, the base is passed in each time so a slider changing it rebuilds the table automatically
func (t *PowTable) Pow(base float64, exponent float64) float64 {
	if base != t.base {
		t.build(base)
	}
	i := int(exponent)
	if i >= 0 && i < POW_TABLE_SIZE && float64(i) == exponent {
		return t.values[i]
	}
	return math.Exp(exponent * t.logBase)
}

// Cosine fall from 1 at x=0 to 0 at x=1, 0.5 * (1 + cos(pi*x)), read from a table with linear interpolation instead of calling math.Cos
const COSINE_FALL_TABLE_SIZE = 256

var cosineFallTable = func() [COSINE_FALL_TABLE_SIZE + 1]float64 {
	var table [COSINE_FALL_TABLE_SIZE + 1]float64
	for i := range table {
		table[i] = 0.5 * (1.0 + math.Cos(math.Pi*float64(i)/COSINE_FALL_TABLE_SIZE))
	}
	return table
}()

func CosineFall(x float64) float64 {
	if x <= 0 {
		return 1.0
	}
	if x >= 1 {
		return 0.0
	}
	f := x * COSINE_FALL_TABLE_SIZE
	i := int(f)
	return Lerp(cosineFallTable[i], cosineFallTable[i+1], f-float64(i))
}
