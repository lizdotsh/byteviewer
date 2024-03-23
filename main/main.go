package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
)

var (
	enabledEncodings = []encoding{}
	bufferSize       int
	inputFile        string
	numLines         int
)

func init() {
	for i, e := range encodings {
		flag.BoolVar(&encodings[i].Enabled, e.Name, false, e.Desc)
	}
	flag.StringVar(&inputFile, "file", "", "The file to read input from (stdin by default)")
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
		fmt.Fprintln(os.Stderr, "width must be divisible by 8")
		// cli error format

		return
	}

	// Create a buffered reader
	reader := bufio.NewReader(os.Stdin);
	if inputFile != "" {
		file, err := os.Open(inputFile)
		if err != nil {
			fmt.Println("Error opening ", inputFile, ":", err)
			os.Exit(1)
		}
		reader = bufio.NewReader(file)
		defer file.Close()
	}

	// read full buffer

	idx := 0
	printHeader(enabledEncodings)

ReadLoop:
	for {
		chunk := make([]byte, bufferSize)
		n, err := io.ReadFull(reader, chunk)

		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				processLine(chunk[:n], idx)
				break ReadLoop
			}
			fmt.Fprintln(os.Stderr, "error reading input:", err)
			return
		}

		// Only process the bytes that were actually read
		processLine(chunk[:n], idx)

		if idx >= numLines && numLines != 0 {
			break
		}

		idx++
	}
}
