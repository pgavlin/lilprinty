package markdown

import (
	"github.com/pgavlin/goldmark"
	mdtext "github.com/pgavlin/goldmark/text"
	"github.com/pgavlin/lilprinty/internal/bitmap"
	"github.com/pgavlin/lilprinty/internal/font"
)

func Render(device bitmap.Device, bytes []byte, proportionalFamily, monospaceFamily *font.Family, headingStyles []BlockStyle, paragraphStyle BlockStyle) error {
	parser := goldmark.DefaultParser()
	renderer := NewRenderer(proportionalFamily, monospaceFamily, headingStyles, paragraphStyle)
	return renderer.Render(device, bytes, parser.Parse(mdtext.NewReader(bytes)))
}
