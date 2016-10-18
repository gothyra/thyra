// +build !windows

package game

import (
	"fmt"
	"io"
	"strconv"
	"unicode/utf8"

	log "gopkg.in/inconshreveable/log15.v2"
)

const (
	ti_mouse_enter = "\x1b[?1000h\x1b[?1002h\x1b[?1015h\x1b[?1006h"
	ti_mouse_leave = "\x1b[?1006l\x1b[?1015l\x1b[?1002l\x1b[?1000l"
)

const (
	t_enter_ca = iota
	t_exit_ca
	t_clear_screen
	t_max_funcs
)

const (
	coord_invalid = -2
	attr_invalid  = Attribute(0xFFFF)
)

var (
	// term specific sequences
	keys  []string
	funcs []string
	termw int
	termh int

	lastx      = coord_invalid
	lasty      = coord_invalid
	cursor_x   = cursor_hidden
	cursor_y   = cursor_hidden
	foreground = ColorDefault
	background = ColorDefault
	intbuf     = make([]byte, 0, 16)
)

func write_cursor(x, y int, c Client) {
	c.Buff.WriteString("\033[")
	c.Buff.Write(strconv.AppendUint(intbuf, uint64(y+1), 10))
	c.Buff.WriteString(";")
	c.Buff.Write(strconv.AppendUint(intbuf, uint64(x+1), 10))
	c.Buff.WriteString("H")

	io.WriteString(c.Conn, c.Buff.String())
}

func send_char(x, y int, ch rune, c Client) {

	var buf [8]byte
	n := utf8.EncodeRune(buf[:], ch)
	if x-1 != lastx || y != lasty {
		write_cursor(x, y, c)
	}
	lastx, lasty = x, y
	c.Buff.Write(buf[:n])

	io.WriteString(c.Conn, c.Buff.String())

}

func send_clear(c *Client) error {

	io.WriteString(c.Conn, "\033[2J")

	if !is_cursor_hidden(cursor_x, cursor_y) {
		write_cursor(cursor_x, cursor_y, *c)
	}

	// we need to invalidate cursor position too and these two vars are
	// used only for simple cursor positioning optimization, cursor
	// actually may be in the correct place, but we simply discard
	// optimization once and it gives us simple solution for the case when
	// cursor moved
	lastx = coord_invalid
	lasty = coord_invalid

	c.Buff.Reset()
	return nil
}

func update_size_maybe(c *Client) error {
	//TODO : get terminal size from client
	termw, termh = 132, 32

	c.Bbuffer.resize(termw, termh)
	c.Fbuffer.resize(termw, termh)
	c.Fbuffer.clear()

	log.Info(fmt.Sprintf("New->W:%d H:%d", termw, termh))
	return send_clear(c)

}

func setup_term() (err error) {

	funcs = make([]string, t_max_funcs)
	funcs[t_max_funcs-2] = ti_mouse_enter
	funcs[t_max_funcs-1] = ti_mouse_leave
	return nil
}
