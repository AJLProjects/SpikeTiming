package main

import (
	"fmt"
	"math"
	"runtime"
)

func checkWholeNumber(brainWidth int, numberOfCores int) {

	var num = float64(brainWidth) / float64(numberOfCores*2)
	var _, rem = math.Modf(num)
	if rem > 0 {

		var floored = math.Floor(float64(brainWidth) / float64(numberOfCores*3))
		var suggested = floored * float64(numberOfCores*3)
		panic("doesnt divide nicely, try: " + fmt.Sprint(suggested))
	}
}

// 5k*5k is 25 million, just enough this takes up 50GB AT LEAST
// 200 is standard
func SetUpBrain(brainWidth int, brainHeight int, brainPattern []byte) *Brain {

	fmt.Printf("Size: %v %v \n", brainWidth, brainHeight)

	threads := runtime.NumCPU()
	fmt.Printf("This machine has %d CPU threads. \n", threads) // 32 for my pc
	runtime.GOMAXPROCS(threads)

	fmt.Println("Creating Neuron Grid...")

	checkWholeNumber(brainWidth, threads)

	var stripWidth = brainWidth / (threads * 2) // 2 lanes per thread

	fmt.Println("Strip width ", stripWidth)

	fmt.Println("Making brain")

	myBrain := NewBrain(brainWidth, brainHeight, threads, brainPattern)

	myBrain.AddStandardNeurons()

	fmt.Println("Finished adding standard neurons")

	myBrain.AddNegativeNeurons()

	fmt.Println("Finished adding negative neurons")

	myBrain.ConnectNeurons()

	fmt.Println("Finished connecting neurons")

	fmt.Println("Finished making brain")

	// TODO: Kind of remove this, make it event driven if possible
	/*for y := 0; y < gridHeight; y++ {
		for x := 0; x < gridWidth; x++ {
			fmt.Println(myBrain.BrainSquares)
		}
	}*/

	//fmt.Println("Press 'Enter' to continue...")
	//bufio.NewReader(os.Stdin).ReadBytes('\n')

	MakeMultiThreadStrips(myBrain, threads)

	fmt.Println("Finished associating threadstrips")

	// Iterate through the strips and associate

	return myBrain
}
