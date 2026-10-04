package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"runtime/pprof"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// Size of the brain view in screen pixels, set in main to fit the brain at the starting zoom
var screenWidth int
var screenHeight int

const START_ZOOM = 1.5

// Wall clock time spent running simulation cycles each tick (ebiten ticks 60 times a second by default)
// Leaves the rest of the ~16.6ms tick for drawing and input, raise it for a faster simulation but a less responsive window
const SIM_TIME_BUDGET = 12 * time.Millisecond

// Profiling is off unless run with -cpuprofile, e.g. neuron-grid.exe -cpuprofile cpu.prof
var cpuProfilePath = flag.String("cpuprofile", "", "write a CPU profile to this file")
var profileSeconds = flag.Int("profileseconds", 30, "how many seconds of the simulation to profile")

// Draw paints current game state.
func Draw(zoom float64, camX int, camY int, brain *Brain, pix []byte, time float64) {

	// Screen rows are brain X and screen columns are brain Y
	for x := 0; x < screenHeight; x++ {
		for y := 0; y < screenWidth; y++ {

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
					fmt.Println("in str: ", g.brain.Neurons[in.From].IsExcitatory, g.brain.Connections[in.Conn].Strength)
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
	return screenWidth + g.panel.Width, screenHeight
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

// startCPUProfile records where CPU time goes for the given duration, then writes it to path
// View it with: go tool pprof -top neuron-grid.exe cpu.prof
func startCPUProfile(path string, duration time.Duration) {
	f, err := os.Create(path)
	if err != nil {
		log.Fatal("Could not create CPU profile: ", err)
	}
	if err := pprof.StartCPUProfile(f); err != nil {
		log.Fatal("Could not start CPU profile: ", err)
	}
	fmt.Printf("Profiling CPU for %v...\n", duration)

	go func() {
		time.Sleep(duration)
		pprof.StopCPUProfile()
		f.Close()
		fmt.Println("Wrote CPU profile to", path)
	}()
}

func main() {

	flag.Parse()

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

	// Size the view to show the whole brain at the starting zoom, screen columns are brain Y and rows are brain X
	screenWidth = int(float64(brain.Height) * START_ZOOM)
	screenHeight = int(float64(brain.Width) * START_ZOOM)

	// Don't make the window bigger than the monitor, leaving room for the title bar and taskbar
	monitorWidth, monitorHeight := ebiten.Monitor().Size()
	if monitorHeight > 0 {
		screenHeight = min(screenHeight, monitorHeight-100)
	}

	// The panel's width depends on how many slider columns fit at this height, so the width is clamped after it's made
	panel := NewSliderPanel(screenHeight)
	if monitorWidth > 0 {
		screenWidth = min(screenWidth, monitorWidth-panel.Width)
	}
	panel.X = screenWidth

	// Started after setup so the profile only covers the running simulation
	if *cpuProfilePath != "" {
		startCPUProfile(*cpuProfilePath, time.Duration(*profileSeconds)*time.Second)
	}

	g := &Game{
		brain:  brain,
		pixels: make([]byte, screenWidth*screenHeight*4),
		camX:   0,
		camY:   0,
		zoom:   START_ZOOM,

		brainImage: ebiten.NewImage(screenWidth, screenHeight),
		panel:      panel,
	}

	ebiten.SetWindowSize(screenWidth+panel.Width, screenHeight)
	ebiten.SetWindowTitle("Game of Life (Ebitengine Demo)")
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
