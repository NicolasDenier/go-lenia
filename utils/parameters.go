package utils

import (
	"fmt"

	"fyne.io/fyne/v2"
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
	text := widget.NewLabel(label)
	valueLabel := widget.NewLabel(p.GetStringValue())
	p.CreateSlider(min, max, precision)
	box := container.NewBorder(nil, nil, text, valueLabel, p.Slider)
	if otherVar == nil {
		p.OnSliderChange(valueLabel)
	} else {
		p.OnSliderChangeOther(valueLabel, otherVar)
	}
	return box
}

func (p *Parameter) Update(value float64) {
	// update the parameter linked variable
	*p.variable = value
}
