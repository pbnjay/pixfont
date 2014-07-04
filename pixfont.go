// Package pixfont is a simple pixel font library that allows for text drawing in the
// standard image and image/draw packages. It is very useful when heavyweight TrueType
// fonts and the freetype-go package are not available or desired. Since fonts are
// simple bitmap images, scaling is not possible and aliasing will occur.
//
// See the included fontgen tool if you wish to convert or include your own pixel font
// in your project.
package pixfont

import (
	"bytes"
	"image/color"
)

// DefaultFont is used by the convienence method DrawString, and is initialized
// to the Public Domain 8x8 fixed font with some unicode characters.
var DefaultFont = Font8x8

// Drawable is an interface which supports setting an x,y coordinate to a color.
type Drawable interface {
	Set(x, y int, c color.Color)
}

// PixFont represents a simple bitmap or pixel-based font that can be drawn using
// simple opaque-pixel operations (supported by image.Image and easily included
// in other packages).
type PixFont struct {
	charWidth  uint8
	charHeight uint8
	charmap    map[rune]uint16
	data       []uint32
}

// NewPixFont creates a new PixFont with the provided character width/height and
// character map of offsets into a packed uint32 array of bits.
func NewPixFont(w, h uint8, cm map[rune]uint16, d []uint32) *PixFont {
	return &PixFont{w, h, cm, d}
}

// DrawRune uses this PixFont to display a single rune in the provided color and
// position in Drawable. The x,y position represents the top-left corner of the rune.
// Drawable.Set is called for each opaque pixel in the font, leaving all other pixels
// in the Drawable as-is. If the rune has no representation in the PixFont, then
// DrawRune returns false and no drawing is done.
func (p *PixFont) DrawRune(dr Drawable, x, y int, c rune, clr color.Color) bool {
	poff, haveChar := p.charmap[c]
	if !haveChar {
		return false
	}
	pindex := int(poff >> 2)
	psub := (poff & 0x03) * 8
	d := p.data[pindex : pindex+int(p.charHeight)]
	for yy := 0; yy < int(p.charHeight); yy++ {
		bitMask := uint32(1) << psub
		for xx := 0; xx < int(p.charWidth); xx++ {
			if (d[yy] & bitMask) != 0 {
				dr.Set(x+xx, y+yy, clr)
			}
			bitMask <<= 1
		}
	}
	return true
}

// DrawString uses this PixFont to display text in the provided color and the specified
// start position in Drawable. The x,y position represents the top-left corner of the
// first letter of s. Text is drawn by repeated calls to DrawRune for each character.
func (p *PixFont) DrawString(dr Drawable, x, y int, s string, clr color.Color) {
	for _, c := range s {
		p.DrawRune(dr, x, y, c, clr)
		x += int(p.charWidth) + 1
	}
}

// DrawString is a convienence method that calls DrawString using the DefaultFont
func DrawString(dr Drawable, x, y int, s string, clr color.Color) {
	DefaultFont.DrawString(dr, x, y, s, clr)
}

///////

// StringDrawable implements Drawable so you can do FIGlet-inspired pixel fonts in
// text. Obviously it's much simpler though.
type StringDrawable struct {
	lines [][]byte
}

func (s *StringDrawable) Set(x, y int, c color.Color) {
	for len(s.lines) <= y {
		s.lines = append(s.lines, make([]byte, x))
	}

	if len(s.lines[y]) <= x {
		nb := make([]byte, 1+(x-len(s.lines[y])))
		s.lines[y] = append(s.lines[y], nb...)
	}

	s.lines[y][x] = byte('X')
}

// String returns the current string representation of this Drawable.
func (s *StringDrawable) String() string {
	return s.PrefixString("")
}

// PrefixString returns the current string representation of this Drawable with a
// user-provided prefix before each line. Useful for adding output in code comments.
func (s *StringDrawable) PrefixString(p string) string {
	r := ""
	for _, line := range s.lines {
		r += p + string(bytes.Replace(line, []byte{0}, []byte(" "), -1)) + "\n"
	}
	return r
}
