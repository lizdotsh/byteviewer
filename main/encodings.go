package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"
	"unicode"
	"unicode/utf8"
)

type encoding struct {
	Name        string
	EncoderFunc func([]byte) (string, int)
	Enabled     bool
	ByteLength  int
	// single character separator
	Separator string
	Desc      string
	MaxWidth  int
	buffer    []byte
}

func (e *encoding) Encode(chunk []byte) string {

	output := make([]string, 0)
	e.buffer = append(e.buffer, chunk...)
	increment := max(e.ByteLength, 1)
	start := 0
	// loop by bytelength at a time
	for end := increment; end <= len(e.buffer); end += increment {
		encoded, consumed := e.EncoderFunc(e.buffer[start:end])
		start += consumed
		output = append(output, encoded)
	}
	e.buffer = e.buffer[start:]
	// join with separator
	// return string with padding
	wdth := e.EncodingWidth(bufferSize)
	if len(output) > wdth {
		wdth += 2
	}
	return fmt.Sprintf("%-*s", wdth, strings.Join(output, e.Separator))
}

// map unicode control chars to ascii equivalents
func unicodeControlToASCII(unicodeRune rune) rune {
	if !unicode.IsControl(unicodeRune) {
		return unicodeRune
	}
	switch unicodeRune {
	case 0x0000:
		return '␀' // null
	case 0x0007:
		return '␇' // bell
	case 0x0008:
		return '⌫' // backspace
	case 0x0009:
		return '⇥' // tab
	case 0x000A, 0x000B, 0x000C, 0x000D, 0x0085, 0x2028, 0x2029:
		return '⏎' // newline and related
	case 0x001B:
		return '⎋' // escape
	default:
		// return generic sp char
		return '␀'
	}
}

var mapInvalidChar = map[uint8]rune{
	'\n':   '⏎',
	'\t':   '⇥',
	'\r':   '↵',
	'\v':   '↴',
	'\f':   '↵',
	'\b':   '⌫',
	'\a':   '␇',
	'\x1b': '⎋',
	'\x00': '␀',
}

func parseASCII(chunk []byte) (string, int) {
	var output string
	for _, b := range chunk {
		if b >= 32 && b <= 126 { // Printable ASCII range
			rn, ok := mapInvalidChar[b]
			if ok {
				output += fmt.Sprintf("%c", rn)
			} else {
				output += fmt.Sprintf("%c", b)
			}
		} else {
			output += "." // Non-printable characters are represented as a dot
		}
	}
	return output, len(chunk)

}

func parseUTF8(chunk []byte) (string, int) {
	if utf8.Valid(chunk) {
		r, _ := utf8.DecodeRune(chunk)
		// replace control chars with unicode equivalents
		return fmt.Sprintf("%c", r), len(chunk)
	} else {
		for i := 1; i < len(chunk); i++ {
			if utf8.RuneStart(chunk[i]) || i > utf8.UTFMax {
				// Either a new rune has been started without the last one being finished or we've gotten
				// more bytes than fit in a UTF-8 rune.
				// Non-printable characters are represented as U+FFFD (REPLACEMENT CHARACTER)
				return "�", i
			}
		}
	}
	return "", 0
}

func (e *encoding) EncodingWidth(bytewidth int) int {
	numEntries := bytewidth / max(e.ByteLength, 1)
	return ((e.MaxWidth * numEntries) + (numEntries - 1) * len(e.Separator)) // separators
}

var encodings = []encoding{
	{
		Name: "int8",
		EncoderFunc: func(b []byte) (string, int) {
			return fmt.Sprintf("%d", int8(b[0])), len(b)
		},
		Enabled:    false,
		ByteLength: 1,
		Separator:  `,`,
		MaxWidth:   4,
		Desc:       `Signed 8-bit integer`,
	},
	{
		Name: "uint8",
		EncoderFunc: func(b []byte) (string, int) {
			return fmt.Sprintf("%d", uint8(b[0])), len(b)
		},
		Enabled:    false,
		ByteLength: 1,
		Separator:  `,`,
		MaxWidth:   3,
		Desc:       `Unsigned 8-bit integer`,
	},
	{
		Name: "int16",
		EncoderFunc: func(b []byte) (string, int) {
			return fmt.Sprintf("%d", int16(binary.LittleEndian.Uint16(b))), len(b)
		},
		Enabled:    false,
		ByteLength: 2,
		Separator:  `,`,
		MaxWidth:   6,
		Desc:       `Signed 16-bit integer`,
	},
	{
		Name: "uint16",
		EncoderFunc: func(b []byte) (string, int) {
			return fmt.Sprintf("%d", binary.LittleEndian.Uint16(b)), len(b)
		},
		Enabled:    false,
		ByteLength: 2,
		Separator:  `,`,
		MaxWidth:   11,
		Desc:       `Unsigned 16-bit integer`,
	},
	{
		Name: "int32",
		EncoderFunc: func(b []byte) (string, int) {
			return fmt.Sprintf("%d", int32(binary.LittleEndian.Uint32(b))), len(b)
		},
		Enabled:    false,
		ByteLength: 4,
		Separator:  `,`,
		MaxWidth:   10,
		Desc:       `Signed 32-bit integer`,
	},
	{
		Name: "uint32",
		EncoderFunc: func(b []byte) (string, int) {
			return fmt.Sprintf("%d", binary.LittleEndian.Uint32(b)), len(b)
		},
		Enabled:    false,
		ByteLength: 4,
		Separator:  `,`,
		MaxWidth:   11,
		Desc:       `Unsigned 32-bit integer`,
	},
	{
		Name: "float32",
		EncoderFunc: func(b []byte) (string, int) {
			return fmt.Sprintf("%12.6f\n", math.Float32frombits(binary.BigEndian.Uint32(b))), len(b)
		},
		Enabled:    false,
		ByteLength: 4,
		Separator:  `,`,
		MaxWidth:   12,
		Desc:       `IEEE 754 single-precision binary floating-point format: sign bit, 8 bits exponent, 23 bits mantissa`,
	},
	{
		Name: "int64",
		EncoderFunc: func(b []byte) (string, int) {
			return fmt.Sprintf("%d", int64(binary.BigEndian.Uint64(b))), len(b)
		},
		Enabled:    false,
		ByteLength: 8,
		Separator:  `,`,
		MaxWidth:   20,
		Desc:       `Signed 64-bit integer`,
	},
	{
		Name: "uint64",
		EncoderFunc: func(b []byte) (string, int) {
			return fmt.Sprintf("%d", binary.BigEndian.Uint64(b)), len(b)
		},
		Enabled:    false,
		ByteLength: 8,
		Separator:  `,`,
		MaxWidth:   20,
		Desc:       `Unsigned 64-bit integer`,
	},
	{
		Name: "float64",
		EncoderFunc: func(b []byte) (string, int) {
			return fmt.Sprintf("%12.6f\n", math.Float64frombits(binary.BigEndian.Uint64(b))), len(b)
		},
		Enabled:    false,
		ByteLength: 8,
		Separator:  `,`,
		MaxWidth:   12,
		Desc:       `IEEE 754 double-precision binary floating-point format: sign bit, 11 bits exponent, 52 bits mantissa`,
	},
	{
		Name:        "hex",
		EncoderFunc: func(b []byte) (string, int) { return fmt.Sprintf("%x", b), len(b) },
		Enabled:     false,
		ByteLength:  1,
		Separator:   `,`,
		MaxWidth:    2,
		Desc:        `Hexadecimal encoding`,
	},
	{
		Name:        "ascii",
		EncoderFunc: parseASCII,
		Enabled:     false,
		ByteLength:  1,
		Separator:   ``,
		MaxWidth:    1,
		Desc:        `ASCII encoded text. Non-printable characters are represented as a dot and the following characters are replaced with their unicode equivalents: \\n, \\t, \\r, \\v, \\f, \\b, \\a, \\x1b`,
	},
	{
		Name:        "utf8",
		EncoderFunc: parseUTF8,
		Enabled:     false,
		ByteLength:  0,
		Separator:   ``,
		MaxWidth:    1,
		Desc:        `UTF-8 encoded text. Replaces control characters with unicode symbol equivalents (mostly). Uses a global variable to rollover between chunks.`,
	},
}
