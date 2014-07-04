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
//      ./fontget -img mypixelfont.png -o myfont
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

	textName = flag.String("txt", "", "text file to extract pixel font from")
	outName  = flag.String("o", "", "package name to create (becomes <myfont>.go)")
)

func generatePixFont(name string, w, h int, d map[rune]map[int]string) {
	template := `
		package %s

		import "bitbucket.org/pbnjay/pixfont"

		var Font *pixfont.PixFont

		func init() {
			charMap := %#v
			data := %#v
			Font = pixfont.NewPixFont(%d, %d, charMap, data)
		}
	`
	encoded := []uint32{}
	cm := make(map[rune]uint16)

	// convert from simple character encoding to packed bitfield
	// NB fonts should be at most 32 pixels wide to fit in the uint32
	//    (height is limited to uint8 255)
	idx := 0
	sub := -1 - (w / 8)
	for c, matrix := range d {
		sub += 1 + (w / 8)
		if sub > 3 {
			sub = 0
			idx = len(encoded)
		}

		cm[c] = uint16(sub) | (uint16(idx) << 2)
		for y := 0; y < h; y++ {
			var line uint32
			var b uint32 = 1
			if sub != 0 {
				line = encoded[idx+y]
				b <<= uint(8 * sub)
			}

			if ld, hasLine := matrix[y]; hasLine {
				for x := 0; x < w; x++ {
					if len(ld) > x && ld[x] == 'X' {
						line |= b
					}
					b <<= 1
				}
			}

			if sub == 0 {
				encoded = append(encoded, line)
			} else {
				encoded[idx+y] = line
			}
		}
	}

	fnt := pixfont.NewPixFont(uint8(w), uint8(h), cm, encoded)

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
	code := fmt.Sprintf(template, name, cm, encoded, w, h)
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
	curAlpha := 0
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
				if curAlpha < len(*alphabet) {
					curWidth-- // remove last blank column
					for yy, ln := range curLetter {
						if len(ln) >= curWidth {
							curLetter[yy] = ln[:curWidth]
						}
					}
					allLetters[rune((*alphabet)[curAlpha])] = curLetter
				}
				if curWidth > maxWidth {
					maxWidth = curWidth
				}
				curAlpha++
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
	hh := 0
	lastCh := rune(0)

	for _, bline := range re.FindAll(input, -1) {
		line := string(bline)
		c, pixoffs := utf8.DecodeRuneInString(line)
		pixoffs += 3
		if lastCh != c {
			count = 0
			hh = len(allLetters[lastCh])
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
	*height = hh

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
		generatePixFont(*outName, maxWidth, *height, allLetters)
		fmt.Fprintln(os.Stderr, "Created package file:", *outName+".go")
	}
}
