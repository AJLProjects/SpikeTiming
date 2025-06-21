package main

import (
	"fmt"
	"math"
	"math/rand"
	"sync"
)

// Grid holds the 2D array of neurons.
type Brain struct {
	Width         int
	Height        int
	BrainSquares  [][]BrainSquare
	ThreadStrips  []*MultithreadStrip
	NumberThreads int
}

type BrainSquare struct {
	PosX            int
	PosY            int
	NumberOfNeurons int
	Neurons         []Neuron
}

// NewGrid creates a new grid of neurons.
func NewBrain(width int, height int, cores int) *Brain {
	brainSquares := make([][]BrainSquare, width)
	for x := 0; x < width; x++ {
		brainSquares[x] = make([]BrainSquare, height)
		for y := 0; y < height; y++ {
			brainSquares[x][y] = BrainSquare{
				PosX: int(x),
				PosY: int(y),
			}
		}
	}
	return &Brain{
		Width:         width,
		Height:        height,
		BrainSquares:  brainSquares,
		NumberThreads: cores,
	}
}

func (brain *Brain) AddStandardNeurons() {
	for x := 0; x < brain.Width; x++ {
		for y := 0; y < brain.Height; y++ {
			var cell = &brain.BrainSquares[x][y]
			cell.Neurons = append(cell.Neurons, NewNeuron(x, y, cell, true))
			cell.NumberOfNeurons = len(cell.Neurons)
		}
	}
}

func (brain *Brain) AddNegativeNeurons() {
	for x := 1; x < brain.Width; x += 2 {
		for y := 1; y < brain.Height; y += 2 {
			var cell = &brain.BrainSquares[x][y]
			cell.Neurons = append(cell.Neurons, NewNeuron(x, y, cell, false))
			cell.NumberOfNeurons = len(cell.Neurons)
		}
	}
}

// ConnectNeighbors connects each neuron to its immediate neighbors (Moore neighborhood).
// This creates bidirectional connections.
func (brain *Brain) ConnectNeurons() {

	var updateIncrement = brain.Height / 20

	for y := 0; y < brain.Height; y++ {

		if y%updateIncrement == 0 {
			var percentageConnected = math.Round((float64(y) / float64(brain.Height)) * 100.0)
			fmt.Println("Connected: ", percentageConnected, "%")
		}

		for x := 0; x < brain.Width; x++ {
			currentSquare := &brain.BrainSquares[x][y]
			for s := 0; s < currentSquare.NumberOfNeurons; s++ {
				currentSquare.Neurons[s].InitialConnect(brain)
			}
		}
	}
}

func (brain *Brain) WrapPosition(x int, y int) (outX int, outY int) {

	if x < 0 {
		x = brain.Width + x
	} else if x >= brain.Width {
		x = x - brain.Width
	}

	if y < 0 {
		y = brain.Height + y
	} else if y >= brain.Height {
		y = y - brain.Height
	}

	return x, y
}

func (brain *Brain) Update(time float64) {

	//var num = rand.Int32()
	var pickedNeuron = brain.BrainSquares[rand.Intn(brain.Width)][rand.Intn(brain.Height)].Neurons[0]

	//fmt.Println("Threshold:", pickedNeuron.Threshold)
	pickedNeuron.AddActivation(3, time)
	//fmt.Println("Activation:", pickedNeuron.Activation)

	// The three lanes per thread
	for l := 0; l < 3; l += 1 {

		var wg sync.WaitGroup

		for i := 0; i < brain.NumberThreads; i += 1 {

			wg.Add(1)

			var stripUpdating = brain.ThreadStrips[(i*3)+l]
			go func(_stripUpdating *MultithreadStrip) {
				defer wg.Done()

				// TODO: Might just be better to keep it sdtatic we just invert where we write to?
				// Do a swap of the updates
				var temp = _stripUpdating.NeuronsToCheck
				_stripUpdating.NeuronsToCheck = _stripUpdating.NextNeuronsToCheck
				_stripUpdating.NextNeuronsToCheck = temp[:0] // Temp but cleared

				for _, neuron := range _stripUpdating.NeuronsToCheck {
					neuron.AlreadyAddedToTocheckList = false
					neuron.CheckForFiring(time)
				}

			}(stripUpdating)
		}

		wg.Wait()

	}

}
