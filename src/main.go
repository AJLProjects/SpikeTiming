package main

import (
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

const screenWidth = 500
const screenHeight = 500

// Wall clock time spent running simulation cycles each tick (ebiten ticks 60 times a second by default)
// Leaves the rest of the ~16.6ms tick for drawing and input, raise it for a faster simulation but a less responsive window
const SIM_TIME_BUDGET = 12 * time.Millisecond

// Draw paints current game state.
func Draw(zoom float64, camX int, camY int, brain *Brain, pix []byte, time float64) {

	for x := 0; x < screenWidth; x++ {
		for y := 0; y < screenHeight; y++ {

			// Each brain cell is drawn as a zoom x zoom block of screen pixels
			mappedPosX := int(float64(x)/zoom) + camX
			mappedPosY := int(float64(y)/zoom) + camY

			var normalPos = x*screenWidth + y

			if mappedPosX < 0 || mappedPosY < 0 || mappedPosX >= brain.Width || mappedPosY >= brain.Height {
				pix[4*normalPos] = 0
				pix[4*normalPos+1] = 0
				pix[4*normalPos+2] = 0
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

			var normalPos = y*screenWidth + x // pix is the screen buffer, Draw writes screen row y at y*screenWidth

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
	zoom   float64

	brainImage *ebiten.Image // The brain is drawn here first, then onto the screen left of the slider panel
	panel      *SliderPanel
}

func (g *Game) Update() error {

	g.panel.Update()

	cursorX, cursorY := ebiten.CursorPosition()

	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) && !g.panel.Contains(cursorX, cursorY) {
		g.mouseX = cursorX
		g.mouseY = cursorY
	}

	if ebiten.IsKeyPressed(ebiten.KeyEnter) {

		// Same mapping as Draw, screen row (mouseY) is brain X
		brainX := int(float64(g.mouseY)/g.zoom) + g.camX
		brainY := int(float64(g.mouseX)/g.zoom) + g.camY

		if brainX >= 0 && brainY >= 0 && brainX < g.brain.Width && brainY < g.brain.Height {

			var square = &g.brain.BrainSquares[brainX][brainY]

			if !square.isWall {
				var neuron = &square.Neurons[0]

				fmt.Println("Activation: ", neuron.Activation)

				for _, in := range neuron.Incoming {
					fmt.Println("in str: ", in.From.IsExcitatory, in.Strength)
					fmt.Println("in exp: ", in.DebugLast)
				}
			}
		}
	}

	camSpeed := 5 // In brain cells, the camera is positioned in brain space

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
		g.zoom = 0.5
	} else if ebiten.IsKeyPressed(ebiten.Key2) {
		g.zoom = 1
	} else if ebiten.IsKeyPressed(ebiten.Key3) {
		g.zoom = 1.5
	}

	g.stepSimulation()

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {

	Draw(g.zoom, g.camX, g.camY, g.brain, g.pixels, g.time)

	DrawCursor(g.brain, g.pixels, g.mouseX, g.mouseY)

	g.brainImage.WritePixels(g.pixels)
	screen.DrawImage(g.brainImage, nil)

	g.panel.Draw(screen)
}

// stepSimulation runs as many simulation cycles as fit in SIM_TIME_BUDGET, so the simulation speed isn't tied to the frame rate
func (g *Game) stepSimulation() {
	start := time.Now()
	for time.Since(start) < SIM_TIME_BUDGET {
		g.brain.Update(g.time)
		g.time += 1.0
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth + panelWidth, screenHeight
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
		fmt.Fprintf(os.Stderr, "Error decoding png: %v\n", err2)
		os.Exit(1)
	}

	rgbaPixels := imageToRGBA(img)

	var brain = SetUpBrain(img.Bounds().Dx(), img.Bounds().Dy(), rgbaPixels)

	g := &Game{
		brain:  brain,
		pixels: make([]byte, screenWidth*screenHeight*4),
		camX:   0,
		camY:   0,
		zoom:   1.5,

		brainImage: ebiten.NewImage(screenWidth, screenHeight),
		panel:      NewSliderPanel(screenWidth),
	}

	ebiten.SetWindowSize(screenWidth+panelWidth, screenHeight)
	ebiten.SetWindowTitle("Game of Life (Ebitengine Demo)")
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
