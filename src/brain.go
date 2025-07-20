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
	isWall          bool
	DirVecX         float64
	DirVecY         float64
}

// NewGrid creates a new grid of neurons.
func NewBrain(width int, height int, cores int, brainPattern []byte) *Brain {
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
			if !cell.isWall {
				cell.Neurons = append(cell.Neurons, NewNeuron(x, y, cell, true))
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
				cell.Neurons = append(cell.Neurons, NewNeuron(x, y, cell, false))
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
