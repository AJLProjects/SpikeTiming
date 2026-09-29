package main

import (
	"fmt"
)

// 5k*5k is 25 million, just enough this takes up 50GB AT LEAST
// 200 is standard
func SetUpBrain(brainWidth int, brainHeight int, brainPattern []byte) *Brain {

	fmt.Printf("Size: %v %v \n", brainWidth, brainHeight)

	fmt.Println("Making brain")

	myBrain := NewBrain(brainWidth, brainHeight, brainPattern)

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

	return myBrain
}
