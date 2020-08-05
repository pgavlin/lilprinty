package markdown

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"unicode"
	"unicode/utf8"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"

	"github.com/pgavlin/lilprinty/internal/bitmap"
)

// 1pt = 1/72nd of an inch

type content interface {
	isContent()
}

type text struct {
	face  font.Face
	bytes []byte
}

func (text) isContent() {}

type glyph struct {
	bits                    *bitmap.Image
	leftMargin, rightMargin float64
}

func (glyph) isContent() {}

type linebreak struct{}

func (linebreak) isContent() {}

type indent struct {
	vrules []float64
	points float64
}

func (indent) isContent() {}

type segment interface {
	isSegment()
}

type textSegment struct {
	face  font.Face
	runes []rune
}

func (textSegment) isSegment() {}

type glyphSegment struct {
	bits                    *bitmap.Image
	leftMargin, rightMargin fixed.Int26_6
}

func (glyphSegment) isSegment() {}

type indentSegment struct {
	vrules []fixed.Int26_6
	width  fixed.Int26_6
}

func (indentSegment) isSegment() {}

type line struct {
	segments []segment
}

func measureWord(l line, word []segment) (fixed.Int26_6, fixed.Int26_6) {
	var firstKern, wordWidth fixed.Int26_6
	prevC, prevMargin := rune(-1), fixed.I(0)
	if len(l.segments) > 0 {
		lastSegment := l.segments[len(l.segments)-1]
		switch s := lastSegment.(type) {
		case textSegment:
			prevC = s.runes[len(s.runes)-1]
		case glyphSegment:
			prevMargin = s.rightMargin
		}
	}
	for i, s := range word {
		switch s := s.(type) {
		case textSegment:
			for _, c := range s.runes {
				if prevC >= 0 {
					kernWidth := s.face.Kern(prevC, c)
					if i == 0 {
						firstKern = kernWidth
					}
					wordWidth += kernWidth
				} else if prevMargin != 0 {
					wordWidth += prevMargin
				}
				a, ok := s.face.GlyphAdvance(c)
				if ok {
					wordWidth += a
					prevC, prevMargin = c, 0
				}
			}
		case glyphSegment:
			if prevC >= 0 {
				wordWidth += s.leftMargin
			} else if prevMargin != 0 {
				wordWidth += prevMargin + s.leftMargin
			}
			wordWidth += fixed.I(s.bits.Bounds().Dx())
			prevC, prevMargin = -1, s.rightMargin
		default:
			panic(fmt.Errorf("unexpected segment in word: %T", s))
		}
	}
	return firstKern, wordWidth
}

func layoutParagraph(outputWidth fixed.Int26_6, outputDPI float64, contents []content, raw bool) []line {
	var lines []line

	var l line
	var word []segment
	var lineWidth, indentWidth fixed.Int26_6

	appendWord := func() {
		if len(word) == 0 {
			return
		}

		// Measure the word.
		firstKern, wordWidth := measureWord(l, word)

		switch {
		case lineWidth+wordWidth < outputWidth:
			// The word fits on the current line.
			l.segments, lineWidth = append(l.segments, word...), lineWidth+wordWidth
		// case wordWidth-firstKern > outputWidth:
		// TODO: The word doesn't fit on a line at all. Hyphenate it.
		default:
			// Start a new line with the current word.
			lines = append(lines, l)
			l, lineWidth = line{segments: word}, indentWidth+wordWidth-firstKern
		}

		word = nil
	}

	for _, t := range contents {
		switch t := t.(type) {
		case text:
			var runes []rune
			for b := t.bytes; len(b) > 0; {
				r, sz := utf8.DecodeRune(b)
				b = b[sz:]

				if raw {
					if r == '\n' {
						word = append(word, textSegment{face: t.face, runes: runes})
						l.segments = append(l.segments, word...)
						word = nil

						lines = append(lines, l)
						l, lineWidth = line{}, indentWidth
					} else {
						runes = append(runes, r)
						if len(b) == 0 {
							word = append(word, textSegment{face: t.face, runes: runes})
						}
					}
				} else {
					isSpace := unicode.IsSpace(r)
					if isSpace {
						r = ' '
					}
					runes = append(runes, r)

					// If this is a space character or the end of the contents, process the current word.
					if isSpace || len(b) == 0 {
						// Add a segment to the word.
						word = append(word, textSegment{
							face:  t.face,
							runes: runes,
						})
						runes = nil
					}
					if isSpace {
						appendWord()
					}
				}
			}
		case glyph:
			// Convert the margins to pixels.
			leftMargin := fixed.I(int(math.Ceil(t.leftMargin / 72.0 * outputDPI)))
			rightMargin := fixed.I(int(math.Ceil(t.rightMargin / 72.0 * outputDPI)))

			word = append(word, glyphSegment{
				bits:        t.bits,
				leftMargin:  leftMargin,
				rightMargin: rightMargin,
			})
		case linebreak:
			lines = append(lines, l)
			l, lineWidth = line{}, indentWidth
		case indent:
			appendWord()

			indentWidth = fixed.I(int(math.Ceil(t.points / 72.0 * outputDPI)))

			vrules := make([]fixed.Int26_6, len(t.vrules))
			for i, vr := range t.vrules {
				vrules[i] = fixed.I(int(math.Ceil(vr / 72.0 * outputDPI)))
			}

			l.segments = append(l.segments, indentSegment{vrules: vrules, width: indentWidth})
			lineWidth = indentWidth
		}
	}
	if len(word) != 0 {
		appendWord()
	}
	if len(l.segments) > 0 {
		lines = append(lines, l)
	}
	return lines
}

func printParagraph(output bitmap.Device, contents []content, raw bool) error {
	outputWidth := fixed.I(output.MaxWidth())

	// Layout the paragraph.
	lines := layoutParagraph(outputWidth, output.DPI(), contents, raw)

	// Render each line to a bitmap.
	var indentWidth fixed.Int26_6
	var vrules []fixed.Int26_6
	for _, l := range lines {
		// Calculate the line height.
		var lineHeight fixed.Int26_6
		for _, s := range l.segments {
			switch s := s.(type) {
			case textSegment:
				metrics := s.face.Metrics()
				if metrics.Ascent+metrics.Descent > lineHeight {
					lineHeight = metrics.Ascent + metrics.Descent
				}
			case glyphSegment:
				height := fixed.I(s.bits.Bounds().Dy())
				if height > lineHeight {
					lineHeight = height
				}
			}
		}

		// Create an image for the line.
		img := bitmap.NewThreshold(image.Rect(0, 0, output.MaxWidth(), lineHeight.Ceil()), 140)
		draw.Draw(img, img.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
		src := image.NewUniform(color.Black)
		dot := fixed.P(indentWidth.Ceil(), 0)

		for _, vr := range vrules {
			upperLeft := image.Point{vr.Ceil(), dot.Y.Ceil()}
			vr := image.Rectangle{upperLeft, upperLeft.Add(image.Point{1, lineHeight.Ceil()})}
			draw.Draw(img, vr, src, image.Point{}, draw.Src)
		}

		// Render the line into the image.
		prevC, prevMargin := rune(-1), fixed.I(0)
		for _, s := range l.segments {
			switch s := s.(type) {
			case textSegment:
				metrics := s.face.Metrics()
				dot.Y = lineHeight - metrics.Descent

				for _, c := range s.runes {
					if prevC >= 0 {
						dot.X += s.face.Kern(prevC, c)
					} else if prevMargin != 0 {
						dot.X += prevMargin
					}
					dr, mask, maskp, advance, ok := s.face.Glyph(dot, c)
					if !ok {
						continue
					}
					draw.DrawMask(img, dr, src, image.Point{}, mask, maskp, draw.Over)
					dot.X += advance
					prevC, prevMargin = c, 0
				}
			case glyphSegment:
				if prevC >= 0 {
					dot.X += s.leftMargin
				} else if prevMargin != 0 {
					dot.X += prevMargin + s.leftMargin
				}

				bounds := s.bits.Bounds()
				upperLeft := image.Point{
					X: dot.X.Ceil(),
					Y: lineHeight.Ceil()/2 - bounds.Dy()/2,
				}

				dr := image.Rectangle{upperLeft, upperLeft.Add(bounds.Size())}
				draw.Draw(img, dr, s.bits, image.Point{}, draw.Src)
				dot.X += fixed.I(bounds.Dx())
				prevC, prevMargin = -1, s.rightMargin
			case indentSegment:
				for _, vr := range s.vrules {
					upperLeft := image.Point{vr.Ceil(), dot.Y.Ceil()}
					vr := image.Rectangle{upperLeft, upperLeft.Add(image.Point{1, lineHeight.Ceil()})}
					draw.Draw(img, vr, src, image.Point{}, draw.Src)
				}
				indentWidth, dot.X, vrules = s.width, s.width, s.vrules
				prevC, prevMargin = -1, 0
			}
		}

		// Print the image.
		if err := output.PrintBitmap(img); err != nil {
			return err
		}
	}

	return nil
}
