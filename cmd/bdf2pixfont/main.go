// Command bdf2pixfont opens a BDF format font and creates a new pixel font for it.
package main

import (
	"fmt"
	"os"
	"sort"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "USAGE: %s filename.bdf > filename.txt", os.Args[0])
		os.Exit(1)
	}
	f, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	bfont, err := OpenBDF(f)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	all := make([]rune, 0, len(bfont.Glyphs))
	for r := range bfont.Glyphs {
		all = append(all, r)
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i] < all[j]
	})
	for _, r := range all {
		data := bfont.Glyphs[r]
		fmt.Println(data)
	}

	f.Close()
}
