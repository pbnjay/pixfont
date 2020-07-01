// fontgen is a commandline tool for generating pixel fonts supported by pixfont.
// First is to create an image of your pixel font in your favorite graphics
// program with your set of supported characters. Ex:
//
//      ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789
//
// Ensure that there is a solid color background, single-color font pixels (i.e.
// no anti-aliasing), and that a column of pixels separate each character to
// ensure best results. Then simply run:
//
//      ./fontgen -img mypixelfont.png -o myfont
//
// Add myfont.go to your project, then just use Font.DrawString(...) to add
// text to your image!
//
package main

import (
	"flag"
	"fmt"
	"go/format"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io/ioutil"
	"os"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/pbnjay/pixfont"
)

var (
	imageName = flag.String("img", "", "image file to extract pixel font from")
	startY    = flag.Int("y", 0, "starting Y position")
	height    = flag.Int("h", 0, "chop height")
	startX    = flag.Int("x", 0, "starting X position")
	width     = flag.Int("w", 0, "chop width")
	alphabet  = flag.String("a", "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789", "alphabet to extract")
	varWidth  = flag.Bool("v", false, "produce variable width font")

	textName = flag.String("txt", "", "text file to extract pixel font from")
	outName  = flag.String("o", "", "package name to create (becomes <myfont>.go)")
)

func generatePixFont(name string, w, h int, v bool, d map[rune]map[int]string) {
	template := `
		package %s

		import "github.com/pbnjay/pixfont"

		var Font *pixfont.PixFont

		func init() {
			charMap := %#v
			data := %#v
			Font = pixfont.NewPixFont(%d, %d, charMap, data)
			Font.SetVariableWidth(%t)
		}
	`
	cm := make(map[rune]uint16)

	// Sort the glyph list so the representation is stable across different invocations
	// of fontgen.
	chs := make([]int, 0, len(d))
	for ch, _ := range d {
		chs = append(chs, int(ch))
	}
	sort.IntSlice(chs).Sort()

	// convert from simple character encoding to packed bitfield
	// NB fonts should be at most 32 pixels wide to fit in the uint32
	//    (height is limited to uint8 255)
	//
	// This packed representation stores 1-4 glyphs in a single uint32 (per line).
	// Each uint32 is trested as four individually-addressable uint8s, and each uint8
	// can be chosen as the start of a glyph line. Glyphs are stored with their
	// leftmost pixels in the LSB of a W-bit wide region aligned to the nearest 8 bits.
	// Glyphs that will not fit are bumped to the next u32.
	//
	// That is to say, a 4-character font with an 8-pixel width will be stored
	// one u32 per line, and a 4-character font with a 9-pixel width will be
	// stored 2 u32 per line (since you cannot fit >2 9-bit integers in a single
	// u32 and maintain the 8-bit alignment guarantee)
	//
	// Lines are stored in consecutive uint32s.
	//
	//         24      16       8       0
	//          |       |       |       |
	// 0     DDDD    CCC     BBBB     A   == 0b00001111000011100000111100000100 == 0xf0e0f04
	// 1    D   D   C   C   B   B    A A  == 0b00010001000100010001000100001010 == 0x1111110a
	// 2    D   D       C    BBBB   A   A == 0b00010001000000010000111100010001 == 0x11010f11
	// 3    D   D   C   C   B   B   AAAAA == 0b00010001000100010001000100011111 == 0x1111111f
	// 4     DDDD    CCC     BBBB   A   A == 0b00001111000011100000111100010001 == 0xf0e0f11
	// 5                            EEEEE == 0b00000000000000000000000000011111 == 0x1f
	// 6                                E == 0b00000000000000000000000000000001 == 0x1
	// 7                             EEEE == 0b00000000000000000000000000001111 == 0xf
	// 8                                E == 0b00000000000000000000000000000001 == 0x1
	// 9                            EEEEE == 0b00000000000000000000000000011111 == 0x1f

	u8PerCh := uint16((w-1)>>3) + 1 // 0-8 take up 1 byte, 9-16 take up 2, 17-24 take up 3, 24+ take up 4
	chPerU32 := 4 / u8PerCh         // we can fit 4, 2 or 1 glyphs per u32
	spacing := 4 / chPerU32         // we must skip 1, 2, or 4 8-bit units between each glyph start

	costPerLine := (len(d) + int(chPerU32/2)) / int(chPerU32) // #u32 per horizontal line in font
	costTotal := h * costPerLine                              // #u32s required for the whole font

	encoded := make([]uint32, costTotal)

	// i8 tracks the number of 8-bit units we've skipped
	var i8 uint16
	for _, c := range chs {
		matrix := d[rune(c)]

		i32 := int(i8>>2) * h  // i32 is the index into encoded for the u32 for this char
		dist := int(i8 & 0b11) // how many u8 units into the u32 we're offset
		cm[rune(c)] = uint16((i32 << 2) | dist)

		for y := 0; y < h; y++ {
			line := encoded[i32+y]
			var b uint32 = 1 << uint(8*dist)

			if ld, hasLine := matrix[y]; hasLine {
				for x := 0; x < w; x++ {
					if len(ld) > x && ld[x] == 'X' {
						line |= b
					}
					b <<= 1
				}
			}

			encoded[i32+y] = line
		}

		i8 += spacing
	}

	fnt := pixfont.NewPixFont(uint8(w), uint8(h), cm, encoded)
	fnt.SetVariableWidth(v)

	f, err := os.OpenFile(name+".go", os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return
	}

	// draw a comment header using the new font
	sd := &pixfont.StringDrawable{}
	fnt.DrawString(sd, 0, 0, name, nil)
	fmt.Fprintln(f, sd.PrefixString("// "))

	// create the code from the template and go fmt it
	code := fmt.Sprintf(template, name, cm, encoded, w, h, v)
	bcode, _ := format.Source([]byte(code))
	fmt.Fprintln(f, string(bcode))

	f.Close()
}

func processImage(filename string) (allLetters map[rune]map[int]string, maxWidth int) {
	f, err := os.Open(filename)
	if err != nil {
		fmt.Fprint(os.Stderr, err.Error())
		return nil, 0
	}
	img, _, err := image.Decode(f)
	if err != nil {
		fmt.Fprint(os.Stderr, err.Error())
		return nil, 0
	}
	if *width == 0 {
		*width = img.Bounds().Dx() - *startX
	}
	if *height == 0 {
		*height = img.Bounds().Dy() - *startY
	}
	allLetters = make(map[rune]map[int]string)
	maxWidth = 0

	// generate a greyscale histogram of the image
	pxc := 0
	clrs := make(map[uint8]int)
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			c := img.At(x, y)
			gc := color.GrayModel.Convert(c).(color.Gray)
			clrs[gc.Y]++
			pxc++
		}
	}
	// find a threshold pixel count for what colors to ignore as background
	// (ie assumes background image is fairly solid and colors occur much
	//  more often than font colors)
	pxt := pxc
	pxd := 0
	for pxd < (pxc/2) && pxt > 0 {
		pxt /= 2
		pxd = 0
		for _, n := range clrs {
			if n > pxt {
				pxd += n
			}
		}
	}

	// scan across the image in the crop region, saving pixels as you go.
	// if at any point we see an "empty" column of pixels, we assume it
	// is a character boundary and move to the next alphabet letter.
	curAlpha := *alphabet
	curWidth := 0
	curLetter := make(map[int]string)
	for x := *startX; x < *startX+*width; x++ {
		curWidth++
		isEmpty := true
		ay := 0
		for y := *startY; y < *startY+*height; y++ {
			c := img.At(x, y)
			gc := color.GrayModel.Convert(c).(color.Gray)
			if clrs[gc.Y] <= pxt {
				if _, haveDots := curLetter[ay]; !haveDots {
					curLetter[ay] = strings.Repeat(" ", curWidth-1)
				}
				curLetter[ay] += "X"
				isEmpty = false
			} else {
				if _, haveDots := curLetter[ay]; haveDots {
					curLetter[ay] += " "
				}
			}
			ay++
		}

		if isEmpty {
			if len(curLetter) != 0 {
				if len(curAlpha) > 0 {
					curWidth-- // remove last blank column
					for yy, ln := range curLetter {
						if len(ln) >= curWidth {
							curLetter[yy] = ln[:curWidth]
						}
					}
					r, nbytes := utf8.DecodeRuneInString(curAlpha)
					allLetters[r] = curLetter
					curAlpha = curAlpha[nbytes:]
				}
				if curWidth > maxWidth {
					maxWidth = curWidth
				}
			}
			curWidth = 0
			curLetter = make(map[int]string)
		}
	}

	if *outName != "" {
		return
	}

	// output a simple text representation of the extracted characters
	for _, a := range *alphabet {
		if l, found := allLetters[a]; found {
			w := 0
			for yy := 0; yy < *height; yy++ {
				if len(l[yy]) > w {
					w = len(l[yy])
				}
			}

			leftPad := (maxWidth - w) / 2
			if *varWidth {
				leftPad = 0
			}
			for yy := 0; yy < *height; yy++ {
				l[yy] = strings.Repeat(" ", leftPad) + l[yy]
				fmt.Printf("%c  [%*s]\n", a, -maxWidth, l[yy])
			}
		}
	}
	return
}

func processText(filename string) (allLetters map[rune]map[int]string, maxWidth int) {
	newalpha := ""
	input, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return nil, 0
	}
	allLetters = make(map[rune]map[int]string)
	re := regexp.MustCompile(`[^\n]*\n`)
	count := 0
	hh, maxHeight := 0, 0
	lastCh := rune(0)

	for _, bline := range re.FindAll(input, -1) {
		line := string(bline)
		c, pixoffs := utf8.DecodeRuneInString(line)
		pixoffs += 3
		if lastCh != c {
			count = 0
			hh = len(allLetters[lastCh])
			if hh > maxHeight {
				maxHeight = hh
			}
			allLetters[c] = make(map[int]string)
			newalpha += string(c)
		}
		ww := len(line) - (pixoffs + 2)
		if ww > maxWidth {
			maxWidth = ww
		}
		allLetters[c][count] = line[pixoffs : pixoffs+ww]
		lastCh = c
		count++
	}

	*alphabet = newalpha
	if *width == 0 {
		*width = maxWidth
	}
	if *height == 0 {
		*height = maxHeight
	}

	if *outName != "" {
		return
	}

	// output the same representation again, to allow user to verify it was parsed correctly
	for _, a := range *alphabet {
		if l, found := allLetters[a]; found {
			w := 0
			for yy := 0; yy < *height; yy++ {
				if len(l[yy]) > w {
					w = len(l[yy])
				}
			}

			leftPad := (maxWidth - w) / 2
			for yy := 0; yy < *height; yy++ {
				l[yy] = strings.Repeat(" ", leftPad) + l[yy]
				fmt.Printf("%c  [%*s]\n", a, -maxWidth, l[yy])
			}
		}
	}
	return
}

func main() {
	flag.Parse()

	allLetters := make(map[rune]map[int]string)
	maxWidth := 0

	if *imageName != "" {
		allLetters, maxWidth = processImage(*imageName)
	} else if *textName != "" {
		allLetters, maxWidth = processText(*textName)
	} else {
		fmt.Fprintln(os.Stderr, "-img or -txt should be provided")
		flag.Usage()
		return
	}

	if *outName != "" {
		generatePixFont(*outName, maxWidth, *height, *varWidth, allLetters)
		fmt.Fprintln(os.Stderr, "Created package file:", *outName+".go")
	}
}
