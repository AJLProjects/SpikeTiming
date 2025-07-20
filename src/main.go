package main

import (
	"fmt"
	"image"
	"image/png"
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
)

const screenWidth = 1000
const screenHeight = 1000

// Draw paints current game state.
func Draw(zoom int, camX int, camY int, brain *Brain, pix []byte, time float64) {

	for x := 0; x < screenWidth; x++ {
		for y := 0; y < screenHeight; y++ {

			mappedPosX := x*zoom + camX
			mappedPosY := y*zoom + camY

			var normalPos = x*screenWidth + y

			if mappedPosX < 0 || mappedPosY < 0 || mappedPosX >= brain.Width || mappedPosY >= brain.Height {
				pix[4*normalPos] = 0
				pix[4*normalPos+1] = 0
				pix[4*normalPos+2] = 255
				pix[4*normalPos+3] = 255
				continue
			}

			var curSquare = &brain.BrainSquares[mappedPosX][mappedPosY]

			if curSquare.isWall {
				pix[4*normalPos] = 3
				pix[4*normalPos+1] = 4
				pix[4*normalPos+2] = 5
				pix[4*normalPos+3] = 6
			} else {

				var timeDif = time - curSquare.Neurons[0].LastfireTime
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
}

func DrawCursor(brain *Brain, pix []byte, mouseX int, mouseY int) {

	for x := mouseX - 1; x <= mouseX+1; x++ {
		for y := mouseY - 1; y <= mouseY+1; y++ {

			if x == mouseX && y == mouseY {
				continue
			}

			if x < 0 || y < 0 || x >= screenWidth || y >= screenHeight {
				continue
			}

			var normalPos = x*brain.Width + y

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
	camX   int
	camY   int
	zoom   int
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

	camSpeed := 5 * g.zoom

	if ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
		g.camY += camSpeed
	} else if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
		g.camY -= camSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
		g.camX -= camSpeed
	} else if ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
		g.camX += camSpeed
	}

	if ebiten.IsKeyPressed(ebiten.Key1) {
		g.zoom = 1
	} else if ebiten.IsKeyPressed(ebiten.Key2) {
		g.zoom = 2
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {

	// 10 updates per frame
	for i := 0; i <= 5; i++ {
		g.brain.Update(g.time)
		g.time += 1.0
	}

	Draw(g.zoom, g.camX, g.camY, g.brain, g.pixels, g.time)

	DrawCursor(g.brain, g.pixels, g.mouseX, g.mouseY)

	screen.WritePixels(g.pixels)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}

func imageToRGBA(img image.Image) []uint8 {
	sz := img.Bounds()
	raw := make([]uint8, (sz.Max.X-sz.Min.X)*(sz.Max.Y-sz.Min.Y)*4)
	idx := 0
	for y := sz.Min.Y; y < sz.Max.Y; y++ {
		for x := sz.Min.X; x < sz.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			raw[idx], raw[idx+1], raw[idx+2], raw[idx+3] = uint8(r), uint8(g), uint8(b), uint8(a)
			idx += 4
		}
	}
	return raw
}

func main() {

	file, err := os.Open("map/brain-map.png")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	// Decode the PNG image.
	img, err2 := png.Decode(file)
	if err2 != nil {
		fmt.Fprintf(os.Stderr, "Error decoding png: %v\n", err)
		os.Exit(1)
	}

	rgbaPixels := imageToRGBA(img)

	var brain = SetUpBrain(img.Bounds().Dx(), img.Bounds().Dy(), rgbaPixels)

	g := &Game{
		brain:  brain,
		pixels: make([]byte, screenWidth*screenHeight*4),
		camX:   0,
		camY:   0,
		zoom:   1,
	}

	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Game of Life (Ebitengine Demo)")
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
