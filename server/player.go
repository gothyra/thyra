package server

import (
	"fmt"
	"math"
	"strings"
	"sync"

	"github.com/jpillora/ansi"
	"golang.org/x/crypto/ssh"
	log "gopkg.in/inconshreveable/log15.v2"
)

const (
	ARROW_UP = iota + 65
	ARROW_DOWN
	ARROW_RIGHT
	ARROW_LEFT
)

const (
	ENTER_KEY     = 13
	SPACE_KEY     = 32
	BACKSPACE_KEY = 127
)

const (
	NUM_0 = iota + 48
	NUM_1
	NUM_2
	NUM_3
	NUM_4
	NUM_5
	NUM_6
	NUM_7
	NUM_8
	NUM_9
)

const (
	LOW_ALPHA = 97
	LOW_OMEGA = 122
)

const (
	UPPER_ALPHA = 65
	UPPER_OMEGA = 90
)

var (
	alphabet = []string{
		"a",
		"b",
		"c",
		"d",
		"e",
		"f",
		"g",
		"h",
		"i",
		"j",
		"k",
		"l",
		"m",
		"n",
		"o",
		"p",
		"q",
		"r",
		"s",
		"t",
		"u",
		"v",
		"w",
		"x",
		"y",
		"z"}
)

type resize struct {
	width, height uint32
}

// A Player represents a live TCP connection from a client
type Player struct {
	id                   ID     // identification
	hash                 string //hash of public key
	SSHName, Name, cname string
	rank, index          int
	x, y                 uint8    // position
	w, h                 int      // terminal size
	screenRunes          [][]rune // the player's view of the screen
	screenColors         [][]ID   // the player's view of the screen
	ready                bool
	resizes              chan resize
	conn                 *ansi.Ansi
	logf                 func(format string, args ...interface{})
	once                 *sync.Once
	command              []string
	commandHistory       []string
	rollback             int
	wantHistory          bool
	fromHistoryToCmd     string
	finalCmd             string
}

// NewPlayer returns an initialized Player.
func NewPlayer(id ID, sshName, name, hash string, conn ssh.Channel) *Player {
	if hash == "" {
		hash = name //finally, hash fallsback to name
	}
	p := &Player{
		id:             id,
		hash:           hash,
		SSHName:        sshName,
		Name:           name,
		ready:          false,
		resizes:        make(chan resize),
		conn:           ansi.Wrap(conn),
		once:           &sync.Once{},
		command:        make([]string, 1),
		commandHistory: make([]string, 1),
		rollback:       0,
		wantHistory:    false,
	}
	return p
}

var resizeTmpl = string(ansi.Goto(2, 5)) +
	string(ansi.Set(ansi.Blue)) +
	"Please resize your terminal to %dx%d (+%dx+%d)"

func (p *Player) resizeWatch() {
	for r := range p.resizes {

		p.w = int(r.width)
		p.h = int(r.height)
		log.Info(fmt.Sprintf("Width :%d  Height:%d", p.w, p.h))

		// fits?
		if p.w >= 10 && p.h >= 10 {
			p.conn.EraseScreen()
			// send updates!
			p.ready = true
		} else {
			// doesnt fit
			p.conn.EraseScreen()
			p.conn.Write([]byte(fmt.Sprintf(resizeTmpl, 10, 10,
				int(math.Max(float64(10-p.w), 0)),
				int(math.Max(float64(10-p.h), 0)))))
			p.screenRunes = nil
			p.ready = false
		}
	}
}

func (p *Player) promptBar(s *Server) {

	buff := make([]byte, 3)
	for {
		//log.Debug(fmt.Sprintf("read buff is : %v", buff))
		n, err := p.conn.Read(buff)

		if err != nil {
			break
		}
		b := buff[:n]
		if b[0] == 3 {
			break
		}

		// Ignore until terminal size is more than requested.
		if !p.ready {
			continue
		}

		// Parse Arrows
		if len(b) == 3 && b[0] == ansi.Esc && b[1] == 91 {
			cursorBehavor := []byte{0, 0, 0}
			switch c := b[2]; {

			// We use ARROW_UP to go back in command history.
			case c == ARROW_UP:
				//cursorBehavor = []byte{ansi.Esc, 91, 65}
				p.wantHistory = true
				p.arrowUp()

			// We use ARROW_DOWN to go forward in command history.
			case c == ARROW_DOWN:
				//cursorBehavor = []byte{ansi.Esc, 91, 66}
				p.wantHistory = true
				p.arrowDown()

			// TODO : User right -left to insert - delete the p.command
			case c == ARROW_RIGHT:
				cursorBehavor = []byte{ansi.Esc, 91, 67}
			case c == ARROW_LEFT:
				cursorBehavor = []byte{ansi.Esc, 91, 68}
			}
			p.conn.Write(cursorBehavor)
		} else {
			p.rollback = 0
		}

		switch n := b[0]; {

		// Check uppercase letters
		case n >= UPPER_ALPHA && n <= UPPER_OMEGA:
			num := b[0] - 65
			p.conn.Write([]byte(strings.ToUpper(alphabet[num])))
			p.command = append(p.command, strings.ToUpper(alphabet[num]))

		// Check for lowercase letters
		case n >= LOW_ALPHA && n <= LOW_OMEGA:
			num := b[0] - 97
			p.conn.Write([]byte(alphabet[num]))
			p.command = append(p.command, alphabet[num])

		// Check for numbers
		case n >= NUM_0 && n <= NUM_9:
			num := b[0] - 48
			p.conn.Write([]byte(fmt.Sprintf("%d", num)))
			p.command = append(p.command, fmt.Sprintf("%d", num))

		// Enter key
		case n == ENTER_KEY:
			p.enterKey(s)
		// Space key
		case n == SPACE_KEY:
			p.conn.Write([]byte(" "))
			p.command = append(p.command, " ")

		// Backspace key
		// TODO : Backspace is deleting the p.command. Make this work.!
		case n == BACKSPACE_KEY:
			backSpace := []byte{27, 91, 68}
			p.conn.Write([]byte("\b "))
			p.conn.Write([]byte(backSpace))

		//  Key ] only for debuging purpose.
		case n == 93:
			p.drawPromptBar()
			log.Info(fmt.Sprintf("%#v", p.commandHistory))
			log.Info(fmt.Sprintf("%#v", p.command))
		}
	}
}

func (p *Player) getCommandAsString() string {
	cmd := ""
	for i := range p.command {
		cmd += p.command[i]
	}
	return cmd
}

func (p *Player) fillPromptBar() string {
	promptBar := ""
	for i := 0; i < p.w; i++ {
		promptBar += "-"
	}
	return promptBar
}

func (p *Player) drawPromptBar() {
	p.conn.Write([]byte(string(ansi.Goto(uint16(p.h)-2, 1)) + p.fillPromptBar()))
	p.conn.Write([]byte(string(ansi.Goto(uint16(p.h), 1)) + p.fillPromptBar()))
	p.conn.Write(ansi.Goto(uint16(p.h)-1, 1))
}

// TODO : Split the string from commandHistory to p.command array
func (p *Player) arrowUp() {

	p.conn.Write(ansi.EraseLine)
	p.conn.Write(ansi.Goto(uint16(p.h-1), 1))

	p.rollback++

	log.Debug(fmt.Sprintf("Len %d , Rollback %d", len(p.commandHistory), p.rollback))
	if len(p.commandHistory)-p.rollback > 0 {
		if len(p.commandHistory)-p.rollback == 1 {
			p.rollback--
			p.conn.Write([]byte(p.commandHistory[1]))
			p.fromHistoryToCmd = p.commandHistory[1]

		} else if len(p.commandHistory)-p.rollback > 1 {
			p.conn.Write([]byte(p.commandHistory[len(p.commandHistory)-p.rollback]))
			p.fromHistoryToCmd = p.commandHistory[len(p.commandHistory)-p.rollback]
		}
	} else {
		log.Debug("No command history")
		p.wantHistory = false
		p.rollback--
	}
}

// TODO : Split the string from commandHistory to p.command array
func (p *Player) arrowDown() {

	p.conn.Write(ansi.EraseLine)
	p.conn.Write(ansi.Goto(uint16(p.h-1), 1))
	p.rollback--

	log.Debug(fmt.Sprintf("Len %d , Rollback %d", len(p.commandHistory), p.rollback))
	if len(p.commandHistory)-p.rollback < len(p.commandHistory) {
		if len(p.commandHistory)-p.rollback == 1 {
			p.rollback++
			p.conn.Write([]byte(p.commandHistory[1]))
			p.fromHistoryToCmd = p.commandHistory[1]
		} else {
			p.conn.Write([]byte(p.commandHistory[len(p.commandHistory)-p.rollback]))
			p.fromHistoryToCmd = p.commandHistory[len(p.commandHistory)-p.rollback]
		}
	} else {
		log.Debug("No command history")
		p.wantHistory = false
		p.rollback++
	}
}

// This function sends an event to s.Events channel.
// GOD thread will handle those events.
func (p *Player) enterKey(s *Server) {
	p.conn.Write(ansi.EraseLine)
	p.conn.Write(ansi.Goto(uint16(p.h)-1, 1))
	p.conn.Write(ansi.CursorHide)

	if p.wantHistory && p.fromHistoryToCmd != "" {
		p.wantHistory = false
		p.finalCmd = p.fromHistoryToCmd

	} else {
		p.finalCmd = p.getCommandAsString()
	}
	p.commandHistory = append(p.commandHistory, p.finalCmd)

	for _, onlineClient := range s.onlinePlayers {
		onlineClient.conn.Write([]byte(string(ansi.Goto(uint16(onlineClient.h-onlineClient.h+s.lines), 1)) + p.Name + " : " + p.finalCmd))
		onlineClient.conn.Write(ansi.Goto(uint16(onlineClient.h+s.lines), 1))
	}

	s.lines++
	p.conn.Write(ansi.CursorShow)

	p.drawPromptBar()
	event := Event{Player: p, EventType: p.getCommandAsString()}
	s.Events <- event
	// Clear command array to re-use it again.
	p.command = []string{}
}
