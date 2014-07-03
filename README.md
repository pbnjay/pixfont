pixfont
=======

A simple, lightweight Pixel Font package for Go that works with the standard `image/draw` package.

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
        img := image.NewRGBA(image.Rect(0, 0, 100, 100))
        pixfont.F.DrawString(img, 10, 10, "Hello World!", color.Black)

        f, _ := os.OpenFile("hello.png", os.O_CREATE|os.O_RDWR, 0644)
        png.Encode(f, img)
}

```

Also included is a "font ripper" tool which will convert an image or text file into a Go font
package file which you can import and use in your codebase without run-time dependencies.
