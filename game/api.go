package game

import (
	"fmt"
	"io"
	"os"
	"syscall"

	"github.com/mattn/go-runewidth"
)

func Init(c Client) error {
	var err error

	out, err = os.OpenFile("/dev/tty", syscall.O_WRONLY, 0)
	if err != nil {
		return err
	}
	in, err = syscall.Open("/dev/tty", syscall.O_RDONLY, 0)
	if err != nil {
		return err
	}

	err = setup_term(c)
	if err != nil {
		return fmt.Errorf("termbox: error while reading terminfo data: %v", err)
	}

	io.WriteString(c.Conn, funcs[t_enter_ca])
	io.WriteString(c.Conn, funcs[t_enter_keypad])
	io.WriteString(c.Conn, funcs[t_hide_cursor])
	io.WriteString(c.Conn, funcs[t_clear_screen])

	termw, termh = get_term_size(out.Fd())
	back_buffer.init(termw, termh)
	front_buffer.init(termw, termh)
	back_buffer.clear()
	front_buffer.clear()

	return nil
}

// Interrupt an in-progress call to PollEvent by causing it to return
// EventInterrupt.  Note that this function will block until the PollEvent
// function has successfully been interrupted.
func Interrupt() {
	interrupt_comm <- struct{}{}
}

// Finalizes termbox library, should be called after successful initialization
// when termbox's functionality isn't required anymore.
func Close(c Client) {
	quit <- 1
	io.WriteString(c.Conn, funcs[t_show_cursor])
	io.WriteString(c.Conn, funcs[t_sgr0])
	io.WriteString(c.Conn, funcs[t_clear_screen])
	io.WriteString(c.Conn, funcs[t_exit_ca])
	io.WriteString(c.Conn, funcs[t_exit_keypad])
	io.WriteString(c.Conn, funcs[t_exit_mouse])
	//tcsetattr(out.Fd(), &orig_tios)

	out.Close()
	//syscall.Close(in)

	// reset the state, so that on next Init() it will work again
	termw = 0
	termh = 0
	//input_mode = InputEsc
	out = nil
	in = 0
	lastfg = attr_invalid
	lastbg = attr_invalid
	lastx = coord_invalid
	lasty = coord_invalid
	cursor_x = cursor_hidden
	cursor_y = cursor_hidden
	//foreground = ColorDefault
	//background = ColorDefault
	//IsInit = false
}

// Synchronizes the internal back buffer with the terminal.
func Flush(c Client) error {
	// invalidate cursor position
	lastx = coord_invalid
	lasty = coord_invalid

	update_size_maybe(c)

	for y := 0; y < front_buffer.height; y++ {
		line_offset := y * front_buffer.width
		for x := 0; x < front_buffer.width; {
			cell_offset := line_offset + x
			back := &back_buffer.cells[cell_offset]
			front := &front_buffer.cells[cell_offset]
			if back.Ch < ' ' {
				back.Ch = ' '
			}
			w := runewidth.RuneWidth(back.Ch)
			if w == 0 || w == 2 && runewidth.IsAmbiguousWidth(back.Ch) {
				w = 1
			}
			if *back == *front {
				x += w
				continue
			}
			*front = *back
			//send_attr(back.Fg, back.Bg, c)

			if w == 2 && x == front_buffer.width-1 {
				// there's not enough space for 2-cells rune,
				// let's just put a space in there
				send_char(x, y, ' ', c)
			} else {
				send_char(x, y, back.Ch, c)
				if w == 2 {
					next := cell_offset + 1
					front_buffer.cells[next] = Cell{
						Ch: 0,
						Fg: back.Fg,
						Bg: back.Bg,
					}
				}
			}

			x += w
		}
	}
	if !is_cursor_hidden(cursor_x, cursor_y) {
		write_cursor(cursor_x, cursor_y, c)
	}

	return flush(c)
}

// Sets the position of the cursor. See also HideCursor().
func SetCursor(x, y int, c Client) {
	if is_cursor_hidden(cursor_x, cursor_y) && !is_cursor_hidden(x, y) {
		io.WriteString(c.Conn, funcs[t_show_cursor])
	}

	if !is_cursor_hidden(cursor_x, cursor_y) && is_cursor_hidden(x, y) {
		io.WriteString(c.Conn, funcs[t_hide_cursor])
	}

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
func SetCell(x, y int, ch rune, fg, bg Attribute, c Client) {

	if x < 0 || x >= back_buffer.width {
		return
	}
	if y < 0 || y >= back_buffer.height {
		return
	}

	back_buffer.cells[y*back_buffer.width+x] = Cell{ch, fg, bg}

}

// Returns a slice into the termbox's back buffer. You can get its dimensions
// using 'Size' function. The slice remains valid as long as no 'Clear' or
// 'Flush' function calls were made after call to this function.
func CellBuffer(c Client) []Cell {
	return back_buffer.cells
}

// Returns the size of the internal back buffer (which is mostly the same as
// terminal's window size in characters). But it doesn't always match the size
// of the terminal window, after the terminal size has changed, the internal
// back buffer will get in sync only after Clear or Flush function calls.
func Size() (width int, height int) {
	return termw, termh
}

// Clears the internal back buffer.
func Clear(fg, bg Attribute, c Client) error {

	foreground, background = fg, bg
	err := update_size_maybe(c)
	back_buffer.clear()

	return err
}

// Sync comes handy when something causes desync between termbox's understanding
// of a terminal buffer and the reality. Such as a third party process. Sync
// forces a complete resync between the termbox and a terminal, it may not be
// visually pretty though.

func Sync(c Client) error {

	back_buffer.clear()
	err := send_clear(c)
	if err != nil {
		return err
	}

	return Flush(c)
}
