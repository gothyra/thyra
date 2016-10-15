// +build !windows

package game

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"syscall"
	"unicode/utf8"
	"unsafe"

	log "gopkg.in/inconshreveable/log15.v2"
)

// private API

const (
	t_enter_ca = iota
	t_exit_ca
	t_show_cursor
	t_hide_cursor
	t_clear_screen
	t_sgr0
	t_underline
	t_bold
	t_blink
	t_reverse
	t_enter_keypad
	t_exit_keypad
	t_enter_mouse
	t_exit_mouse
	t_max_funcs
)

const (
	coord_invalid = -2
	attr_invalid  = Attribute(0xFFFF)
)

type input_event struct {
	data []byte
	err  error
}

var (
	// term specific sequences
	keys  []string
	funcs []string

	// termbox inner state
	//orig_tios    syscall_Termios
	back_buffer  Cellbuf
	front_buffer Cellbuf
	termw        int
	termh        int
	//input_mode  = InputEsc
	//output_mode = OutputNormal
	out        *os.File
	in         int
	lastfg     = attr_invalid
	lastbg     = attr_invalid
	lastx      = coord_invalid
	lasty      = coord_invalid
	cursor_x   = cursor_hidden
	cursor_y   = cursor_hidden
	foreground = ColorDefault
	background = ColorDefault
	//	inbuf      = make([]byte, 0, 64)
	//outbuf         Client.Buff
	//sigwinch       = make(chan os.Signal, 1)
	//sigio          = make(chan os.Signal, 1)
	quit           = make(chan int)
	input_comm     = make(chan input_event)
	interrupt_comm = make(chan struct{})
	intbuf         = make([]byte, 0, 16)

	// grayscale indexes
	grayscale = []Attribute{
		0, 17, 233, 234, 235, 236, 237, 238, 239, 240, 241, 242, 243, 244,
		245, 246, 247, 248, 249, 250, 251, 252, 253, 254, 255, 256, 232,
	}
)

func write_cursor(x, y int, c Client) {
	c.Buff.WriteString("\033[")
	c.Buff.Write(strconv.AppendUint(intbuf, uint64(y+1), 10))
	c.Buff.WriteString(";")
	c.Buff.Write(strconv.AppendUint(intbuf, uint64(x+1), 10))
	c.Buff.WriteString("H")

	io.WriteString(c.Conn, c.Buff.String())
}

type winsize struct {
	rows    uint16
	cols    uint16
	xpixels uint16
	ypixels uint16
}

func get_term_size(fd uintptr) (int, int) {
	var sz winsize
	_, _, _ = syscall.Syscall(syscall.SYS_IOCTL,
		fd, uintptr(syscall.TIOCGWINSZ), uintptr(unsafe.Pointer(&sz)))
	return int(sz.cols), int(sz.rows)
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

func flush(c *Client) error {

	//_, err := io.Copy(out, &c.Buff)
	c.Buff.Reset()

	return nil
}

func send_clear(c *Client) error {
	//send_attr(foreground, background, c)
	//io.WriteString(c.Conn, funcs[t_clear_screen])
	//c.Buff.WriteString(funcs[t_clear_screen])
	log.Info("Before CLEAR SCREEN")
	io.WriteString(c.Conn, "\033[2J")
	log.Info("After CLEAR SCREEN")
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

	return flush(c)
}

func update_size_maybe(c *Client) error {
	var err error
	out, err = os.OpenFile("/dev/tty", syscall.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer out.Close()
	termw, termh = get_term_size(out.Fd())

	c.Bbuffer.resize(termw, termh)
	c.Fbuffer.resize(termw, termh)
	c.Fbuffer.clear()

	log.Info(fmt.Sprintf("New->W:%d H:%d", termw, termh))
	return send_clear(c)

}
