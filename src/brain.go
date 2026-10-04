package main

import (
	"fmt"
	"math"
	"math/rand"
)

// Grid holds the 2D array of neurons.
type Brain struct {
	Width        int
	Height       int
	BrainSquares [][]BrainSquare

	// Every neuron lives here, each BrainSquare.Neurons is a subslice of it, so it must never be appended to after AddNeurons
	Neurons []Neuron
	// Every connection lives here by value, each neuron's outgoing connections are one contiguous block of it
	Connections []Connection

	// Neurons to check
	NeuronsToCheck     []*Neuron
	NextNeuronsToCheck []*Neuron
}

type BrainSquare struct {
	PosX            int
	PosY            int
	NumberOfNeurons int
	Neurons         []Neuron
	isWall          bool
	DirVecX         float64
	DirVecY         float64
}

// NewGrid creates a new grid of neurons.
func NewBrain(width int, height int, brainPattern []byte) *Brain {
	brainSquares := make([][]BrainSquare, width)
	for x := 0; x < width; x++ {
		brainSquares[x] = make([]BrainSquare, height)
		for y := 0; y < height; y++ {

			var patternPos = (x*width + y) * 4

			// Its wall if its black
			isWall := (brainPattern[patternPos+0] == 0 && brainPattern[patternPos+1] == 0 && brainPattern[patternPos+2] == 0)

			// Convert to float before subtracting, otherwise the byte subtraction wraps around for values below 127
			var xAmount = (float64(brainPattern[patternPos+0]) - 127.0) / 127.0
			var yAmount = (float64(brainPattern[patternPos+1]) - 127.0) / 127.0

			var m = math.Sqrt(xAmount*xAmount + yAmount*yAmount)

			//fmt.Printf("Dirvec: %v, %v \n", xAmount/m, 1.0/m)

			// No direction (exactly mid grey) leaves the dir vec at zero rather than dividing by zero
			var dirVecX, dirVecY float64
			if m > 0 {
				dirVecX = yAmount / m // Flipped because the pattern is read transposed, BrainSquares[x][y] is image row x, column y
				dirVecY = xAmount / m
			}

			brainSquares[x][y] = BrainSquare{
				PosX:    int(x),
				PosY:    int(y),
				isWall:  isWall,
				DirVecX: dirVecX,
				DirVecY: dirVecY,
			}
		}
	}
	return &Brain{
		Width:              width,
		Height:             height,
		BrainSquares:       brainSquares,
		NeuronsToCheck:     make([]*Neuron, 0),
		NextNeuronsToCheck: make([]*Neuron, 0),
	}
}

// Every non wall square gets a standard (excitatory) neuron, and every square on odd x and odd y also gets a negative (inhibitory) one
func hasNegativeNeuron(x int, y int) bool {
	return x%2 == 1 && y%2 == 1
}

// AddNeurons puts every neuron into the single Brain.Neurons slice, in square order, then points each square's Neurons at its part of it
// Connections refer to neurons by index into Brain.Neurons, so it is sized exactly up front and never reallocated
func (brain *Brain) AddNeurons() {

	total := 0
	for x := 0; x < brain.Width; x++ {
		for y := 0; y < brain.Height; y++ {
			if !brain.BrainSquares[x][y].isWall {
				total++
				if hasNegativeNeuron(x, y) {
					total++
				}
			}
		}
	}

	brain.Neurons = make([]Neuron, 0, total)

	for x := 0; x < brain.Width; x++ {
		for y := 0; y < brain.Height; y++ {
			var cell = &brain.BrainSquares[x][y]
			if cell.isWall {
				continue
			}

			start := len(brain.Neurons)
			brain.Neurons = append(brain.Neurons, NewNeuron(brain, cell, uint32(len(brain.Neurons)), true))
			if hasNegativeNeuron(x, y) {
				brain.Neurons = append(brain.Neurons, NewNeuron(brain, cell, uint32(len(brain.Neurons)), false))
			}

			cell.Neurons = brain.Neurons[start:len(brain.Neurons):len(brain.Neurons)] // Capped so an append to a square can't overwrite the next square
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
			if !currentSquare.isWall {
				for s := 0; s < currentSquare.NumberOfNeurons; s++ {
					currentSquare.Neurons[s].InitialConnect(brain, currentSquare)
				}
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
	// Pointers, so the input goes to the real neuron rather than a copy
	var randomSquare = &brain.BrainSquares[rand.Intn(brain.Width)][rand.Intn(brain.Height)]
	if !randomSquare.isWall {
		var pickedNeuron = &randomSquare.Neurons[0]
		pickedNeuron.AddActivationInput(3, time)
	}

	// Swap the lists, what was queued last update gets checked now
	var temp = brain.NeuronsToCheck
	brain.NeuronsToCheck = brain.NextNeuronsToCheck
	brain.NextNeuronsToCheck = temp[:0] // Temp but cleared
	for _, neuron := range brain.NeuronsToCheck {
		neuron.AlreadyAddedToTocheckList = false
		neuron.CheckForFiring(time)
	}
}
