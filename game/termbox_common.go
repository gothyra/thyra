package game

import (
	"fmt"

	log "gopkg.in/inconshreveable/log15.v2"
)

// private API, common OS agnostic part

type Cellbuf struct {
	Width  int
	Height int
	Cells  []Cell
}

func New(Width, Height int) *Cellbuf {
	//flag.Parse()
	log.Info(fmt.Sprintf("Enter Cellbuf init %d , %d :", Width, Height))
	return &Cellbuf{
		Width:  Width,
		Height: Height,
		Cells:  make([]Cell, Width*Height),
	}
}

func (this *Cellbuf) resize(Width, Height int) {
	if this.Width == Width && this.Height == Height {
		return
	}

	oldw := this.Width
	oldh := this.Height
	oldCells := this.Cells

	this = New(Width, Height)
	this.clear()

	minw, minh := oldw, oldh

	if Width < minw {
		minw = Width
	}
	if Height < minh {
		minh = Height
	}

	for i := 0; i < minh; i++ {
		srco, dsto := i*oldw, i*Width
		src := oldCells[srco : srco+minw]
		dst := this.Cells[dsto : dsto+minw]
		copy(dst, src)
	}
}

func (this *Cellbuf) clear() {

	log.Info(fmt.Sprintf("clear():Size Cells:", len(this.Cells)))
	for i := range this.Cells {
		c := &this.Cells[i]
		c.Ch = ' '
		c.Fg = foreground
		c.Bg = background
	}
}

const cursor_hidden = -1

func is_cursor_hidden(x, y int) bool {
	return x == cursor_hidden || y == cursor_hidden
}
