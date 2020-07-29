package bitmap

import (
	"image"
	"image/color"
)

// ColorModel is the default color model for bitmaps.
var ColorModel color.Model = ThresholdColorModel(128)

type thresholdModel byte

func (t thresholdModel) Convert(c color.Color) color.Color {
	return model(color.GrayModel.Convert(c).(color.Gray), byte(t))
}

// ThresholdColorModel returns a color model with the given threshold.
func ThresholdColorModel(threshold byte) color.Model {
	return thresholdModel(threshold)
}

func model(c color.Gray, threshold byte) color.Color {
	if c.Y >= threshold {
		return color.White
	}
	return color.Black
}

// A Image is a 1-bit image.
type Image struct {
	src       *image.Gray
	threshold byte
}

// New creates a new Image with the given bounds.
func New(r image.Rectangle) *Image {
	return &Image{src: image.NewGray(r), threshold: 128}
}

// NewWithThreshold creates a new Image with the given bounds and threshold.
func NewThreshold(r image.Rectangle, threshold byte) *Image {
	return &Image{src: image.NewGray(r), threshold: threshold}
}

// ColorModel returns the Image's color model.
func (b *Image) ColorModel() color.Model {
	return ThresholdColorModel(b.threshold)
}

// Bounds returns the domain for which At can return non-zero color.
// The bounds do not necessarily contain the point (0, 0).
func (b *Image) Bounds() image.Rectangle {
	return b.src.Bounds()
}

// At returns the color of the pixel at (x, y).
// At(Bounds().Min.X, Bounds().Min.Y) returns the upper-left pixel of the grid.
// At(Bounds().Max.X-1, Bounds().Max.Y-1) returns the lower-right one.
//
// Set bits return color.White; unset bits return color.Black.
func (b *Image) At(x, y int) color.Color {
	return model(b.src.GrayAt(x, y), b.threshold)
}

// BitAt returns true if the bit at (x, y) is set and false if it is not.
func (b *Image) BitAt(x, y int) bool {
	return b.src.GrayAt(x, y).Y >= b.threshold
}

// Set sets the color of the pixel at (x, y).
func (b *Image) Set(x, y int, c color.Color) {
	b.src.Set(x, y, b.ColorModel().Convert(c))
}

// SetBit sets or clears the bit at (x, y).
func (b *Image) SetBit(x, y int, v bool) {
	if v {
		b.src.Set(x, y, color.White)
	} else {
		b.src.Set(x, y, color.Black)
	}
}
