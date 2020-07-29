package font

import "github.com/golang/freetype/truetype"

type FaceFamily struct {
	family    *Family
	pointSize float64

	regularFace    *Face
	boldFace       *Face
	italicFace     *Face
	boldItalicFace *Face
}

func (ff *FaceFamily) Family() *Family {
	return ff.family
}

func (ff *FaceFamily) Size() float64 {
	return ff.pointSize
}

func (ff *FaceFamily) WithSize(pointSize float64) *FaceFamily {
	return ff.family.Size(pointSize)
}

func (ff *FaceFamily) Face(bold, italic bool) *Face {
	switch {
	case bold && !italic:
		return ff.Bold()
	case !bold && italic:
		return ff.Italic()
	case bold && italic:
		return ff.BoldItalic()
	}
	return ff.Regular()
}

func (ff *FaceFamily) Regular() *Face {
	if ff.regularFace == nil {
		opts := ff.family.options
		opts.Size = ff.pointSize
		ff.regularFace = &Face{
			Face:       truetype.NewFace(ff.family.regularFont, &opts),
			faceFamily: ff,
		}
	}
	return ff.regularFace
}

func (ff *FaceFamily) Bold() *Face {
	if ff.boldFace == nil {
		opts := ff.family.options
		opts.Size = ff.pointSize
		ff.boldFace = &Face{
			Face:       truetype.NewFace(ff.family.boldFont, &opts),
			faceFamily: ff,
			bold:       true,
		}
	}
	return ff.boldFace
}

func (ff *FaceFamily) Italic() *Face {
	if ff.italicFace == nil {
		opts := ff.family.options
		opts.Size = ff.pointSize
		ff.italicFace = &Face{
			Face:       truetype.NewFace(ff.family.italicFont, &opts),
			faceFamily: ff,
			italic:     true,
		}
	}
	return ff.italicFace
}

func (ff *FaceFamily) BoldItalic() *Face {
	if ff.boldItalicFace == nil {
		opts := ff.family.options
		opts.Size = ff.pointSize
		ff.boldItalicFace = &Face{
			Face:       truetype.NewFace(ff.family.boldItalicFont, &opts),
			faceFamily: ff,
			bold:       true,
			italic:     true,
		}
	}
	return ff.boldItalicFace
}
