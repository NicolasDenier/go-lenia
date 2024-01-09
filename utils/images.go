package utils

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"time"

	"fyne.io/fyne/v2"
)

func CropImage(img image.Image, width, height int) image.Image {
	// crop an image to keep only the left half (the raster)
	cropSize := image.Rect(0, 0, width, height)
	new := image.NewRGBA(cropSize)
	for i := 0; i < width; i++ {
		for j := 0; j < height; j++ {
			r, g, b, a := img.At(i, j).RGBA()
			new.SetRGBA(i, j, color.RGBA{uint8(r), uint8(g), uint8(b), uint8(a)})
		}
	}
	return new
}

func SaveImage(w fyne.Window, width, height int) error {
	// capture the current rendered image
	img := w.Canvas().Capture()
	img = CropImage(img, width, height)
	// create the file
	t := time.Now()
	date := fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02d",
		t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second())
	path := fmt.Sprintf("images/%s.png", date)
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	// encode the image to PNG format
	err = png.Encode(file, img)
	if err != nil {
		return err
	}
	return nil
}
