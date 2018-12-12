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

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: %s [flags] path [path ...]\n", os.Args[0])
		flag.PrintDefaults()
	}
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

func stderrf(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format, a...)
}

func fatalf(format string, a ...interface{}) {
	stderrf(format, a...)
	exit(1)
}

func unwrapPathError(err error) error {
	if e, ok := err.(*os.PathError); ok {
		return e.Err
	}
	return err
}

func closeIO(c io.Closer, err *error) {
	if e := c.Close(); e != nil && *err == nil {
		*err = e
	}
}

func readMap(filename string) (m *Map, err error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer closeIO(file, &err)

	r, err := gzip.NewReader(file)
	if err != nil {
		return nil, err
	}
	defer closeIO(r, &err)

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

func writePNG(filename string, img image.Image) (err error) {
	out, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer closeIO(out, &err)

	return png.Encode(out, img)
}

func main() {
	defer handleExit()

	flag.Parse()

	if flag.NArg() == 0 {
		flag.Usage()
		exit(2)
	}

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
			fatalf("%s: %v\n", filename, unwrapPathError(err))
		}
	}

	img := render(maps)

	if err := writePNG(options.outfile, img); err != nil {
		fatalf("%s: %v\n", options.outfile, unwrapPathError(err))
	}
}
