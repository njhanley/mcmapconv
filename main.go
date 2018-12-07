package main

import (
	"compress/gzip"
	"flag"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io"
	"os"

	"github.com/njhanley/nbt"
)

var options struct {
	outfile string
	verbose bool
}

func init() {
	flag.StringVar(&options.outfile, "o", "out.png", "output filename")
	flag.BoolVar(&options.verbose, "v", false, "verbose mode")
}

type exitCode int

func exit(code int) {
	panic(exitCode(code))
}

func handleExit() {
	if v := recover(); v != nil {
		if code, ok := v.(exitCode); ok {
			os.Exit(int(code))
		}
		panic(v)
	}
}

func info(prefix string, v interface{}) {
	if e, ok := v.(*os.PathError); ok {
		v = e.Err
	}

	if options.verbose {
		fmt.Fprintf(os.Stderr, "%s: %+v\n", prefix, v)
	} else {
		fmt.Fprintf(os.Stderr, "%s: %v\n", prefix, v)
	}
}

func fatal(prefix string, v interface{}) {
	info(prefix, v)
	exit(1)
}

func closeIO(c io.Closer, name string) {
	if err := c.Close(); err != nil {
		fatal(name, err)
	}
}

func readMap(filename string) *Map {
	file, err := os.Open(filename)
	if err != nil {
		fatal(filename, err)
	}
	defer closeIO(file, filename)

	r, err := gzip.NewReader(file)
	if err != nil {
		fatal(filename, err)
	}
	defer closeIO(r, filename)

	tag, err := nbt.NewDecoder(r).Decode()
	if err != nil {
		fatal(filename, err)
	}

	m, err := NewMap(tag)
	if err != nil {
		fatal(filename, err)
	}

	return m
}

func render(a []*Map) image.Image {
	var region image.Rectangle
	for _, m := range a {
		region = region.Union(m.Region)
	}

	canvas := image.NewRGBA(region)
	for scale := 4; scale >= 0; scale-- {
		for _, m := range a {
			if m.Scale == scale {
				draw.Draw(canvas, m.Region, m, image.ZP, draw.Src)
			}
		}
	}

	return canvas
}

func main() {
	defer handleExit()

	flag.Parse()

	maps := make([]*Map, flag.NArg())
	for i, filename := range flag.Args() {
		m := readMap(filename)
		if m.Dimension == Overworld {
			maps[i] = m
		}
	}

	img := render(maps)

	out, err := os.Create(options.outfile)
	if err != nil {
		fatal(options.outfile, err)
	}
	defer closeIO(out, options.outfile)

	if err := png.Encode(out, img); err != nil {
		fatal(options.outfile, err)
	}
}
