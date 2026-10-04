package main

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const panelWidth = 300
const panelPadding = 12
const sliderTop = 40
const sliderRowHeight = 46
const sliderLabelHeight = 20 // Space for the label text above the bar
const sliderBarHeight = 8
const sliderKnobWidth = 6
const debugCharWidth = 6 // Width of a character in ebitenutil's debug font

var panelBackground = color.RGBA{0x20, 0x22, 0x28, 0xFF}
var sliderTrackColor = color.RGBA{0x45, 0x48, 0x52, 0xFF}
var sliderFillColor = color.RGBA{0x3D, 0x8B, 0xF0, 0xFF}
var sliderKnobColor = color.RGBA{0xFF, 0xFF, 0xFF, 0xFF}

// Slider edits a float64 variable in place, so the simulation picks up the change on its next read
type Slider struct {
	Label    string
	Value    *float64
	Min      float64
	Max      float64
	Default  float64
	LogScale bool // For values that span several orders of magnitude, Min must be > 0
}

func (s *Slider) fraction() float64 {
	if s.LogScale {
		return math.Log(*s.Value/s.Min) / math.Log(s.Max/s.Min)
	}
	return (*s.Value - s.Min) / (s.Max - s.Min)
}

func (s *Slider) setFraction(f float64) {
	f = math.Max(0, math.Min(1, f))
	if s.LogScale {
		*s.Value = s.Min * math.Pow(s.Max/s.Min, f)
	} else {
		*s.Value = Lerp(s.Min, s.Max, f)
	}
}

// SliderPanel is drawn to the right of the brain view, starting at X
type SliderPanel struct {
	X       int
	Sliders []*Slider
	active  int // Index of the slider being dragged, -1 when none
}

func NewSliderPanel(x int) *SliderPanel {

	slider := func(label string, value *float64, min float64, max float64, logScale bool) *Slider {
		return &Slider{Label: label, Value: value, Min: min, Max: max, Default: *value, LogScale: logScale}
	}

	return &SliderPanel{
		X:      x,
		active: -1,
		Sliders: []*Slider{
			slider("Refactory period", &REFACTORY_PERIOD, 0, 30, false),
			slider("Refactory peak fraction", &REFACTORY_PEAK_FRACTION, 0, 1, false),
			slider("Activation decay rate", &ACTIVATION_DECAY_RATE, 0.8, 0.999, false),
			slider("Neighbour punish/reward fire window", &NEIGHBOUR_FIRE_WINDOW, 0.5, 50, true),
			slider("Neighbour sync rate", &NEIGHBOUR_SYNC_RATE, 0, 1, false),
			slider("Neighbour sync crossover from good to bad", &NEIGHBOUR_SYNC_CROSSOVER, 0.25, 8, false),
			slider("Neighbour sync max strength", &NEIGHBOUR_SYNC_MAX_STRENGTH, 0.1, 3, false),
			slider("Quiet reward per cycle", &QUIET_REWARD_PER_CYCLE, 0.00001, 0.05, true),
			slider("Post synaptic fire reward", &POST_SYNAPTIC_FIRE_REWARD, 0, 0.5, false),
		},
	}
}

func (p *SliderPanel) barRect(i int) (x float32, y float32, w float32, h float32) {
	x = float32(p.X + panelPadding)
	y = float32(sliderTop + i*sliderRowHeight + sliderLabelHeight)
	return x, y, panelWidth - 2*panelPadding, sliderBarHeight
}

// sliderAt returns the index of the slider row under the point, or -1
func (p *SliderPanel) sliderAt(x int, y int) int {
	if x < p.X || x >= p.X+panelWidth || y < sliderTop {
		return -1
	}
	i := (y - sliderTop) / sliderRowHeight
	if i >= len(p.Sliders) {
		return -1
	}
	return i
}

// Contains reports whether the point is over the panel, so clicks there don't move the brain cursor
func (p *SliderPanel) Contains(x int, y int) bool {
	return x >= p.X
}

// Update handles dragging with the left mouse button, right clicking a slider resets it to its default
func (p *SliderPanel) Update() {

	mouseX, mouseY := ebiten.CursorPosition()

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		p.active = p.sliderAt(mouseX, mouseY)
	}
	if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		p.active = -1
	}

	if p.active >= 0 {
		barX, _, barW, _ := p.barRect(p.active)
		p.Sliders[p.active].setFraction((float64(mouseX) - float64(barX)) / float64(barW))
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		if i := p.sliderAt(mouseX, mouseY); i >= 0 {
			*p.Sliders[i].Value = p.Sliders[i].Default
		}
	}
}

func (p *SliderPanel) Draw(screen *ebiten.Image) {

	vector.DrawFilledRect(screen, float32(p.X), 0, panelWidth, float32(screen.Bounds().Dy()), panelBackground, false)

	ebitenutil.DebugPrintAt(screen, "Drag to change, right click to reset", p.X+panelPadding, 12)

	for i, s := range p.Sliders {

		barX, barY, barW, barH := p.barRect(i)
		labelY := sliderTop + i*sliderRowHeight

		label := s.Label
		if *s.Value != s.Default {
			label += " *" // Mark values that have been changed from their default
		}
		ebitenutil.DebugPrintAt(screen, label, int(barX), labelY)

		valueText := fmt.Sprintf("%.4g", *s.Value)
		ebitenutil.DebugPrintAt(screen, valueText, int(barX+barW)-len(valueText)*debugCharWidth, labelY)

		fillW := barW * float32(math.Max(0, math.Min(1, s.fraction())))
		vector.DrawFilledRect(screen, barX, barY, barW, barH, sliderTrackColor, false)
		vector.DrawFilledRect(screen, barX, barY, fillW, barH, sliderFillColor, false)
		vector.DrawFilledRect(screen, barX+fillW-sliderKnobWidth/2, barY-4, sliderKnobWidth, barH+8, sliderKnobColor, false)
	}
}
