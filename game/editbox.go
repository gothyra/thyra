package game

import (
	"bytes"
	"fmt"
	"unicode/utf8"

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

func rune_advance_len(r rune, pos int) int {
	if r == '\t' {
		return tabstop_length - pos%tabstop_length
	}
	return runewidth.RuneWidth(r)
}

func voffset_coffset(text []byte, boffset int) (voffset, coffset int) {
	text = text[:boffset]
	for len(text) > 0 {
		r, size := utf8.DecodeRune(text)
		text = text[size:]
		coffset += 1
		voffset += rune_advance_len(r, voffset)
	}
	return
}

func byte_slice_grow(s []byte, desired_cap int) []byte {
	if cap(s) < desired_cap {
		ns := make([]byte, len(s), desired_cap)
		copy(ns, s)
		return ns
	}
	return s
}

func byte_slice_remove(text []byte, from, to int) []byte {
	size := to - from
	copy(text[from:], text[to:])
	text = text[:len(text)-size]
	return text
}

func byte_slice_insert(text []byte, offset int, what []byte) []byte {
	n := len(text) + len(what)
	text = byte_slice_grow(text, n)
	text = text[:n]
	copy(text[offset+len(what):], text[offset:])
	copy(text[offset:], what)
	return text
}

const preferred_horizontal_threshold = 5
const tabstop_length = 8

type EditBox struct {
	text           []byte
	line_voffset   int
	cursor_boffset int // cursor offset in bytes
	cursor_voffset int // visual cursor offset in termbox cells
	cursor_coffset int // cursor offset in unicode code points
}

// Draws the EditBox in the given location, 'h' is not used at the moment
func (eb *EditBox) Draw(x, y, w, h int, c *Client) {
	eb.AdjustVOffset(w)

	const coldef = ColorDefault
	fill(x, y, w, h, Cell{Ch: ' '}, c)

	t := eb.text
	lx := 0
	tabstop := 0
	for {
		rx := lx - eb.line_voffset
		if len(t) == 0 {
			break
		}

		if lx == tabstop {
			tabstop += tabstop_length
		}

		if rx >= w {
			SetCell(x+w-1, y, '→',
				coldef, coldef, c)
			break
		}

		r, size := utf8.DecodeRune(t)
		if r == '\t' {
			for ; lx < tabstop; lx++ {
				rx = lx - eb.line_voffset
				if rx >= w {
					goto next
				}

				if rx >= 0 {
					SetCell(x+rx, y, ' ', coldef, coldef, c)
				}
			}
		} else {
			if rx >= 0 {
				SetCell(x+rx, y, r, coldef, coldef, c)
			}
			lx += runewidth.RuneWidth(r)
		}
	next:
		t = t[size:]
	}

	if eb.line_voffset != 0 {
		SetCell(x, y, '←', coldef, coldef, c)
	}
}

// Adjusts line visual offset to a proper value depending on width
func (eb *EditBox) AdjustVOffset(width int) {
	ht := preferred_horizontal_threshold
	max_h_threshold := (width - 1) / 2
	if ht > max_h_threshold {
		ht = max_h_threshold
	}

	threshold := width - 1
	if eb.line_voffset != 0 {
		threshold = width - ht
	}
	if eb.cursor_voffset-eb.line_voffset >= threshold {
		eb.line_voffset = eb.cursor_voffset + (ht - width + 1)
	}

	if eb.line_voffset != 0 && eb.cursor_voffset-eb.line_voffset < ht {
		eb.line_voffset = eb.cursor_voffset - ht
		if eb.line_voffset < 0 {
			eb.line_voffset = 0
		}
	}
}

func (eb *EditBox) MoveCursorTo(boffset int) {
	eb.cursor_boffset = boffset
	eb.cursor_voffset, eb.cursor_coffset = voffset_coffset(eb.text, boffset)
}

func (eb *EditBox) RuneUnderCursor() (rune, int) {
	return utf8.DecodeRune(eb.text[eb.cursor_boffset:])
}

func (eb *EditBox) RuneBeforeCursor() (rune, int) {
	return utf8.DecodeLastRune(eb.text[:eb.cursor_boffset])
}

func (eb *EditBox) MoveCursorOneRuneBackward() {
	if eb.cursor_boffset == 0 {
		return
	}
	_, size := eb.RuneBeforeCursor()
	eb.MoveCursorTo(eb.cursor_boffset - size)
}

func (eb *EditBox) MoveCursorOneRuneForward() {
	if eb.cursor_boffset == len(eb.text) {
		return
	}
	_, size := eb.RuneUnderCursor()
	eb.MoveCursorTo(eb.cursor_boffset + size)
}

func (eb *EditBox) MoveCursorToBeginningOfTheLine() {
	eb.MoveCursorTo(0)
}

func (eb *EditBox) MoveCursorToEndOfTheLine() {
	eb.MoveCursorTo(len(eb.text))
}

func (eb *EditBox) DeleteRuneBackward() {
	if eb.cursor_boffset == 0 {
		return
	}

	eb.MoveCursorOneRuneBackward()
	_, size := eb.RuneUnderCursor()
	eb.text = byte_slice_remove(eb.text, eb.cursor_boffset, eb.cursor_boffset+size)
}

func (eb *EditBox) DeleteRuneForward() {
	if eb.cursor_boffset == len(eb.text) {
		return
	}
	_, size := eb.RuneUnderCursor()
	eb.text = byte_slice_remove(eb.text, eb.cursor_boffset, eb.cursor_boffset+size)
}

func (eb *EditBox) DeleteTheRestOfTheLine() {
	eb.text = eb.text[:eb.cursor_boffset]
}

func (eb *EditBox) ClearEdit() {
	eb.text = []byte{}
}

func (eb *EditBox) InsertRune(r rune) {
	var buf [utf8.UTFMax]byte
	n := utf8.EncodeRune(buf[:], r)
	eb.text = byte_slice_insert(eb.text, eb.cursor_boffset, buf[:n])
	eb.MoveCursorOneRuneForward()
}

// Please, keep in mind that cursor depends on the value of line_voffset, which
// is being set on Draw() call, so.. call this method after Draw() one.
func (eb *EditBox) CursorX() int {
	return eb.cursor_voffset - eb.line_voffset
}

var edit_box EditBox

const edit_box_width = 120

func redraw_all(c *Client) {
	log.Info(fmt.Sprintf("Redraw: %s ,W:%d H:%d ", c.Player.Nickname, c.Bbuffer.Width, c.Bbuffer.Height))
	const coldef = ColorDefault

	w, h := Size()

	midy := h/2 + 12
	midx := (w - edit_box_width) - 10

	log.Info("Redraw : Before Reply")
	reply := <-c.Reply
	Clear(coldef, coldef, c)
	log.Info("Redraw : After Reply")
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

func Go_editbox(c *Client) {
	err := Init(c)
	if err != nil {
		panic(err)
	}
	defer Close(*c)

	for {
		redraw_all(c)

	}
}
