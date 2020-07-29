package main

import (
	"fmt"
	"image"
	"image/color"
	"sort"

	"github.com/pgavlin/lilprinty/internal/bitmap"
)

type slice struct {
	contents *bitmap.Image
	yOrigin  int
}

type preview struct {
	slices []slice
	height int
}

func (preview) MaxWidth() int {
	return 384
}

func (preview) DPI() float64 {
	return 203.2
}

func (p *preview) PrintBitmap(img *bitmap.Image) error {
	if img.Bounds().Dx() > 384 {
		return fmt.Errorf("bitmap must be less than 384 pixels wide")
	}
	p.slices = append(p.slices, slice{
		contents: img,
		yOrigin:  p.height,
	})
	p.height += img.Bounds().Dy()
	return nil
}

func (p *preview) Feed(lines int) error {
	p.height += lines
	return nil
}

func (p *preview) ColorModel() color.Model {
	return bitmap.ColorModel
}

func (p *preview) Bounds() image.Rectangle {
	return image.Rect(0, 0, 384+80, p.height)
}

func (p *preview) At(x, y int) color.Color {
	if x < 0 || x >= 384+80 || y < 0 || y >= p.height {
		return color.Black
	}

	x -= 40
	if x < 0 || x >= 384 {
		return color.White
	}

	i := sort.Search(len(p.slices), func(i int) bool {
		s := p.slices[i]
		return s.yOrigin+s.contents.Bounds().Dy() > y
	})
	if i >= len(p.slices) {
		return color.White
	}

	s := p.slices[i]
	y -= s.yOrigin
	if y < 0 || y >= s.contents.Bounds().Dy() {
		return color.White
	}

	if s.contents.BitAt(x, y) {
		return color.White
	}
	return color.Black
}
