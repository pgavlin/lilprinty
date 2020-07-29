package renderer

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"net/url"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/datamatrix"
	"github.com/pgavlin/goldmark/ast"
	mdtext "github.com/pgavlin/goldmark/text"
	"golang.org/x/image/math/fixed"

	"github.com/pgavlin/lilprinty/internal/bitmap"
	"github.com/pgavlin/lilprinty/internal/font"
)

const indentAmount = 4.5

type listState struct {
	node        *ast.List
	markerWidth float64
	index       int
}

// BlockStyle describes the style for a block node.
type BlockStyle struct {
	PointSize    float64 // The size of the block's font face in points.
	TopMargin    float64 // The top margin of the block in points.
	BottomMargin float64 // The bottom margin of the block in points.
}

type Renderer struct {
	proportionalFamily *font.Family
	monospaceFamily    *font.Family

	headingStyles  []BlockStyle
	paragraphStyle BlockStyle

	listStack   []listState
	faceStack   []*font.Face
	paragraph   []content
	vrules      []float64
	indentWidth float64
}

func NewRenderer(proportionalFamily, monospaceFamily *font.Family, headingStyles []BlockStyle, paragraphStyle BlockStyle) *Renderer {
	return &Renderer{
		proportionalFamily: proportionalFamily,
		monospaceFamily:    monospaceFamily,
		headingStyles:      headingStyles,
		paragraphStyle:     paragraphStyle,
	}
}

func (r *Renderer) Render(device bitmap.Device, source []byte, n ast.Node) error {
	return ast.Walk(n, func(n ast.Node, enter bool) (ast.WalkStatus, error) {
		switch n := n.(type) {
		case *ast.Document:
			return r.renderDocument(device, source, n, enter)

		// blocks
		case *ast.Heading:
			return r.renderHeading(device, source, n, enter)
		case *ast.Blockquote:
			return r.renderBlockquote(device, source, n, enter)
		case *ast.CodeBlock:
			return r.renderCodeBlock(device, source, n, enter)
		case *ast.FencedCodeBlock:
			return r.renderFencedCodeBlock(device, source, n, enter)
		case *ast.List:
			return r.renderList(device, source, n, enter)
		case *ast.ListItem:
			return r.renderListItem(device, source, n, enter)
		case *ast.Paragraph:
			return r.renderParagraph(device, source, n, enter)
		case *ast.TextBlock:
			return r.renderTextBlock(device, source, n, enter)
		case *ast.ThematicBreak:
			return r.renderThematicBreak(device, source, n, enter)

		// inlines
		case *ast.AutoLink:
			return r.renderAutoLink(device, source, n, enter)
		case *ast.CodeSpan:
			return r.renderCodeSpan(device, source, n, enter)
		case *ast.Emphasis:
			return r.renderEmphasis(device, source, n, enter)
		case *ast.Image:
			return r.renderImage(device, source, n, enter)
		case *ast.Link:
			return r.renderLink(device, source, n, enter)
		case *ast.Text:
			return r.renderText(device, source, n, enter)
		case *ast.String:
			return r.renderString(device, source, n, enter)
		}

		return ast.WalkContinue, nil
	})
}

func (r *Renderer) pushFace(face *font.Face) {
	r.faceStack = append(r.faceStack, face)
}

func (r *Renderer) popFace() {
	r.faceStack = r.faceStack[:len(r.faceStack)-1]
}

func (r *Renderer) face() *font.Face {
	return r.faceStack[len(r.faceStack)-1]
}

func (r *Renderer) printMargin(device bitmap.Device, points float64) error {
	dpi := device.DPI()
	lines := int(math.Ceil(points / 72.0 * dpi))
	if len(r.vrules) == 0 {
		return device.Feed(lines)
	}

	margin := bitmap.New(image.Rect(0, 0, device.MaxWidth(), lines))
	draw.Draw(margin, margin.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)

	for _, vr := range r.vrules {
		upperLeft := image.Point{floatToFixed(vr / 72.0 * dpi).Ceil(), 0}
		vr := image.Rectangle{upperLeft, upperLeft.Add(image.Point{1, lines})}
		draw.Draw(margin, vr, image.NewUniform(color.Black), image.Point{}, draw.Src)
	}
	return device.PrintBitmap(margin)
}

func (r *Renderer) appendContent(c ...content) {
	if len(r.paragraph) == 0 && r.indentWidth != 0 {
		r.paragraph = append(r.paragraph, indent{vrules: r.vrules, points: r.indentWidth})
	}
	r.paragraph = append(r.paragraph, c...)
}

func (r *Renderer) printParagraph(device bitmap.Device, style BlockStyle, raw bool) error {
	if len(r.paragraph) > 0 {
		if err := r.printMargin(device, style.TopMargin); err != nil {
			return err
		}
		if err := printParagraph(device, r.paragraph, raw); err != nil {
			return err
		}
		if err := r.printMargin(device, style.BottomMargin); err != nil {
			return err
		}
		r.paragraph = nil
	}
	return nil
}

// renderDocument renders an *ast.Document node to the given Device.
func (r *Renderer) renderDocument(device bitmap.Device, source []byte, node *ast.Document, enter bool) (ast.WalkStatus, error) {
	if enter {
		r.listStack, r.faceStack, r.paragraph = nil, nil, nil
	} else {
		if err := r.printMargin(device, 30); err != nil {
			return ast.WalkStop, err
		}
	}
	return ast.WalkContinue, nil
}

// renderHeading renders an *ast.Heading node to the given Device.
func (r *Renderer) renderHeading(device bitmap.Device, source []byte, node *ast.Heading, enter bool) (ast.WalkStatus, error) {
	style := r.paragraphStyle
	if node.Level < len(r.headingStyles) {
		style = r.headingStyles[node.Level]
	}

	if enter {
		// Set the font.
		r.pushFace(r.proportionalFamily.Size(style.PointSize).Bold())
	} else {
		// Print the current paragraph.
		if err := r.printParagraph(device, style, false); err != nil {
			return ast.WalkStop, err
		}
		r.popFace()
	}
	return ast.WalkContinue, nil
}

// renderBlockquote renders an *ast.Blockquote node to the given Device.
func (r *Renderer) renderBlockquote(device bitmap.Device, source []byte, node *ast.Blockquote, enter bool) (ast.WalkStatus, error) {
	if enter {
		r.vrules = append(r.vrules, r.indentWidth)
		r.indentWidth += indentAmount
	} else {
		r.vrules = r.vrules[:len(r.vrules)-1]
		r.indentWidth -= indentAmount
	}
	return ast.WalkContinue, nil
}

func (r *Renderer) renderCode(device bitmap.Device, source []byte, lines *mdtext.Segments) error {
	face := r.monospaceFamily.Size(r.paragraphStyle.PointSize).Regular()
	for i := 0; i < lines.Len(); i++ {
		line := lines.At(i)
		r.appendContent(text{face: face, bytes: line.Value(source)})
	}
	return r.printParagraph(device, r.paragraphStyle, true)
}

// renderCodeBlock renders an *ast.CodeBlock node to the given Device.
func (r *Renderer) renderCodeBlock(device bitmap.Device, source []byte, node *ast.CodeBlock, enter bool) (ast.WalkStatus, error) {
	if enter {
		if err := r.renderCode(device, source, node.Lines()); err != nil {
			return ast.WalkStop, err
		}
	}
	return ast.WalkSkipChildren, nil
}

// renderFencedCodeBlock renders an *ast.FencedCodeBlock node to the given Device.
func (r *Renderer) renderFencedCodeBlock(device bitmap.Device, source []byte, node *ast.FencedCodeBlock, enter bool) (ast.WalkStatus, error) {
	if enter {
		if err := r.renderCode(device, source, node.Lines()); err != nil {
			return ast.WalkStop, err
		}
	}
	return ast.WalkSkipChildren, nil
}

// renderList renders an *ast.List node to the given Device.
func (r *Renderer) renderList(device bitmap.Device, source []byte, node *ast.List, enter bool) (ast.WalkStatus, error) {
	if enter {
		// Measure the maximum marker width.
		face := r.proportionalFamily.Size(r.paragraphStyle.PointSize).Regular()

		var markerWidth fixed.Int26_6
		if node.IsOrdered() {
			for i := 0; i < node.ChildCount(); i++ {
				runes := []rune(fmt.Sprintf("%d.", i+node.Start))
				_, width := measureWord(line{}, []segment{textSegment{face: face, runes: runes}})
				if width > markerWidth {
					markerWidth = width
				}
			}
		} else {
			_, markerWidth = measureWord(line{}, []segment{textSegment{face: face, runes: []rune("•")}})
		}

		r.listStack = append(r.listStack, listState{
			node:        node,
			markerWidth: fixedToFloat(markerWidth) / device.DPI() * 72.0,
			index:       node.Start,
		})
	} else {
		r.listStack = r.listStack[:len(r.listStack)-1]
	}

	return ast.WalkContinue, nil
}

// renderListItem renders an *ast.ListItem node to the given Device.
func (r *Renderer) renderListItem(device bitmap.Device, source []byte, node *ast.ListItem, enter bool) (ast.WalkStatus, error) {
	state := &r.listStack[len(r.listStack)-1]
	if enter {
		// Set the font and write the marker.
		var buf bytes.Buffer
		if state.node.IsOrdered() {
			fmt.Fprintf(&buf, "%d.", state.index)
			state.index++
		} else {
			fmt.Fprintf(&buf, "•")
		}

		face := r.proportionalFamily.Size(r.paragraphStyle.PointSize).Regular()

		r.appendContent(indent{points: r.indentWidth + indentAmount},
			text{face: face, bytes: buf.Bytes()},
			indent{points: r.indentWidth + indentAmount + state.markerWidth + 2.5})

		r.indentWidth += indentAmount + state.markerWidth + 2.5
	} else {
		r.indentWidth -= indentAmount + state.markerWidth + 2.5
	}

	return ast.WalkContinue, nil
}

// renderParagraph renders an *ast.Paragraph node to the given Device.
func (r *Renderer) renderParagraph(device bitmap.Device, source []byte, node *ast.Paragraph, enter bool) (ast.WalkStatus, error) {
	if enter {
		// Set the font.
		r.pushFace(r.proportionalFamily.Size(r.paragraphStyle.PointSize).Regular())
	} else {
		// Print the current paragraph.
		if err := r.printParagraph(device, r.paragraphStyle, false); err != nil {
			return ast.WalkStop, err
		}
		r.popFace()
	}
	return ast.WalkContinue, nil
}

// renderTextBlock renders an *ast.TextBlock node to the given Device.
func (r *Renderer) renderTextBlock(device bitmap.Device, source []byte, node *ast.TextBlock, enter bool) (ast.WalkStatus, error) {
	if enter {
		// Set the font.
		r.pushFace(r.proportionalFamily.Size(r.paragraphStyle.PointSize).Regular())
	} else {
		// Print the current paragraph.
		if err := r.printParagraph(device, r.paragraphStyle, false); err != nil {
			return ast.WalkStop, err
		}
		r.popFace()
	}
	return ast.WalkContinue, nil
}

// renderThematicBreak renders an *ast.ThematicBreak node to the given Device.
func (r *Renderer) renderThematicBreak(device bitmap.Device, source []byte, node *ast.ThematicBreak, enter bool) (ast.WalkStatus, error) {
	// Write a horizontal rule
	if enter {
		margin := int(math.Ceil(r.paragraphStyle.PointSize / 72.0 * device.DPI() / 2))
		img := bitmap.New(image.Rect(0, 0, device.MaxWidth(), margin*2+1))
		draw.Draw(img, img.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
		draw.Draw(img, image.Rect(0, margin, device.MaxWidth(), margin+1), image.NewUniform(color.Black), image.Point{}, draw.Src)
		if err := device.PrintBitmap(img); err != nil {
			return ast.WalkStop, err
		}
	}
	return ast.WalkContinue, nil
}

func (r *Renderer) renderURLAsDataMatrix(device bitmap.Device, url string) (*bitmap.Image, error) {
	// Shorten the URL first.
	url, err := shortenURL(url)
	if err != nil {
		return nil, err
	}

	// Render the shortened URL to a data matrix barcode.
	code, err := datamatrix.Encode(url)
	if err != nil {
		return nil, err
	}

	// Scale the barcode up as necessary for legibility. Doubling the area of each pixel seems to be enough.
	pointSize := r.face().Size()
	pixelHeight := int(math.Ceil(pointSize / 72.0 * device.DPI()))
	if pixelHeight < code.Bounds().Dy()*2 {
		pixelHeight = code.Bounds().Dy() * 2
	}
	code, err = barcode.Scale(code, pixelHeight, pixelHeight)
	if err != nil {
		return nil, err
	}

	// Convert the barcode to a bitmap.
	return bitmap.ForDevice(device, code, false), nil
}

// renderAutoLink renders an *ast.AutoLink node to the given Device.
func (r *Renderer) renderAutoLink(device bitmap.Device, source []byte, node *ast.AutoLink, enter bool) (ast.WalkStatus, error) {
	// data matrix
	return ast.WalkContinue, nil
}

// renderCodeSpan renders an *ast.CodeSpan node to the given Device.
func (r *Renderer) renderCodeSpan(device bitmap.Device, source []byte, node *ast.CodeSpan, enter bool) (ast.WalkStatus, error) {
	if enter {
		// Push the appropriate monospace font face.
		current := r.face()
		r.pushFace(r.monospaceFamily.Face(current.Size(), current.Bold(), current.Italic()))
	} else {
		r.popFace()
	}
	return ast.WalkContinue, nil
}

// renderEmphasis renders an *ast.Emphasis node to the given Device.
func (r *Renderer) renderEmphasis(device bitmap.Device, source []byte, node *ast.Emphasis, enter bool) (ast.WalkStatus, error) {
	if enter {
		// Push the appropriate font face.
		if node.Level >= 2 {
			r.pushFace(r.face().WithBold(true))
		} else {
			r.pushFace(r.face().WithItalic(true))
		}
	} else {
		r.popFace()
	}
	return ast.WalkContinue, nil
}

// renderImage renders an *ast.Image node to the given Device.
func (r *Renderer) renderImage(device bitmap.Device, source []byte, node *ast.Image, enter bool) (ast.WalkStatus, error) {
	// Download the image.
	img, err := downloadImage(string(node.Destination))
	if err != nil {
		// Ignore failures; just print the empty set character.
		r.appendContent(text{
			face:  r.face(),
			bytes: []byte("∅"),
		})
	} else {
		r.appendContent(glyph{
			bits: bitmap.ForDevice(device, img, true),
		})
	}
	return ast.WalkContinue, nil
}

// renderLink renders an *ast.Link node to the given Device.
func (r *Renderer) renderLink(device bitmap.Device, source []byte, node *ast.Link, enter bool) (ast.WalkStatus, error) {
	if !enter {
		dataMatrix, err := r.renderURLAsDataMatrix(device, string(node.Destination))
		if err != nil {
			if _, ok := err.(*url.Error); ok {
				return ast.WalkContinue, nil
			}
			return ast.WalkStop, err
		}
		r.appendContent(glyph{
			bits: dataMatrix,
		})
	}
	return ast.WalkContinue, nil
}

// renderText renders an *ast.Text node to the given Device.
func (r *Renderer) renderText(device bitmap.Device, source []byte, node *ast.Text, enter bool) (ast.WalkStatus, error) {
	if !enter {
		return ast.WalkContinue, nil
	}

	// Append the text to the current paragraph using the current font face.
	r.appendContent(text{
		face:  r.face(),
		bytes: node.Segment.Value(source),
	})

	switch {
	case node.SoftLineBreak():
		r.appendContent(text{
			face:  r.face(),
			bytes: []byte{' '},
		})
	case node.HardLineBreak():
		r.paragraph = append(r.paragraph, linebreak{})
	}

	return ast.WalkContinue, nil
}

// renderString renders an *ast.String node to the given Device.
func (r *Renderer) renderString(device bitmap.Device, source []byte, node *ast.String, enter bool) (ast.WalkStatus, error) {
	if !enter {
		return ast.WalkContinue, nil
	}

	// Append the text to the current paragraph using the current font face.
	r.appendContent(text{
		face:  r.face(),
		bytes: node.Value,
	})

	return ast.WalkContinue, nil
}
