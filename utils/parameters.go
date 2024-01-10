package utils

import (
	"fmt"
	"image/color"
	"math"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
)

// a changable parameter through a slider
type Parameter struct {
	Bind   binding.Float
	Slider *widget.Slider
	// pointer to the variable it is linked to
	variable *float64
}

type ManageParameter interface {
	Initialize()
	GetValue()
	GetStringValue()
	CreateSlider()
	GetSliderBox()
	OnSliderChange()
	OnSliderChangeOther()
	Update()
}

func (p *Parameter) Initialize(value float64, configVar *float64) {
	// set an initial value to a parameter
	// update linked variable
	p.variable = configVar
	//p.Update(value)
	// binding
	p.Bind = binding.NewFloat()
	p.Bind.Set(value)
}

func (p *Parameter) GetValue() float64 {
	// get the current value of the parameter
	v, _ := p.Bind.Get()
	return v
}

func (p *Parameter) GetStringValue() string {
	// get the current value as string
	return fmt.Sprintf("%.3f", p.GetValue())
}

func (p *Parameter) CreateSlider(min, max, precision float64) {
	// create the parameter slider
	p.Slider = widget.NewSliderWithData(min, max, p.Bind)
	p.Slider.Step = precision
}

func (p *Parameter) OnSliderChange(valueLabel *widget.Label) {
	// update the linked variable on change and the value label
	p.Slider.OnChangeEnded = func(v float64) {
		p.Update(v)
		valueLabel.SetText(p.GetStringValue())
		valueLabel.Refresh()
	}
}

func (p *Parameter) OnSliderChangeOther(valueLabel *widget.Label, otherVar *float64) {
	// update the linked variables on change and the value label
	p.Slider.OnChangeEnded = func(v float64) {
		p.Update(v)
		*otherVar = 1 / v
		valueLabel.SetText(p.GetStringValue())
		valueLabel.Refresh()
	}
}

func (p *Parameter) GetSliderBox(min, max, precision float64, label string, otherVar *float64) *fyne.Container {
	// generate a box containing the name of a variable, a slider and its value that is updated on slider change
	text := widget.NewLabel(label)
	valueLabel := widget.NewLabel(p.GetStringValue())
	p.CreateSlider(min, max, precision)
	box := container.NewBorder(nil, nil, text, valueLabel, p.Slider)
	if otherVar == nil {
		p.OnSliderChange(valueLabel)
	} else {
		// also update another variable (for example T updates dT=1/T)
		p.OnSliderChangeOther(valueLabel, otherVar)
	}
	return box
}

func (p *Parameter) Update(value float64) {
	// update the parameter linked variable
	*p.variable = value
}

func FlagToBeta(s string) []float64 {
	// parse the -b flag values to a float array
	var beta []float64
	for _, value := range strings.Split(s, ",") {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil {
			beta = append(beta, parsed)
		}
	}
	return beta
}

// colormap choice

type ColormapButton struct {
	colors  *[][]int
	Buttons *widget.RadioGroup
}

type ManageColormapButton interface {
	initColormaps()
	GetColor()
}

func (c *ColormapButton) initColormaps(raster *canvas.Raster) {
	c.Buttons.OnChanged = func(value string) {
		switch value {
		case "White":
			*c.colors = [][]int{{255, 255, 255}, {0, 0, 0}}
		case "Black":
			*c.colors = [][]int{{0, 0, 0}, {255, 255, 255}}
		case "Inferno":
			*c.colors = [][]int{
				{0, 0, 4},
				{87, 16, 110},
				{188, 55, 84},
				{249, 142, 9},
				{252, 255, 164},
			}
		case "Viridis":
			*c.colors = [][]int{
				{68, 1, 84},
				{59, 82, 139},
				{33, 145, 140},
				{94, 201, 98},
				{253, 231, 37},
			}
		}
		raster.Refresh()
	}
}

func CreateColormapButton(colors *[][]int, raster *canvas.Raster) ColormapButton {
	radio := widget.NewRadioGroup([]string{"White", "Black", "Inferno", "Viridis"}, nil)
	cButton := ColormapButton{
		colors:  colors,
		Buttons: radio,
	}
	cButton.initColormaps(raster)
	cButton.Buttons.SetSelected("White")
	return cButton
}

func interpolate(x float64, a, b int) uint8 {
	// gives the value at x between a and b. x between 0 and 1
	return uint8(float64(a) + float64(b-a)*x)
}

func (c *ColormapButton) GetColor(v float64) color.Color {
	// return the color corresponding to v
	scaledV := v * float64((len(*c.colors) - 1))
	index1 := int(math.Floor(scaledV))
	index2 := int(math.Ceil(scaledV))
	x := scaledV - float64(index1)
	c1 := (*c.colors)[index1]
	c2 := (*c.colors)[index2]
	return color.RGBA{
		interpolate(x, c1[0], c2[0]),
		interpolate(x, c1[1], c2[1]),
		interpolate(x, c1[2], c2[2]),
		0xff,
	}
}
