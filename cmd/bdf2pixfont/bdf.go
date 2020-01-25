package main

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

/// https://www.adobe.com/content/dam/acom/en/devnet/font/pdfs/5005.BDF_Spec.pdf

// BDFontChar represents a single glyph in the BDF font definition.
type BDFontChar struct {
	Name     string // "SPACE"
	Encoding rune   // 32
	Width    int    // pixels, e.g. 5

	BoundingBox [4]int   // Width, Height, X offset, Y offset
	Bitmap      []uint32 // [Height]
}

func (x *BDFontChar) String() string {
	// width, height, x-offset, y-offset
	xpad := strings.Repeat(" ", x.BoundingBox[2])
	rpad := strings.Repeat(" ", x.Width-(x.BoundingBox[0]+x.BoundingBox[2]))

	s := []string{}
	if x.BoundingBox[3] > 0 {
		for y := 0; y < x.BoundingBox[3]; y++ {
			s = append(s, fmt.Sprintf("%c  [%s]", x.Encoding, xpad+strings.Repeat(" ", x.BoundingBox[0])+rpad))
		}
	}

	for _, b := range x.Bitmap {
		raster := fmt.Sprintf("%032b", b)
		o := 32 - ((((x.BoundingBox[0] - 1) / 8) + 1) * 8)
		raster = raster[o : o+x.BoundingBox[0]]
		raster = strings.ReplaceAll(raster, "0", " ")
		raster = strings.ReplaceAll(raster, "1", "X")
		s = append(s, fmt.Sprintf("%c  [%s]", x.Encoding, xpad+raster+rpad))
	}
	return strings.Join(s, "\n")
}

// BDFont represents a set of glyphs in the BDF font definition.
type BDFont struct {
	Version  string // "STARTFONT 2.1"
	Comments string
	FontName string

	PointSize   int // font point size e.g. 8
	ResolutionX int // display resolution e.g. 72
	ResolutionY int

	BoundingBox [4]int // Width, Height, X offset, Y offset

	NumProperties int
	Properties    map[string]string

	NumGlyphs int
	Glyphs    map[rune]*BDFontChar
}

func OpenBDF(f io.Reader) (*BDFont, error) {
	fnt := &BDFont{}
	var err error

	s := bufio.NewScanner(f)
	for s.Scan() {
		parts := strings.SplitN(s.Text(), " ", 2)
		if len(parts) == 1 {
			parts = append(parts, "")
		}
		if fnt.NumProperties != len(fnt.Properties) {
			fnt.Properties[parts[0]] = strings.Trim(parts[1], `"`)
			continue
		}
		pfunc, ok := parsers[parts[0]]
		if ok {
			pfunc(fnt, parts[1])
		}

		if fnt.NumGlyphs > 0 {
			break
		}
	}

	fnt.Glyphs = make(map[rune]*BDFontChar, fnt.NumGlyphs)
	s.Scan()
	for i := 0; i < fnt.NumGlyphs; i++ {
		line := s.Text()
		for !strings.HasPrefix(line, "STARTCHAR") {
			s.Scan()
			line = s.Text()
		}

		ch := &BDFontChar{
			Name: strings.TrimPrefix(line, "STARTCHAR "),
		}

		//log.Println(ch.Name)

		s.Scan()
		parts := strings.SplitN(s.Text(), " ", 2)
		for parts[0] != "BITMAP" {
			if len(parts) == 1 {
				parts = append(parts, "")
			}
			if cfunc, ok := charparsers[parts[0]]; ok {
				cfunc(ch, parts[1])
			}

			s.Scan()
			parts = strings.SplitN(s.Text(), " ", 2)
		}

		// bounding box offsets are from left-baseline origin
		// so we move them to the top left by adding ascent
		//    e.g. char-W char-H char-X char-Y
		// becomes: font-W font-H left-padding top-padding
		if ch.BoundingBox[2] < 0 {
			// can't support negative X offsets
			ch.BoundingBox[2] = 0
		}

		// NB DESCENT and OFFSET are negative for character that extend below the baseline.
		// so ASCENT = (FONT_HEIGHT + DESCENT) [e.g. 24 + -4 = 20] for ascent=20, descent=4
		// font starts at baseline, y = ascent, so the first bitmap pixel is computed to:
		//   ascent - y_offset - height, or 20 - (-4) - 22 = 2
		// for a 22px tall bitmap that has 2 pixels blank but extends below the baseline
		//
		// ((FONT_HEIGHT + DESCENT) - Y_OFFSET) - BITMAP_HEIGHT
		ch.BoundingBox[3] = ((fnt.BoundingBox[1] + fnt.BoundingBox[3]) - ch.BoundingBox[3]) - ch.BoundingBox[1]

		ch.Bitmap = make([]uint32, ch.BoundingBox[1])
		for h := 0; h < ch.BoundingBox[1]; h++ {
			s.Scan()
			fmt.Sscanf(s.Text(), "%X", &ch.Bitmap[h])
		}

		fnt.Glyphs[ch.Encoding] = ch
	}

	return fnt, err
}

////////

var charparsers = map[string]func(*BDFontChar, string){
	"ENCODING": func(f *BDFontChar, line string) {
		nc := 0
		fmt.Sscanf(line, "%d", &nc)
		//log.Println("ENC ", line, nc)
		f.Encoding = rune(nc)
	},
	"DWIDTH": func(f *BDFontChar, line string) {
		fmt.Sscanf(line, "%d 0", &f.Width)
	},
	"BBX": func(f *BDFontChar, line string) {
		// width, height, x-offset, y-offset
		fmt.Sscanf(line, "%d %d %d %d", &f.BoundingBox[0], &f.BoundingBox[1], &f.BoundingBox[2], &f.BoundingBox[3])
	},
}

var parsers = map[string]func(*BDFont, string){
	"STARTFONT": func(f *BDFont, line string) {
		f.Version = line
	},
	"COMMENT": func(f *BDFont, line string) {
		f.Comments += line + "\n"
	},
	"FONT": func(f *BDFont, line string) {
		f.FontName = line
	},
	"SIZE": func(f *BDFont, line string) {
		fmt.Sscanf(line, "%d %d %d", &f.PointSize, &f.ResolutionX, &f.ResolutionY)
	},
	"FONTBOUNDINGBOX": func(f *BDFont, line string) {
		fmt.Sscanf(line, "%d %d %d %d", &f.BoundingBox[0], &f.BoundingBox[1], &f.BoundingBox[2], &f.BoundingBox[3])
	},
	"STARTPROPERTIES": func(f *BDFont, line string) {
		fmt.Sscanf(line, "%d", &f.NumProperties)
		f.Properties = make(map[string]string, f.NumProperties)
	},
	"ENDPROPERTIES": func(f *BDFont, line string) {
		f.NumProperties = len(f.Properties)
	},

	"CHARS": func(f *BDFont, line string) {
		fmt.Sscanf(line, "%d", &f.NumGlyphs)
		//log.Println("STARTING CHARS: ", line, f.NumGlyphs)
	},
}
