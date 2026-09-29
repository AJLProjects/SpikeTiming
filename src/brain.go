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

			var xAmount = float64((float64(brainPattern[patternPos+0] - 127.0)) / 127.0)
			var yAmount = float64((float64(brainPattern[patternPos+1] - 127.0)) / 127.0)

			var m = math.Sqrt(xAmount*xAmount + yAmount*yAmount)

			//fmt.Printf("Dirvec: %v, %v \n", xAmount/m, 1.0/m)

			brainSquares[x][y] = BrainSquare{
				PosX:    int(x),
				PosY:    int(y),
				isWall:  isWall,
				DirVecX: yAmount / m, // Not sure why these have to flip round
				DirVecY: xAmount / m,
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

func (brain *Brain) AddStandardNeurons() {
	for x := 0; x < brain.Width; x++ {
		for y := 0; y < brain.Height; y++ {
			var cell = &brain.BrainSquares[x][y]
			if !cell.isWall {
				cell.Neurons = append(cell.Neurons, NewNeuron(brain, cell, true))
				cell.NumberOfNeurons = len(cell.Neurons)
			}
		}
	}
}

func (brain *Brain) AddNegativeNeurons() {
	for x := 1; x < brain.Width; x += 2 {
		for y := 1; y < brain.Height; y += 2 {
			var cell = &brain.BrainSquares[x][y]
			if !cell.isWall {
				cell.Neurons = append(cell.Neurons, NewNeuron(brain, cell, false))
				cell.NumberOfNeurons = len(cell.Neurons)
			}
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
	var randomSquare = brain.BrainSquares[rand.Intn(brain.Width)][rand.Intn(brain.Height)]
	if !randomSquare.isWall {
		var pickedNeuron = randomSquare.Neurons[0]
		pickedNeuron.AddActivation(3, time)
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
