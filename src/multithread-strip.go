package main

type DeviceMultithreadStrip struct {
	ThreadA MultithreadStrip
	ThreadB MultithreadStrip
}

type MultithreadStrip struct {
	startX int
	endX   int
	// Neurons to check
	NeuronsToCheck     []*Neuron //This can become a complet type with NeuronsToCheck as a naggretate, we would then just swap pointers for next
	NextNeuronsToCheck []*Neuron
}

func MakeMultiThreadStrips(brain *Brain, numberOfCores int) {

	var numberOfStrips = numberOfCores * 3

	strips := make([]*MultithreadStrip, 0, numberOfStrips)

	var stripWidth = brain.Width / numberOfStrips

	for i := 0; i < numberOfStrips; i++ {
		strips = append(strips, &MultithreadStrip{
			startX:         i * stripWidth,
			endX:           ((i + 1) * stripWidth) - 1, // This is the index, so when iterating over do <=
			NeuronsToCheck: make([]*Neuron, 0),
		})
	}

	brain.ThreadStrips = strips

	AssociateMultithreadStripsWithNeurons(brain, strips)
}

func AssociateMultithreadStripsWithNeurons(brain *Brain, strips []*MultithreadStrip) {
	for s := 0; s < len(strips); s++ {
		curStrip := strips[s]
		for x := curStrip.startX; x <= curStrip.endX; x += 1 {
			for y := 0; y < brain.Height; y += 1 {
				currentSquare := &brain.BrainSquares[x][y]
				for n := 0; n < currentSquare.NumberOfNeurons; n++ {
					currentSquare.Neurons[n].ThreadStrip = curStrip
				}
			}
		}

	}

}
