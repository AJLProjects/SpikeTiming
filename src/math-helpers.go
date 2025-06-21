package main

import "math"

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
