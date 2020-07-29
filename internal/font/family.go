package font

import (
	"fmt"

	"github.com/golang/freetype/truetype"
)

type Family struct {
	options truetype.Options

	regularFont    *truetype.Font
	boldFont       *truetype.Font
	italicFont     *truetype.Font
	boldItalicFont *truetype.Font

	sizes map[float64]*FaceFamily
}

func ParseFamily(regular, bold, italic, boldItalic []byte, options truetype.Options) (*Family, error) {
	regularFont, err := truetype.Parse(regular)
	if err != nil {
		return nil, fmt.Errorf("failed to parse regular font: %w", err)
	}
	boldFont, err := truetype.Parse(bold)
	if err != nil {
		return nil, fmt.Errorf("failed to parse bold font: %w", err)
	}
	italicFont, err := truetype.Parse(italic)
	if err != nil {
		return nil, fmt.Errorf("failed to parse italic font: %w", err)
	}
	boldItalicFont, err := truetype.Parse(boldItalic)
	if err != nil {
		return nil, fmt.Errorf("failed to parse boldItalic font: %w", err)
	}

	return &Family{
		options:        options,
		regularFont:    regularFont,
		boldFont:       boldFont,
		italicFont:     italicFont,
		boldItalicFont: boldItalicFont,
		sizes:          map[float64]*FaceFamily{},
	}, nil
}

func (f *Family) Size(pointSize float64) *FaceFamily {
	if faceFamily, ok := f.sizes[pointSize]; ok {
		return faceFamily
	}

	faceFamily := &FaceFamily{
		family:    f,
		pointSize: pointSize,
	}
	f.sizes[pointSize] = faceFamily
	return faceFamily
}

func (f *Family) Face(pointSize float64, bold, italic bool) *Face {
	return f.Size(pointSize).Face(bold, italic)
}
