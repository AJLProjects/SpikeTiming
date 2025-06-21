package main

import (
	"fmt"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

// Draw paints current game state.
func Draw(brain *Brain, pix []byte, time float64) {

	for x := 0; x < brain.Width; x++ {
		for y := 0; y < brain.Height; y++ {
			var normalPos = x*brain.Width + y

			var timeDif = time - brain.BrainSquares[x][y].Neurons[0].LastfireTime
			timeDif *= 10
			if timeDif > 255 {
				timeDif = 255
			}

			var activationCol = byte(timeDif)

			//if brain.BrainSquares[x][y].Neurons[0].Activation > 0 {
			pix[4*normalPos] = activationCol
			pix[4*normalPos+1] = activationCol
			pix[4*normalPos+2] = activationCol
			pix[4*normalPos+3] = activationCol

		}
	}
}

func DrawCursor(brain *Brain, pix []byte, mouseX int, mouseY int) {

	for x := mouseX - 1; x <= mouseX+1; x++ {
		for y := mouseY - 1; y <= mouseY+1; y++ {

			if x == mouseX && y == mouseY {
				continue
			}

			wrappedPosX, wrappedPosY := brain.WrapPosition(y, x)

			var normalPos = wrappedPosX*brain.Width + wrappedPosY

			//if brain.BrainSquares[x][y].Neurons[0].Activation > 0 {
			pix[4*normalPos] = 0xFF
			pix[4*normalPos+1] = 0x00
			pix[4*normalPos+2] = 0x00
			pix[4*normalPos+3] = 0xFF

		}
	}
}

type Game struct {
	brain  *Brain
	pixels []byte
	time   float64
	mouseX int
	mouseY int
}

func (g *Game) Update() error {

	cursorX, cursorY := ebiten.CursorPosition()

	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		g.mouseX = cursorX
		g.mouseY = cursorY
	}

	if ebiten.IsKeyPressed(ebiten.KeyEnter) {

		wrappedPosX, wrappedPosY := g.brain.WrapPosition(g.mouseY, g.mouseX)

		var neuron = g.brain.BrainSquares[wrappedPosX][wrappedPosY].Neurons[0]

		fmt.Println("Activation: ", neuron.Activation)

		for _, in := range neuron.Incoming {
			fmt.Println("in str: ", in.From.IsExcitatory, in.Strength)
			fmt.Println("in exp: ", in.DebugLast)
		}

	}

	// 10 updates per frame
	for i := 0; i <= 10; i++ {
		g.brain.Update(g.time)
		g.time += 1.0
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {

	Draw(g.brain, g.pixels, g.time)

	DrawCursor(g.brain, g.pixels, g.mouseX, g.mouseY)

	screen.WritePixels(g.pixels)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}

func main() {

	var brain = SetUpBrain()

	g := &Game{
		brain:  brain,
		pixels: make([]byte, brain.Width*brain.Height*4),
	}

	ebiten.SetWindowSize(brain.Width, brain.Height)
	ebiten.SetWindowTitle("Game of Life (Ebitengine Demo)")
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
