pixfont
=======

A simple, lightweight Pixel Font (aka bitmap fonts) package for Go that works
with the standard `image/draw` package. If you want scaling, anti-aliasing, 
TrueType fonts, or other "fancy" features then I suggest you check out
https://code.google.com/p/freetype-go/

However, if you just want to put a little bit of text in your generated image
and don't care about aliasing, or if you can't afford run-time dependencies or
CGO support, or you needsomething that "just works" without configuration....
Well then `pixfont` has exactly what you want.

Basic, just-put-some-text-in-my-image usage is straightforward:

```go

package main

import (
        "image"
        "image/color"
        "image/png"
        "os"
        
        "github.com/pbnjay/pixfont"
)

func main() {
        img := image.NewRGBA(image.Rect(0, 0, 150, 30))

        pixfont.DrawString(img, 10, 10, "Hello, World!", color.Black)

        f, _ := os.OpenFile("hello.png", os.O_CREATE|os.O_RDWR, 0644)
        png.Encode(f, img)
}

```

`pixfont` comes ready-to-go with a public domain 8x8 pixel fixed-width font from the bygone
era of PCs. Also included is a `fontgen` tool which will help you extract a pixel font from
an image or simple text format and produce Go code that you can compile into your own project.
