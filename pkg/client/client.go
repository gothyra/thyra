package client

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
	log "gopkg.in/inconshreveable/log15.v2"

	"github.com/gothyra/thyra/pkg/area"
)

const (
	ti_mouse_enter = "\x1b[?1000h\x1b[?1002h\x1b[?1015h\x1b[?1006h"
	ti_mouse_leave = "\x1b[?1006l\x1b[?1015l\x1b[?1002l\x1b[?1000l"

	cursorHidden = -1
	coordInvalid = -2
	attrInvalid  = Attribute(0xFFFF)

	tEnterCa = iota
	tExitCa
	tClearScreen
	tMaxFuncs

	editBoxWidth = 120
)

type Reply struct {
	World  []byte
	Events string
	Intro  []byte
	Exits  string
}

type Event struct {
	Client *Client
	Etype  string
}

type Request struct {
	Client *Client
	Cmd    string
}

type LoginRequest struct {
	Username string
	Conn     net.Conn
	Reply    chan bool
}

type Clients []Client

func (cl Clients) String() string {
	var clients []string
	for _, c := range cl {
		clients = append(clients, c.Player.Nickname)
	}
	return fmt.Sprintf("%#v", clients)
}

type Client struct {
	// Conn is the connection used by the user to play.
	Conn net.Conn
	// Player holds all the necessary information for the character of the user.
	Player *area.Player
	// Request is used by the user to send command at the server.
	Request chan<- Request
	// Reply is used by the server to respond to user commands. The Reply channel is
	// operated by the Panel thread which runs in parallel with the main client thread
	// and is responsible for updating the output users see.
	Reply chan Reply

	Buff    bytes.Buffer
	Bbuffer *Cellbuf
	Fbuffer *Cellbuf
	intbuf  []byte

	// termW is the terminal width of this client
	termW int
	// termH is the terminal height of this client
	termH int

	lastx      int
	lasty      int
	cursorX    int
	cursorY    int
	foreground Attribute
	background Attribute
}

func NewClient(c net.Conn, player *area.Player, req chan<- Request) *Client {
	return &Client{
		Conn:    c,
		Player:  player,
		Request: req,
		Reply:   make(chan Reply, 1),

		Bbuffer: new(Cellbuf),
		Fbuffer: new(Cellbuf),
		intbuf:  make([]byte, 0, 16),

		// TODO: Get these from the remote client
		termW: 132,
		termH: 32,

		lastx:      coordInvalid,
		lasty:      coordInvalid,
		cursorX:    cursorHidden,
		cursorY:    cursorHidden,
		foreground: ColorDefault,
		background: ColorDefault,
	}
}

// Redraw should be run as a separate goroutine in parallel with ReadLinesInto.
// This function is responsible for returning output to the user.
func (c *Client) Redraw(wg *sync.WaitGroup, quit <-chan struct{}) {
	defer wg.Done()
	defer io.WriteString(c.Conn, "\033[2J")

	c.initScreen()

	for {
		select {
		case reply := <-c.Reply:
			c.redraw(reply)
		case <-quit:
			log.Warn(fmt.Sprintf("Panel for %q quit", c.Player.Nickname))
			return
		}
	}
}

func (c *Client) initScreen() {
	funcs := make([]string, tMaxFuncs)
	funcs[tMaxFuncs-2] = ti_mouse_enter
	funcs[tMaxFuncs-1] = ti_mouse_leave

	io.WriteString(c.Conn, funcs[tEnterCa])
	io.WriteString(c.Conn, funcs[tClearScreen])

	c.Bbuffer = New(c.termW, c.termH, c.foreground, c.background)
	c.Fbuffer = New(c.termW, c.termH, c.foreground, c.background)
}

// TOOD: A huge comment is needed here about what exactly redraw is doing
func (c *Client) redraw(reply Reply) {
	log.Debug(fmt.Sprintf("Redraw: %s, W: %d H: %d ", c.Player.Nickname, c.Bbuffer.Width, c.Bbuffer.Height))

	// TODO: Ti einai oloi autoi oi magikoi arithmoi?
	// Prepei na tous niwsoume
	midy := c.termH/2 + 12
	midx := (c.termW - editBoxWidth) - 10

	c.clearScreen(ColorDefault, ColorDefault)

	// Fill up backBuffer
	c.setCell(midx-1, midy, '│', ColorDefault, ColorDefault)
	c.setCell(midx+editBoxWidth, midy, '│', ColorDefault, ColorDefault)
	c.setCell(midx-1, midy-1, '┌', ColorDefault, ColorDefault)
	c.setCell(midx-1, midy+1, '└', ColorDefault, ColorDefault)
	c.setCell(midx+editBoxWidth, midy-1, '┐', ColorDefault, ColorDefault)
	c.setCell(midx+editBoxWidth, midy+1, '┘', ColorDefault, ColorDefault)
	c.fill(midx, midy-1, editBoxWidth, 1, Cell{Ch: '─'})
	c.fill(midx, midy+1, editBoxWidth, 1, Cell{Ch: '─'})

	// setCursor writes to the connection!
	c.setCursor(midx, midy)

	counter := 20
	buf := bytes.NewBuffer(reply.World)
	for {
		line, err := buf.ReadString('\n')
		if err != nil {
			// TODO: Log errors other than io.EOF
			// log.Info("world buffer read error: %v", err)
			break
		}
		c.tbprint(midx+100, midy-counter, ColorDefault, ColorDefault, line)
		counter--
	}

	counter2 := 20
	rintro := bytes.NewBuffer(reply.Intro)
	for {
		line, err := rintro.ReadString('\n')
		if err != nil {
			// TODO: Log errors other than io.EOF
			// log.Error("intro buffer read error: %v", err)
			break
		}
		c.tbprint(midx, midy-counter2, ColorDefault, ColorDefault, line)
		counter2--
	}

	c.tbprint(midx, midy-10, ColorDefault, ColorDefault, reply.Events)
	c.tbprint(midx+90, midy-3, ColorDefault, ColorDefault, reply.Exits)

	// So far we have been filling backBuffer; i guess now it's time to flush the content
	// to the user.
	c.flush()
}

// Clears the internal back buffer.
func (c *Client) clearScreen(fg, bg Attribute) {
	c.foreground, c.background = fg, bg
	c.Bbuffer.resize(c.termW, c.termH, c.foreground, c.background)
	c.Fbuffer.resize(c.termW, c.termH, c.foreground, c.background)
	c.Fbuffer.initialized(c.foreground, c.background)
	io.WriteString(c.Conn, "\033[2J")

	if !isCursorHidden(c.cursorX, c.cursorY) {
		c.writeCursor(c.cursorX, c.cursorY)
	}

	// we need to invalidate cursor position too and these two vars are
	// used only for simple cursor positioning optimization, cursor
	// actually may be in the correct place, but we simply discard
	// optimization once and it gives us simple solution for the case when
	// cursor moved
	c.lastx = coordInvalid
	c.lasty = coordInvalid

	c.Buff.Reset()
	c.Bbuffer.initialized(c.foreground, c.background)
}

// TOOD: A comment is needed here about what exactly writeCursor is doing
func (c *Client) writeCursor(x, y int) {
	c.Buff.WriteString("\033[")
	c.Buff.Write(strconv.AppendUint(c.intbuf, uint64(y+1), 10))
	c.Buff.WriteString(";")
	c.Buff.Write(strconv.AppendUint(c.intbuf, uint64(x+1), 10))
	c.Buff.WriteString("H")

	io.WriteString(c.Conn, c.Buff.String())
}

// Changes cell's parameters in the internal back buffer at the specified
// position.
func (c *Client) setCell(x, y int, ch rune, fg, bg Attribute) {
	if x < 0 || x >= c.Bbuffer.Width {
		return
	}
	if y < 0 || y >= c.Bbuffer.Height {
		return
	}

	c.Bbuffer.Cells[y*c.Bbuffer.Width+x] = Cell{ch, fg, bg}
}

// TOOD: A comment is needed here about what exactly fill is doing
func (c *Client) fill(x, y, w, h int, cell Cell) {
	for ly := 0; ly < h; ly++ {
		for lx := 0; lx < w; lx++ {
			c.setCell(x+lx, y+ly, cell.Ch, cell.Fg, cell.Bg)
		}
	}
}

// Sets the position of the cursor. See also HideCursor().
func (c *Client) setCursor(x, y int) {
	c.cursorX, c.cursorY = x, y
	if !isCursorHidden(c.cursorX, c.cursorY) {
		c.writeCursor(c.cursorX, c.cursorY)
	}
}

func (c *Client) tbprint(x, y int, fg, bg Attribute, msg string) {
	for _, rune := range msg {
		c.setCell(x, y, rune, fg, bg)
		x += runewidth.RuneWidth(rune)
	}
}

// Synchronizes the internal back buffer with the terminal.
// TOOD: A huge comment is needed here about what exactly flush is doing
func (c *Client) flush() {
	// invalidate cursor position
	c.lastx = coordInvalid
	c.lasty = coordInvalid

	log.Debug(fmt.Sprintf("Flush Before FOR : %s", c.Player.Nickname))

	width := c.Fbuffer.Width
	height := c.Fbuffer.Height

	for y := 0; y < height; y++ {

		lineOffset := y * width

		for x := 0; x < width; {
			cellOffset := lineOffset + x
			back := c.Bbuffer.Cells[cellOffset]
			front := c.Fbuffer.Cells[cellOffset]
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
				c.sendChar(x, y, ' ')

			} else {
				c.sendChar(x, y, back.Ch)
				if w == 2 {
					next := cellOffset + 1
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
	log.Debug(fmt.Sprintf("Flush After FOR : %s", c.Player.Nickname))

	if !isCursorHidden(c.cursorX, c.cursorY) {
		c.writeCursor(c.cursorX, c.cursorY)
	}
	log.Debug(fmt.Sprintf("Flush End  : %s", c.Player.Nickname))
	c.Buff.Reset()
}

// ReadLinesInto accepts input from a user and sends it back to the server in the
// form of requests.
// TODO: Make this exit gracefully on a server shutdown.
func (c Client) ReadLinesInto(quit <-chan struct{}) {
	bufc := bufio.NewReader(c.Conn)

	for {
		line, err := bufc.ReadString('\n')
		if err != nil {
			log.Error(fmt.Sprintf("%#v", err))
			return
		}
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		select {
		case c.Request <- Request{Client: &c, Cmd: line}:
		case <-quit:
			log.Info(fmt.Sprintf("Player %q quit", c.Player.Nickname))
			return
		}
	}
}

func isCursorHidden(x, y int) bool {
	return x == cursorHidden || y == cursorHidden
}

// TOOD: A comment is needed here about what exactly sendChar is doing
func (c *Client) sendChar(x, y int, ch rune) {
	var buf [8]byte
	n := utf8.EncodeRune(buf[:], ch)
	if x-1 != c.lastx || y != c.lasty {
		c.writeCursor(x, y)
	}
	c.lastx, c.lasty = x, y
	c.Buff.Write(buf[:n])

	io.WriteString(c.Conn, c.Buff.String())
}

// The shortcut for SetCursor(-1, -1).
func (c *Client) hideCursor() {
	c.setCursor(cursorHidden, cursorHidden)
}
