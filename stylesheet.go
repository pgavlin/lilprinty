package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/golang/freetype/truetype"
	woff "github.com/tdewolff/canvas/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gobolditalic"
	"golang.org/x/image/font/gofont/goitalic"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/gomonobold"
	"golang.org/x/image/font/gofont/gomonobolditalic"
	"golang.org/x/image/font/gofont/gomonoitalic"
	"golang.org/x/image/font/gofont/goregular"

	"github.com/pgavlin/lilprinty/internal/font"
	"github.com/pgavlin/lilprinty/internal/renderer"
	"github.com/pgavlin/lilprinty/internal/util"
)

type fontFamily struct {
	Regular    string `json:"regular,omitempty"`
	Bold       string `json:"bold,omitempty"`
	Italic     string `json:"italic,omitempty"`
	BoldItalic string `json:"boldItalic,omitempty"`
}

type blockStyle struct {
	PointSize    float64 `json:"pointSize,omitempty"`
	TopMargin    float64 `json:"topMargin,omitempty"`
	BottomMargin float64 `json:"bottomMargin,omitempty"`
}

type styleSheet struct {
	ProportionalFamily *fontFamily  `json:"proportionalFamily,omitempty"`
	MonospaceFamily    *fontFamily  `json:"monospaceFamily,omitempty"`
	HeadingStyles      []blockStyle `json:"headingStyles,omitEmpty"`
	ParagraphStyle     *blockStyle  `json:"paragraphStyle,omitEmpty"`
}

type style struct {
	proportionalFamily *font.Family
	monospaceFamily    *font.Family
	headingStyles      []renderer.BlockStyle
	paragraphStyle     renderer.BlockStyle
}

func mustParseFontFamily(regular, bold, italic, boldItalic []byte, options truetype.Options) *font.Family {
	family, err := font.ParseFamily(regular, bold, italic, boldItalic, options)
	if err != nil {
		panic(fmt.Errorf("error parsing font family: %v", err))
	}
	return family
}

var fontOptions = truetype.Options{DPI: 203.2, SubPixelsX: 1}

var defaultStyle = style{
	proportionalFamily: mustParseFontFamily(goregular.TTF, gobold.TTF, goitalic.TTF, gobolditalic.TTF, fontOptions),
	monospaceFamily:    mustParseFontFamily(gomono.TTF, gomonobold.TTF, gomonoitalic.TTF, gomonobolditalic.TTF, fontOptions),
	headingStyles: []renderer.BlockStyle{
		{PointSize: 16.0, TopMargin: 3.2, BottomMargin: 1.6},
		{PointSize: 14.0, TopMargin: 2.8, BottomMargin: 1.4},
		{PointSize: 12.0, TopMargin: 2.4, BottomMargin: 1.2},
		{PointSize: 10.0, TopMargin: 2.0, BottomMargin: 1.0},
	},
	paragraphStyle: renderer.BlockStyle{PointSize: 8.0, TopMargin: 1.6, BottomMargin: 0.8},
}

func loadTTF(url string) ([]byte, error) {
	bytes, _, err := util.DownloadFile(url)
	if err != nil {
		return nil, err
	}

	return woff.ToSFNT(bytes)
}

func loadFontFamily(family *fontFamily, defaults *font.Family) (*font.Family, error) {
	if family == nil {
		return defaults, nil
	}

	if family.Regular == "" {
		return nil, fmt.Errorf("font family must specify a regular typeface")
	}

	regular, err := loadTTF(family.Regular)
	if err != nil {
		return nil, fmt.Errorf("error loading regular typeface: %v", err)
	}

	bold, italic, boldItalic := regular, regular, regular

	if family.Bold != "" {
		if bold, err = loadTTF(family.Bold); err != nil {
			return nil, fmt.Errorf("error loading bold typeface: %v", err)
		}
	}
	if family.Italic != "" {
		if italic, err = loadTTF(family.Italic); err != nil {
			return nil, fmt.Errorf("error loading italic typeface: %v", err)
		}
	}
	if family.BoldItalic != "" {
		if boldItalic, err = loadTTF(family.BoldItalic); err != nil {
			return nil, fmt.Errorf("error loading boldItalic typeface: %v", err)
		}
	}

	return font.ParseFamily(regular, bold, italic, boldItalic, fontOptions)
}

func loadBlockStyle(style blockStyle) renderer.BlockStyle {
	result := renderer.BlockStyle{
		PointSize:    style.PointSize,
		TopMargin:    style.TopMargin,
		BottomMargin: style.BottomMargin,
	}
	if result.PointSize == 0 {
		result.PointSize = 10.0
	}
	if result.TopMargin == 0 {
		result.TopMargin = result.PointSize * 0.2
	}
	if result.BottomMargin == 0 {
		result.BottomMargin = result.PointSize * 0.1
	}
	return result
}

func loadStylesheet(path string) (style, error) {
	f, err := os.Open(path)
	if err != nil {
		return style{}, err
	}
	defer f.Close()

	var sheet styleSheet
	if err = json.NewDecoder(f).Decode(&sheet); err != nil {
		return style{}, err
	}

	proportionalFamily, err := loadFontFamily(sheet.ProportionalFamily, defaultStyle.proportionalFamily)
	if err != nil {
		return style{}, err
	}

	monospaceFamily, err := loadFontFamily(sheet.MonospaceFamily, defaultStyle.monospaceFamily)
	if err != nil {
		return style{}, err
	}

	headingStyles := defaultStyle.headingStyles
	if len(sheet.HeadingStyles) > 0 {
		headingStyles = make([]renderer.BlockStyle, len(sheet.HeadingStyles))
		for i, s := range sheet.HeadingStyles {
			headingStyles[i] = loadBlockStyle(s)
		}
	}

	paragraphStyle := defaultStyle.paragraphStyle
	if sheet.ParagraphStyle != nil {
		paragraphStyle = loadBlockStyle(*sheet.ParagraphStyle)
	}

	return style{
		proportionalFamily: proportionalFamily,
		monospaceFamily:    monospaceFamily,
		headingStyles:      headingStyles,
		paragraphStyle:     paragraphStyle,
	}, nil
}
