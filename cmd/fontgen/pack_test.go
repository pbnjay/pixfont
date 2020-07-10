package main

import (
	"fmt"
	"testing"
)

type PackTestCase struct {
	Width, Height    int
	Letters          map[int32]map[int]string
	ExpectedEncoding []uint32
}

var packTestCases = []*PackTestCase{
	&PackTestCase{
		Width:  7,
		Height: 1,
		Letters: map[int32]map[int]string{
			65: map[int]string{0: "XXX   X"},
			66: map[int]string{0: "XX  XX "},
			67: map[int]string{0: "X X X X"},
		},
		ExpectedEncoding: []uint32{
			// Each glyph takes up 7/8 bits
			//-*******|-*******|-*******|-*******
			0b00000000_01010101_00110011_01000111,
		},
	},
	&PackTestCase{
		Width:  8,
		Height: 1,
		Letters: map[int32]map[int]string{
			65: map[int]string{0: "XXX   XX"},
			66: map[int]string{0: "XX  XX  "},
			67: map[int]string{0: "X X X X "},
		},
		ExpectedEncoding: []uint32{
			// Each glyph takes up 8/8 bits
			//********|********|********|********
			0b00000000_01010101_00110011_11000111,
		},
	},
	&PackTestCase{
		Width:  9,
		Height: 1,
		Letters: map[int32]map[int]string{
			65: map[int]string{0: "XXX   XXX"},
			66: map[int]string{0: "XX  XX  X"},
			67: map[int]string{0: "X X X X X"},
		},
		ExpectedEncoding: []uint32{
			// Each glyph takes up 9/16 bits. Because
			// it didn't fit in 8 bits, we can only fit
			// two per uint32
			//-------*********|-------*********
			0b0000000100110011_0000000111000111,
			0b0000000000000000_0000000101010101,
		},
	},
	&PackTestCase{
		Width:  16,
		Height: 1,
		Letters: map[int32]map[int]string{
			65: map[int]string{0: "XXX   XXX   XXX "},
			66: map[int]string{0: "XX  XX  XX  XX  "},
			67: map[int]string{0: "X X X X X X X X "},
		},
		ExpectedEncoding: []uint32{
			// Each glyph takes up 16/16 bits
			//****************|****************
			0b0011001100110011_0111000111000111,
			0b0000000000000000_0101010101010101,
		},
	},
	&PackTestCase{
		Width:  17,
		Height: 1,
		Letters: map[int32]map[int]string{
			65: map[int]string{0: "XXX   XXX   XXX  "},
			66: map[int]string{0: "XX  XX  XX  XX  X"},
			67: map[int]string{0: "X X X X X X X X X"},
		},
		ExpectedEncoding: []uint32{
			// Each glyph takes up 17/32 bits. Because
			// it didn't fit in 16 bits, we can only fit
			// *one* per uint32
			//---------------*****************
			0b00000000000000000111000111000111,
			0b00000000000000010011001100110011,
			0b00000000000000010101010101010101,
		},
	},
	&PackTestCase{
		Width:  32,
		Height: 1,
		Letters: map[int32]map[int]string{
			65: map[int]string{0: "XXX   XXX   XXX   XXX   XXX   XX"},
			66: map[int]string{0: "XX  XX  XX  XX  XX  XX  XX  XX  "},
			67: map[int]string{0: "X X X X X X X X X X X X X X X X "},
		},
		ExpectedEncoding: []uint32{
			// Each glyph takes up 32/32 bits.
			//********************************
			0b11000111000111000111000111000111,
			0b00110011001100110011001100110011,
			0b01010101010101010101010101010101,
		},
	},
}

func TestGlyphPacking(t *testing.T) {
	for _, c := range packTestCases {
		t.Run(fmt.Sprintf("%dx%d", c.Width, c.Height), func(t *testing.T) {
			encoded, _ := packFont(c.Width, c.Height, c.Letters)
			for i, e := range c.ExpectedEncoding {
				if i > len(encoded)+1 {
					t.Fatalf("Expected to find %d lines in encoding, but only found %d", len(c.ExpectedEncoding), len(encoded))
				}
				if e != encoded[i] {
					t.Errorf("Row %d mismatch\nExpected: %032b\n     Got: %032b\n", i, e, encoded[i])
				}
			}
		})
	}
}
