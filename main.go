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
	"path/filepath"

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

func readMap(filename string) (*Map, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer closeIO(file, filename)

	r, err := gzip.NewReader(file)
	if err != nil {
		return nil, err
	}
	defer closeIO(r, filename)

	tag, err := nbt.NewDecoder(r).Decode()
	if err != nil {
		return nil, err
	}

	return NewMap(tag)
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

	var maps []*Map
	for _, filename := range flag.Args() {
		if err := filepath.Walk(filename, func(path string, fi os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if fi.IsDir() {
				if path == filename {
					return nil
				}
				return filepath.SkipDir
			}

			m, err := readMap(path)
			if err != nil {
				return err
			}

			if m.Dimension == Overworld {
				maps = append(maps, m)
			}

			return nil
		}); err != nil {
			fatal(filename, err)
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
