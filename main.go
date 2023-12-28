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
	"unicode/utf8"
)

type encoding struct {
	Name        string
	EncoderFunc func([]byte) string
	Enabled     bool
	ByteLength  int
	// single character separator
	Separator string
	Width     int
}

func (e encoding) Encode(chunk []byte) string {

	output := make([]string, 0)
	// loop by bytelength at a time
	for i := 0; i < len(chunk); i += e.ByteLength {
		output = append(output, e.EncoderFunc(chunk[i:i+e.ByteLength]))
	}
	// join with separator
	return fmt.Sprintf("%-*s", e.EncodingWidth(bufferSize), strings.Join(output, e.Separator))
}

func printASCII(chunk []byte) string {
	var output string
	for _, b := range chunk {
		if b >= 32 && b <= 126 { // Printable ASCII range
			output += fmt.Sprintf("%c", b)
		} else {
			output += "." // Non-printable characters are represented as a dot
		}
	}
	return output

}

var utf8Window []byte

func printUTF8(chunk []byte) string {
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
			output += string(utf8Window)
			utf8Window = utf8Window[:0]
		}
	}
	return output
}

func (e encoding) EncodingWidth(bytewidth int) int {
	return (e.Width * (bytewidth / 8))
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
		Width:      40,
	},
	{
		Name: "uint8",
		EncoderFunc: func(b []byte) string {
			return fmt.Sprintf("%d", uint8(b[0]))
		},
		Enabled:    false,
		ByteLength: 1,
		Separator:  `,`,
		Width:      (4 * 8) - 1,
	},
	{
		Name: "int16",
		EncoderFunc: func(b []byte) string {
			return fmt.Sprintf("%d", int16(binary.LittleEndian.Uint16(b)))
		},
		Enabled:    false,
		ByteLength: 2,
		Separator:  `,`,
		Width:      (7 * 4) - 1,
	},
	{
		Name: "uint16",
		EncoderFunc: func(b []byte) string {
			return fmt.Sprintf("%d", binary.LittleEndian.Uint16(b))
		},
		Enabled:    false,
		ByteLength: 2,
		Separator:  `,`,
		Width:      (12 * 2) - 1,
	},
	{
		Name: "int32",
		EncoderFunc: func(b []byte) string {
			return fmt.Sprintf("%d", int32(binary.LittleEndian.Uint32(b)))
		},
		Enabled:    false,
		ByteLength: 4,
		Separator:  `,`,
		Width:      (11 * 2) - 1,
	},
	{
		Name: "uint32",
		EncoderFunc: func(b []byte) string {
			return fmt.Sprintf("%d", binary.LittleEndian.Uint32(b))
		},
		Enabled:    false,
		ByteLength: 4,
		Separator:  `,`,
		Width:      (11 * 2) - 1,
	},
	{
		Name: "float32",
		EncoderFunc: func(b []byte) string {
			return fmt.Sprintf("%12.6f\n", math.Float32frombits(binary.BigEndian.Uint32(b)))
		},
		Enabled:    false,
		ByteLength: 4,
		Separator:  `,`,
		Width:      (13 * 2) - 1,
	},
	{
		Name: "int64",
		EncoderFunc: func(b []byte) string {
			return fmt.Sprintf("%d", int64(binary.BigEndian.Uint64(b)))
		},
		Enabled:    false,
		ByteLength: 8,
		Separator:  `,`,
		Width:      20,
	},
	{
		Name: "uint64",
		EncoderFunc: func(b []byte) string {
			return fmt.Sprintf("%d", binary.BigEndian.Uint64(b))
		},
		Enabled:    false,
		ByteLength: 8,
		Separator:  `,`,
		Width:      20,
	},
	{
		Name: "float64",
		EncoderFunc: func(b []byte) string {
			return fmt.Sprintf("%12.6f\n", math.Float64frombits(binary.BigEndian.Uint64(b)))
		},
		Enabled:    false,
		ByteLength: 8,
		Separator:  `,`,
		Width:      12,
	},
	{
		Name:        "hex",
		EncoderFunc: func(b []byte) string { return fmt.Sprintf("%x", b) },
		Enabled:     false,
		ByteLength:  8,
		Separator:   `,`,
		Width:       16,
	},
	{
		Name:        "ascii",
		EncoderFunc: printASCII,
		Enabled:     false,
		ByteLength:  8,
		Separator:   ``,
		Width:       8,
	},
	{
		Name:        "utf8",
		EncoderFunc: printUTF8,
		Enabled:     false,
		ByteLength:  8,
		Separator:   ``,
		Width:       8,
	},
}

var (
	enabledEncodings = []encoding{}
	bufferSize       int
	numLines         int
)

func init() {
	for i, e := range encodings {
		flag.BoolVar(&encodings[i].Enabled, e.Name, false, fmt.Sprintf("Show %s", e.Name))
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

	var ln []string
	for _, e := range enabledEncodings {
		ln = append(ln, e.Encode(chunk))
	}
	fmt.Fprintln(os.Stdout, strings.Join(ln, " "))

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
