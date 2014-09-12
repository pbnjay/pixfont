package main

import (
	"flag"
	"image"
	"image/color"
	"image/png"
	"os"

	"github.com/pbnjay/pixfont"

	// replace and uncomment to try your own font
	//"path/to/minecraftia"
)

var (
	outfile  = flag.String("o", "example.png", "output PNG filename")
	msg      = flag.String("m", "Hello, World!", "message text to draw")
)

func main() {
	flag.Parse()

	// replace and uncomment to try your own font
	//pixfont.DefaultFont = minecraftia.Font

	// create a PNG just larger than the text
	textWidth := pixfont.MeasureString(*msg)
	img := image.NewRGBA(image.Rect(0, 0, 20+textWidth, 30))

	pixfont.DrawString(img, 10, 10, *msg, color.Black)

	f, _ := os.OpenFile(*outfile, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
	png.Encode(f, img)
}
