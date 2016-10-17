package game

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/mattn/go-runewidth"
	log "gopkg.in/inconshreveable/log15.v2"
)

func tbprint(x, y int, fg, bg Attribute, msg string, client *Client) {
	for _, c := range msg {
		SetCell(x, y, c, fg, bg, client)
		x += runewidth.RuneWidth(c)
	}
}

func fill(x, y, w, h int, cell Cell, c *Client) {
	for ly := 0; ly < h; ly++ {
		for lx := 0; lx < w; lx++ {
			SetCell(x+lx, y+ly, cell.Ch, cell.Fg, cell.Bg, c)
		}
	}
}

type EditBox struct {
	text           []byte
	line_voffset   int
	cursor_boffset int // cursor offset in bytes
	cursor_voffset int // visual cursor offset in termbox cells
	cursor_coffset int // cursor offset in unicode code points
}

func (eb *EditBox) CursorX() int {
	return eb.cursor_voffset - eb.line_voffset
}

var edit_box EditBox

const edit_box_width = 120

func redraw(c *Client, reply Reply) {
	log.Info(fmt.Sprintf("Redraw: %s, W: %d H: %d ", c.Player.Nickname, c.Bbuffer.Width, c.Bbuffer.Height))
	const coldef = ColorDefault

	w, h := Size()

	midy := h/2 + 12
	midx := (w - edit_box_width) - 10

	// TODO: Why?
	Clear(coldef, coldef, c)

	buf := bytes.NewBuffer(reply.World)
	rintro := bytes.NewBuffer(reply.Intro)

	//log.Info(fmt.Sprintf("%s\n%s\n%s\n%s\n", c.Player.Nickname, buf, rintro, reply.exits))

	// Editbox
	SetCell(midx-1, midy, '│', coldef, coldef, c)
	SetCell(midx+edit_box_width, midy, '│', coldef, coldef, c)
	SetCell(midx-1, midy-1, '┌', coldef, coldef, c)
	SetCell(midx-1, midy+1, '└', coldef, coldef, c)
	SetCell(midx+edit_box_width, midy-1, '┐', coldef, coldef, c)
	SetCell(midx+edit_box_width, midy+1, '┘', coldef, coldef, c)
	fill(midx, midy-1, edit_box_width, 1, Cell{Ch: '─'}, c)
	fill(midx, midy+1, edit_box_width, 1, Cell{Ch: '─'}, c)

	SetCursor(midx+edit_box.CursorX(), midy, *c)

	counter := 20
	for {
		line, err := buf.ReadString('\n')
		if err != nil {
			//log.Info("world buffer read error: %v", err)
			break
		}
		tbprint(midx+100, midy-counter, coldef, coldef, line, c)
		counter--
	}

	counter2 := 20
	for {
		line, err := rintro.ReadString('\n')
		if err != nil {
			//log.Info("intro buffer read error: %v", err)
			break
		}
		tbprint(midx, midy-counter2, coldef, coldef, line, c)
		counter2--
	}

	tbprint(midx, midy-10, coldef, coldef, reply.Events, c)
	tbprint(midx+90, midy-3, coldef, coldef, reply.Exits, c)
	Flush(c)
}

func Panel(c *Client, wg *sync.WaitGroup, quit <-chan struct{}) {
	defer wg.Done()

	err := Init(c)
	if err != nil {
		panic(err)
	}
	defer Close(*c)

	for {
		select {
		case reply := <-c.Reply:
			redraw(c, reply)
		case <-quit:
			log.Info(fmt.Sprintf("Panel for %q quit", c.Player.Nickname))
			return
		}
	}
}
