package renderer

import (
	"bytes"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"math"

	"golang.org/x/image/math/fixed"

	"github.com/pgavlin/lilprinty/internal/util"
)

func fixedToFloat(f fixed.Int26_6) float64 {
	integer := float64(f / 64)
	frac := float64(f&63) / 64.0
	return integer + frac
}

func floatToFixed(f float64) fixed.Int26_6 {
	integer := math.Trunc(f)
	frac := f - integer
	return fixed.Int26_6(int(integer)*64 | int(math.Trunc(frac*64.0))&63)
}

func downloadImage(url string) (image.Image, error) {
	contents, _, err := util.DownloadFile(url)
	if err != nil {
		return nil, err
	}

	img, _, err := image.Decode(bytes.NewReader(contents))
	return img, err
}
