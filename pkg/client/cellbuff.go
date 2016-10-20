package client

import ()

type Cellbuf struct {
	Width  int
	Height int
	Cells  []Cell
}

type Cell struct {
	Ch rune
	Fg Attribute
	Bg Attribute
}

type Attribute uint16

const (
	ColorDefault Attribute = iota
	ColorBlack
	ColorRed
	ColorGreen
	ColorYellow
	ColorBlue
	ColorMagenta
	ColorCyan
	ColorWhite
)

func New(width, height int, foreground, background Attribute) *Cellbuf {
	cb := &Cellbuf{
		Width:  width,
		Height: height,
		Cells:  make([]Cell, width*height),
	}
	return cb.initialized(foreground, background)
}

// resize will resize this cellbuff with the given width and height
func (cb *Cellbuf) resize(width, height int, foreground, background Attribute) {
	if cb.Width == width && cb.Height == height {
		return
	}

	oldw := cb.Width
	oldh := cb.Height
	oldCells := cb.Cells

	cb = New(width, height, foreground, background)

	minw, minh := oldw, oldh

	if width < minw {
		minw = width
	}
	if height < minh {
		minh = height
	}

	for i := 0; i < minh; i++ {
		srco, dsto := i*oldw, i*width
		src := oldCells[srco : srco+minw]
		dst := cb.Cells[dsto : dsto+minw]
		copy(dst, src)
	}
}

func (cb *Cellbuf) initialized(foreground, background Attribute) *Cellbuf {
	for i := range cb.Cells {
		c := &cb.Cells[i]
		c.Ch = ' '
		c.Fg = foreground
		c.Bg = background
	}
	return cb
}
