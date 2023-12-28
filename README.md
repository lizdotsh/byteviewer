# Byteviewer

Very simple CLI tool that reads a file from the standard input and prints it in a series of different encodings. Right now only supports very basic encodings, and reads n bytes at a time (set with -width, must be a multiple of 8)

## Installation

Simply build the program and install it on your local system

```
go build -o byteviewer ./main
sudo mv byteviewer /usr/local/bin
```

## Usage

Simply read from any standard input and flag all the formats you want to display, like so:

```bash
cat myfile | byteviewer -int32 -hex -uint8

uint8                           int32                 hex
-----------------------------------------------------------------------
184,11,0,0,89,110,0,0           3000,28249            b80b0000596e0000
92,126,0,0,95,142,0,0           32348,36447           5c7e00005f8e0000
98,158,0,0,101,174,0,0          40546,44645           629e000065ae0000
104,190,0,0,107,206,0,0         48744,52843           68be00006bce0000
110,222,0,0,113,238,0,0         56942,61041           6ede000071ee0000
```

(sample output)

You can also just do:

```bash
byteviewer -int32 -hex -uint8 < myfile
```

slightly faster than using cat, but doesn't really matter

Supported formats (WIP):

1. int8
2. uint8
3. int16
4. uint16
5. int32
6. uint32
7. float32
8. int64
9. uint64
10. float64
11. hex
12. ascii
13. utf8
