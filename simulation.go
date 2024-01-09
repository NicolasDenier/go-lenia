package main

import (
	"flag"
	"fmt"
	"image/color"
	"math"
	"rd/utils"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"gonum.org/v1/gonum/mat"
)

/*
Lenia system
https://arxiv.org/pdf/1812.05433.pdf

press 's' to save image
press 'c' to close window
*/

const width = 600
const height = 600

var simulationApp = app.New()
var kFlag bool
var running bool = true
var wg sync.WaitGroup

// define system parameters
var R utils.Parameter
var T utils.Parameter
var Mu utils.Parameter
var Sigma utils.Parameter

// create and initialize a new config as current setup
var setup utils.Config

func initParameters(R_val, T_val, Mu_val, Sigma_val float64, Beta_val []float64) {
	setup = utils.NewConfig(width, height, R_val, T_val, Mu_val, Sigma_val, Beta_val)
	// assign each parameter to a setup variable and set the initial values
	R.Initialize(R_val, &setup.R)
	T.Initialize(T_val, &setup.T)
	Mu.Initialize(Mu_val, &setup.Mu)
	Sigma.Initialize(Sigma_val, &setup.Sigma)
}

func displayState(i, j, w, h int) color.Color {
	// update the pixels colors according to the reaction diffusion state matrices
	if i < width && j < height {
		amount := 1 - setup.A.At(i, j)
		col := uint8(utils.Clip(amount, 0, 1) * 255)
		return color.RGBA{
			col,
			col,
			col,
			0xff}
	} else {
		return color.Black
	}
}

func displayKernel(i, j, w, h int) color.Color {
	// update the pixels colors according to the reaction diffusion state matrices
	len := int(setup.R*2 + 1)
	if i < len && j < len {
		amount := setup.Kernel.At(i, j) / mat.Max(setup.Kernel)
		col := uint8(utils.Clip(amount, 0, 1) * 255)
		return color.RGBA{
			col,
			col,
			col,
			0xff}
	} else {
		return color.White
	}
}

func animate(raster *canvas.Raster) {
	// update the canvas at a regulat time tick
	for range time.Tick(time.Millisecond * time.Duration(1000*setup.Dt)) {
		if running {
			wg.Add(1)
			setup.Update()
			raster.Refresh()
			wg.Done()
		}
	}
}

func getMargin(length int) float32 {
	return float32(math.Round(float64(length)*0.23) + 1)
}

func initWindow(title string, winWidth, winHeight float32) fyne.Window {
	// define the window and its properties
	w := simulationApp.NewWindow(title)
	w.SetFixedSize(true) // starts as floating window
	w.SetPadded(false)
	w.Resize(fyne.NewSize(winWidth, winHeight))
	return w
}

func StartButton() *widget.Button {
	startButton := widget.NewButton("stop", nil)
	startButton.OnTapped = func() {
		running = !running
		if running {
			startButton.Text = "stop"
		} else {
			startButton.Text = "start"
		}
		startButton.Refresh()
	}
	return startButton
}

func RestartButton(raster *canvas.Raster) *widget.Button {
	restartButton := widget.NewButton("restart", func() {
		// stop simulation, wait for last update to complete and restart it with a new initial state
		wasRunning := running
		running = false
		wg.Wait()
		setup.A = mat.NewDense(width, height, nil)
		setup.InitState()
		raster.Refresh()
		running = wasRunning
	})
	return restartButton
}

func leniaWindow() fyne.Window {
	// define the lenia app
	// define window size
	winWidth := 2 * (width - getMargin(width))
	winHeight := height - getMargin(height)
	w := initWindow("Lenia State", winWidth, winHeight)
	// raster is the pixel matrix and its update function
	raster := canvas.NewRasterWithPixels(displayState)
	// buttons
	buttons := container.New(layout.NewHBoxLayout(),
		StartButton(), RestartButton(raster))

	// sliders
	controls := container.New(layout.NewVBoxLayout(),
		R.GetSliderBox(0, 200, 1, "R", &setup.Dx),
		T.GetSliderBox(0, 100, 1, "T", &setup.Dt),
		Mu.GetSliderBox(0, 1, 0.001, "Mu", nil),
		Sigma.GetSliderBox(0, 1, 0.001, "Sigma", nil),
		buttons)
	// 2 columns: lenia state and parameters
	grid := container.New(layout.NewGridLayout(2), raster, controls)
	w.SetContent(grid)
	// launch animation
	go animate(raster)
	return w
}

func kernelWindow() fyne.Window {
	winWidth := 2*float32(setup.R) + 1
	winMargin := getMargin(int(winWidth))
	w := initWindow("Lenia Kernel", winWidth-winMargin, winWidth-winMargin)
	raster := canvas.NewRasterWithPixels(displayKernel)
	w.SetContent(raster)
	return w
}

func listenKeys(w fyne.Window) {
	// listen for key press
	w.Canvas().SetOnTypedKey(func(k *fyne.KeyEvent) {
		switch k.Name {
		// screenshot
		case "S":
			fmt.Println("Image saved")
			if kFlag {
				winWidth := int(2 * setup.R)
				utils.SaveImage(w, winWidth, winWidth)
			} else {
				utils.SaveImage(w, width, height)
			}
		// close
		case "C":
			w.Close()
		}
	})
}

func main() {
	var w fyne.Window
	flag.BoolVar(&kFlag, "k", false, "display the kernel")
	flag.Parse()

	initParameters(80, 10, 0.23, 0.024, []float64{1, 0.6, 0.3})

	if kFlag {
		w = kernelWindow()
	} else {
		w = leniaWindow()
	}
	listenKeys(w)
	w.ShowAndRun()
}
