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
	&PackTestCase{
		Width:  5,
		Height: 5,
		Letters: map[int32]map[int]string{
			65: map[int]string{
				0: "  X  ",
				1: " X X ",
				2: "X   X",
				3: "XXXXX",
				4: "X   X",
			},
			66: map[int]string{
				0: "XXXX ",
				1: "X   X",
				2: "XXXX ",
				3: "X   X",
				4: "XXXX ",
			},
			67: map[int]string{
				0: " XXX ",
				1: "X   X",
				2: "X    ",
				3: "X   X",
				4: " XXX ",
			},
			68: map[int]string{
				0: "XXXX ",
				1: "X   X",
				2: "X   X",
				3: "X   X",
				4: "XXXX ",
			},
			69: map[int]string{
				0: "XXXXX",
				1: "X    ",
				2: "XXXX ",
				3: "X    ",
				4: "XXXXX",
			},
		},
		ExpectedEncoding: []uint32{
			// Each glyph takes up 5/8 bits
			//---*****|---*****|---*****|---*****
			0b00001111_00001110_00001111_00000100,
			0b00010001_00010001_00010001_00001010,
			0b00010001_00000001_00001111_00010001,
			0b00010001_00010001_00010001_00011111,
			0b00001111_00001110_00001111_00010001,
			0b00000000_00000000_00000000_00011111,
			0b00000000_00000000_00000000_00000001,
			0b00000000_00000000_00000000_00001111,
			0b00000000_00000000_00000000_00000001,
			0b00000000_00000000_00000000_00011111,
		},
	},
}

func TestGlyphPacking(t *testing.T) {
	for _, c := range packTestCases {
		t.Run(fmt.Sprintf("%dx%d", c.Width, c.Height), func(t *testing.T) {
			encoded, _ := packFont(c.Width, c.Height, c.Letters)
			if len(c.ExpectedEncoding) != len(encoded) {
				t.Fatalf("Expected to find %d lines in encoding, but found %d", len(c.ExpectedEncoding), len(encoded))
			}
			for i, e := range c.ExpectedEncoding {
				if e != encoded[i] {
					t.Errorf("Row %d mismatch\nExpected: %032b\n     Got: %032b\n", i, e, encoded[i])
				}
			}
		})
	}
}
