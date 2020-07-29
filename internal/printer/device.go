package printer

import (
	"fmt"
	"io"

	"github.com/pgavlin/lilprinty/internal/bitmap"
)

type Device struct {
	w io.Writer
}

func New(w io.Writer) *Device {
	return &Device{w: w}
}

func (d *Device) MaxWidth() int {
	return 384
}

func (d *Device) DPI() float64 {
	return 203.2
}

func (d *Device) PrintBitmap(img *bitmap.Image) error {
	if img.Bounds().Dx() > 384 {
		return fmt.Errorf("bitmap must be less than 384 pixels wide")
	}

	// Print the image one scanline at a time.
	row := make([]byte, 4+img.Bounds().Dx()/8)
	row[0], row[1], row[2], row[3] = 0x12, 0x2A, 1, byte(len(row)-4)

	for y := 0; y < img.Bounds().Dy(); y++ {
		// Render the row.
		for x := 0; x < img.Bounds().Dx(); x++ {
			byteOffset, bitOffset := 4+x/8, 7-x%8

			// A set bit in the bitmap corresponds to the color white, but a set bit in the output corresponds to the
			// color black, so we need to invert the bits when rendering the row.
			if img.BitAt(x, y) {
				row[byteOffset] &^= 1 << bitOffset
			} else {
				row[byteOffset] |= 1 << bitOffset
			}
		}

		// Write the row.
		if _, err := d.w.Write(row); err != nil {
			return err
		}
	}
	return nil
}

func (p *Device) Feed(lines int) error {
	if lines < 0 || lines >= 256 {
		return fmt.Errorf("lines must be in the range [0, 256)")
	}
	_, err := p.w.Write([]byte{0x1b, 0x4a, byte(lines)})
	return err
}
