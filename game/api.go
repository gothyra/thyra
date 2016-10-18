package game

import (
	"fmt"
	"io"

	runewidth "github.com/mattn/go-runewidth"

	log "gopkg.in/inconshreveable/log15.v2"
)

func Init(c *Client) error {

	var err error

	err = setup_term()
	if err != nil {
		return fmt.Errorf("termbox: error while reading terminfo data: %v", err)
	}

	io.WriteString(c.Conn, funcs[t_enter_ca])
	io.WriteString(c.Conn, funcs[t_clear_screen])

	//TODO : get terminal size from Client
	termw, termh = 132, 32

	log.Info(fmt.Sprintf("TermW: %d, TermH: %d ", termw, termh))

	backb := New(termw, termh)
	frontb := New(termw, termh)

	log.Info("Clear Start")
	backb.clear()
	frontb.clear()

	c.Bbuffer = backb
	c.Fbuffer = frontb
	log.Info(fmt.Sprintf("Init OK: %s", c.Player.Nickname))
	return nil
}

func Close(c Client) {
	io.WriteString(c.Conn, "\033[2J")

	// reset the state, so that on next Init() it will work again
	termw = 0
	termh = 0
	//input_mode = InputEsc

	lastx = coord_invalid
	lasty = coord_invalid
	cursor_x = cursor_hidden
	cursor_y = cursor_hidden
}

// Synchronizes the internal back buffer with the terminal.
func Flush(c *Client) error {
	// invalidate cursor position
	lastx = coord_invalid
	lasty = coord_invalid

	//update_size_maybe(c)
	log.Info("Before FOR")

	width := c.Fbuffer.Width
	height := c.Fbuffer.Height

	for y := 0; y < height; y++ {

		line_offset := y * width

		for x := 0; x < width; {
			cell_offset := line_offset + x
			back := c.Bbuffer.Cells[cell_offset]
			front := c.Fbuffer.Cells[cell_offset]
			if back.Ch < ' ' {
				back.Ch = ' '
			}
			w := runewidth.RuneWidth(back.Ch)

			if w == 0 || w == 2 && runewidth.IsAmbiguousWidth(back.Ch) {
				w = 1
			}
			if back == front {
				x += w
				continue
			}
			front = back

			if w == 2 && x == width-1 {

				// there's not enough space for 2-cells rune,
				// let's just put a space in there
				send_char(x, y, ' ', *c)

			} else {
				send_char(x, y, back.Ch, *c)
				if w == 2 {
					next := cell_offset + 1
					c.Fbuffer.Cells[next] = Cell{
						Ch: 0,
						Fg: back.Fg,
						Bg: back.Bg,
					}
				}
			}

			x += w
		}
	}
	log.Info("After FOR")

	if !is_cursor_hidden(cursor_x, cursor_y) {
		write_cursor(cursor_x, cursor_y, *c)
	}
	log.Info(fmt.Sprintf("Flush: %s", c.Player.Nickname))
	c.Buff.Reset()
	return nil
}

// Sets the position of the cursor. See also HideCursor().
func SetCursor(x, y int, c Client) {
	cursor_x, cursor_y = x, y
	if !is_cursor_hidden(cursor_x, cursor_y) {
		write_cursor(cursor_x, cursor_y, c)
	}
}

// The shortcut for SetCursor(-1, -1).
func HideCursor(c Client) {
	SetCursor(cursor_hidden, cursor_hidden, c)
}

// Changes cell's parameters in the internal back buffer at the specified
// position.
func SetCell(x, y int, ch rune, fg, bg Attribute, c *Client) {
	if x < 0 || x >= c.Bbuffer.Width {
		return
	}
	if y < 0 || y >= c.Bbuffer.Height {
		return
	}

	c.Bbuffer.Cells[y*c.Bbuffer.Width+x] = Cell{ch, fg, bg}
}

func Size() (width int, height int) {
	return termw, termh
}

// Clears the internal back buffer.
func Clear(fg, bg Attribute, c *Client) error {
	foreground, background = fg, bg
	err := update_size_maybe(c)
	c.Bbuffer.clear()

	return err
}
