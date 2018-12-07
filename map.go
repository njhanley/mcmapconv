package main

import (
	"image"
	"image/color"
	"runtime"

	"github.com/njhanley/nbt"
)

type Dimension int

const (
	Overworld Dimension = 0
	Nether    Dimension = -1
	End       Dimension = 1
)

func (dim Dimension) String() string {
	switch dim {
	case Overworld:
		return "Overworld"
	case Nether:
		return "Nether"
	case End:
		return "End"
	default:
		return "Unknown"
	}
}

type Map struct {
	Scale     int
	Dimension Dimension
	Region    image.Rectangle

	center image.Point
	bounds image.Rectangle
	colors []byte
}

func NewMap(tag *nbt.NamedTag) (m *Map, err error) {
	defer func() {
		switch v := recover(); e := v.(type) {
		case nil:
		case *runtime.TypeAssertionError:
			err = e
		default:
			panic(v)
		}
	}()

	data := tag.ToCompound()["data"].ToCompound()

	m = &Map{
		Scale:     int(data["scale"].ToByte()),
		Dimension: Dimension(data["dimension"].ToInt()),
		center: image.Point{
			X: int(data["xCenter"].ToInt()),
			Y: int(data["zCenter"].ToInt()),
		},
		colors: data["colors"].ToByteArray(),
	}

	width := 128 << uint(m.Scale)
	m.bounds = image.Rect(0, 0, width, width)
	m.Region = m.bounds.Sub(image.Pt(width/2, width/2)).Add(m.center)

	return m, nil
}

func (m *Map) ColorModel() color.Model {
	return color.RGBAModel
}

func (m *Map) Bounds() image.Rectangle {
	return m.bounds
}

func (m *Map) At(x, y int) color.Color {
	if image.Pt(x, y).In(m.bounds) {
		return colorTable[m.colors[(x>>uint(m.Scale))+(y>>uint(m.Scale))*128]]
	}
	return color.RGBA{}
}
