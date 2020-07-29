package bitmap

import (
	"image"

	"github.com/MaxHalford/halfgone"
	"github.com/nfnt/resize"
)

// A Device is an infinitely-tall, 1-bit output device with a maximum width and a DPI.
type Device interface {
	MaxWidth() int
	DPI() float64

	PrintBitmap(img *Image) error
	Feed(lines int) error
}

// ForDevice converts the input image to a device-appropriate bitmap, downscaling and dithering as necessary.
func ForDevice(output Device, img image.Image, dither bool) *Image {
	// Scale down to size.
	thumb := resize.Thumbnail(uint(output.MaxWidth()), uint(img.Bounds().Dy()), img, resize.Bilinear)

	// Convert the image to grayscale.
	gray := halfgone.ImageToGray(thumb)

	// Apply Floyd-Steinberg dithering.
	if dither {
		var floydSteinberg halfgone.FloydSteinbergDitherer
		gray = floydSteinberg.Apply(gray)
	}
	return &Image{src: gray, threshold: 128}
}
