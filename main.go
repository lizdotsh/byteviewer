package main

import (
	"bufio"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"
)

type encoding struct {
	Name        string
	EncoderFunc func([]byte) string
	Enabled     bool
	ByteLength  int
	// single character separator
	Separator string
	Desc      string
	MaxWidth  int
}

func (e *encoding) Encode(chunk []byte) string {

	output := make([]string, 0)
	// loop by bytelength at a time
	for i := 0; i < len(chunk); i += e.ByteLength {
		output = append(output, e.EncoderFunc(chunk[i:i+e.ByteLength]))
	}
	// join with separator
	// return string with padding
	wdth := e.EncodingWidth(bufferSize)
	if len(output) < wdth {
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

func parseASCII(chunk []byte) string {
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
	return output

}

var utf8Window []byte

func parseUTF8(chunk []byte) string {
	var output string
	for _, b := range chunk {
		if (utf8.RuneStart(b) && len(utf8Window) > 0) || len(utf8Window) >= utf8.UTFMax {
			// Either a new rune has been started without the last one being finished or we've gotten
			// more bytes than fit in a UTF-8 rune. Give up on the current window.
			output += "�"               // Non-printable characters are represented as U+FFFD (REPLACEMENT CHARACTER)
			utf8Window = utf8Window[:0] // Clear the window
		}
		utf8Window = append(utf8Window, b)
		if len(utf8Window) > 0 && utf8.Valid(utf8Window) {
			r, _ := utf8.DecodeRune(utf8Window)

			// replace control chars with unicode equivalents
			output += fmt.Sprintf("%c", unicodeControlToASCII(r))
			utf8Window = utf8Window[:0]
		}
	}
	// replace \n with \u23CE (RETURN SYMBOL)
	return output
}

func (e *encoding) EncodingWidth(bytewidth int) int {
	numEntries := (8 / e.ByteLength)

	return (bytewidth / 8) * ((e.MaxWidth * numEntries) + (numEntries - 1)) // separators
}

var encodings = []encoding{
	{
		Name: "int8",
		EncoderFunc: func(b []byte) string {
			return fmt.Sprintf("%d", int8(b[0]))
		},
		Enabled:    false,
		ByteLength: 1,
		Separator:  `,`,
		MaxWidth:   4,
		Desc:       `Signed 8-bit integer`,
	},
	{
		Name: "uint8",
		EncoderFunc: func(b []byte) string {
			return fmt.Sprintf("%d", uint8(b[0]))
		},
		Enabled:    false,
		ByteLength: 1,
		Separator:  `,`,
		MaxWidth:   3,
		Desc:       `Unsigned 8-bit integer`,
	},
	{
		Name: "int16",
		EncoderFunc: func(b []byte) string {
			return fmt.Sprintf("%d", int16(binary.LittleEndian.Uint16(b)))
		},
		Enabled:    false,
		ByteLength: 2,
		Separator:  `,`,
		MaxWidth:   6,
		Desc:       `Signed 16-bit integer`,
	},
	{
		Name: "uint16",
		EncoderFunc: func(b []byte) string {
			return fmt.Sprintf("%d", binary.LittleEndian.Uint16(b))
		},
		Enabled:    false,
		ByteLength: 2,
		Separator:  `,`,
		MaxWidth:   11,
		Desc:       `Unsigned 16-bit integer`,
	},
	{
		Name: "int32",
		EncoderFunc: func(b []byte) string {
			return fmt.Sprintf("%d", int32(binary.LittleEndian.Uint32(b)))
		},
		Enabled:    false,
		ByteLength: 4,
		Separator:  `,`,
		MaxWidth:   10,
		Desc:       `Signed 32-bit integer`,
	},
	{
		Name: "uint32",
		EncoderFunc: func(b []byte) string {
			return fmt.Sprintf("%d", binary.LittleEndian.Uint32(b))
		},
		Enabled:    false,
		ByteLength: 4,
		Separator:  `,`,
		MaxWidth:   11,
		Desc:       `Unsigned 32-bit integer`,
	},
	{
		Name: "float32",
		EncoderFunc: func(b []byte) string {
			return fmt.Sprintf("%12.6f\n", math.Float32frombits(binary.BigEndian.Uint32(b)))
		},
		Enabled:    false,
		ByteLength: 4,
		Separator:  `,`,
		MaxWidth:   12,
		Desc:       `IEEE 754 single-precision binary floating-point format: sign bit, 8 bits exponent, 23 bits mantissa`,
	},
	{
		Name: "int64",
		EncoderFunc: func(b []byte) string {
			return fmt.Sprintf("%d", int64(binary.BigEndian.Uint64(b)))
		},
		Enabled:    false,
		ByteLength: 8,
		Separator:  `,`,
		MaxWidth:   20,
		Desc:       `Signed 64-bit integer`,
	},
	{
		Name: "uint64",
		EncoderFunc: func(b []byte) string {
			return fmt.Sprintf("%d", binary.BigEndian.Uint64(b))
		},
		Enabled:    false,
		ByteLength: 8,
		Separator:  `,`,
		MaxWidth:   20,
		Desc:       `Unsigned 64-bit integer`,
	},
	{
		Name: "float64",
		EncoderFunc: func(b []byte) string {
			return fmt.Sprintf("%12.6f\n", math.Float64frombits(binary.BigEndian.Uint64(b)))
		},
		Enabled:    false,
		ByteLength: 8,
		Separator:  `,`,
		MaxWidth:   12,
		Desc:       `IEEE 754 double-precision binary floating-point format: sign bit, 11 bits exponent, 52 bits mantissa`,
	},
	{
		Name:        "hex",
		EncoderFunc: func(b []byte) string { return fmt.Sprintf("%x", b) },
		Enabled:     false,
		ByteLength:  8,
		Separator:   `,`,
		MaxWidth:    16,
		Desc:        `Hexadecimal encoding`,
	},
	{
		Name:        "ascii",
		EncoderFunc: parseASCII,
		Enabled:     false,
		ByteLength:  8,
		Separator:   ``,
		MaxWidth:    8,
		Desc:        `ASCII encoded text. Non-printable characters are represented as a dot and the following characters are replaced with their unicode equivalents: \\n, \\t, \\r, \\v, \\f, \\b, \\a, \\x1b`,
	},
	{
		Name:        "utf8",
		EncoderFunc: parseUTF8,
		Enabled:     false,
		ByteLength:  8,
		Separator:   ``,
		MaxWidth:    8,
		Desc:        `UTF-8 encoded text. Replaces the following characters with their unicode equivalents: \\n, \\t, \\r, \\v, \\f, \\b, \\a, \\x1b. Uses a global variable to rollover between chunks.`,
	},
}

var (
	enabledEncodings = []encoding{}
	bufferSize       int
	numLines         int
)

func init() {
	for i, e := range encodings {
		flag.BoolVar(&encodings[i].Enabled, e.Name, false, e.Desc)
	}
	flag.IntVar(&bufferSize, "width", 8, "How many bytes to print per line (must be multiple of 8)")

	flag.IntVar(&numLines, "n", 0, "How many lines to print")
	flag.Parse()

	for _, e := range encodings {
		if e.Enabled {
			enabledEncodings = append(enabledEncodings, e)
		}
	}
	if len(enabledEncodings) == 0 {
		for i, e := range encodings {
			if e.Name == "hex" || e.Name == "ascii" || e.Name == "int8" {
				encodings[i].Enabled = true
				enabledEncodings = append(enabledEncodings, e)
			}
		}
	}

}
func printHeader(enc []encoding) {
	sepWidth := 0
	for _, e := range enc {
		stri := fmt.Sprintf("%-*s ", e.EncodingWidth(bufferSize), e.Name)
		sepWidth += len(stri)
		fmt.Fprint(os.Stdout, stri)
	}
	fmt.Fprint(os.Stdout, "\n")
	var sep string
	for i := 0; i < sepWidth; i++ {
		sep += "-"
	}
	fmt.Fprintln(os.Stdout, sep)
}
func processLine(chunk []byte, idx int) {

	var ln string
	for i := 0; i < len(enabledEncodings); i++ {
		ln += enabledEncodings[i].Encode(chunk) + " "
	}
	fmt.Fprintln(os.Stdout, ln)

}
func main() {
	if bufferSize%8 != 0 {
		fmt.Println("width must be divisible by 8")
		return
	}
	reader := bufio.NewReader(os.Stdin)
	idx := 0
	printHeader(enabledEncodings)
	for {
		chunk := make([]byte, bufferSize)
		n, err := reader.Read(chunk)
		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Fprintln(os.Stderr, "error reading standard input:", err)
			return
		}
		if idx >= numLines && numLines != 0 {
			break
		}

		// Only process the bytes that were actually read
		processLine(chunk[:n], idx)
		idx++
	}
}
