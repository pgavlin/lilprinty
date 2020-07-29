package font

import "golang.org/x/image/font"

type Face struct {
	font.Face

	faceFamily   *FaceFamily
	bold, italic bool
}

func (f *Face) Family() *Family {
	return f.faceFamily.family
}

func (f *Face) FaceFamily() *FaceFamily {
	return f.faceFamily
}

func (f *Face) Size() float64 {
	return f.faceFamily.Size()
}

func (f *Face) Bold() bool {
	return f.bold
}

func (f *Face) Italic() bool {
	return f.italic
}

func (f *Face) WithSize(pointSize float64) *Face {
	return f.Family().Face(pointSize, f.bold, f.italic)
}

func (f *Face) WithBold(bold bool) *Face {
	return f.faceFamily.Face(bold, f.italic)
}

func (f *Face) WithItalic(italic bool) *Face {
	return f.faceFamily.Face(f.bold, italic)
}
