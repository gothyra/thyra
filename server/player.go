package server

import (
	"fmt"
	"math"
	"strings"

	"github.com/jpillora/ansi"
	"golang.org/x/crypto/ssh"
	log "gopkg.in/inconshreveable/log15.v2"
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
	command              []string
	commandHistory       []string
	rollback             int
	wantHistory          bool
	position             int
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
		command:        make([]string, 0),
		commandHistory: make([]string, 1),
		rollback:       0,
		wantHistory:    false,
		position:       0,
	}
	return p
}

var resizeTmpl = string(ansi.Goto(2, 5)) +
	string(ansi.Set(ansi.Blue)) +
	"Please resize your terminal to %dx%d (+%dx+%d)"

func (p *Player) resetScreen() {
	p.screenRunes = make([][]rune, p.w)
	p.screenColors = make([][]ID, p.w)
	for w := 0; w < p.w; w++ {
		p.screenRunes[w] = make([]rune, p.h)
		p.screenColors[w] = make([]ID, p.h)
		for h := 0; h < p.h; h++ {
			p.screenRunes[w][h] = 'x'
			p.screenColors[w][h] = ID(255)
		}
	}
}

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
		log.Debug(fmt.Sprintf("read buff is : %v", buff))
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
				p.wantHistory = true
				p.arrowUp()

			// We use ARROW_DOWN to go forward in command history.
			case c == ARROW_DOWN:
				p.wantHistory = true
				p.arrowDown()

			// We use ARROW_RIGHT to move right through the command for backspace and delete purpose.
			case c == ARROW_RIGHT:
				if len(p.command) > p.position {
					cursorBehavor = []byte{ansi.Esc, 91, 67}
					p.position++
				}
			// We use ARROW_RIGHT to move left through the command for backspace and delete purpose.
			case c == ARROW_LEFT:
				if len(p.command) > 0 {
					cursorBehavor = []byte{ansi.Esc, 91, 68}
					p.position--
				}
			}
			p.conn.Write(cursorBehavor)
		} else {
			p.rollback = 0
		}

		switch n := b[0]; {

		// Check Special chars 1st part
		case n >= 33 && n <= 47:
			num := b[0] - 33
			p.conn.Write([]byte(specialChars1[num]))
			p.command = append(p.command, specialChars1[num])
			p.position++

			// Check Special chars 2nd part
		case n >= 58 && n <= 64:
			num := b[0] - 58
			p.conn.Write([]byte(specialChars2[num]))
			p.command = append(p.command, specialChars2[num])
			p.position++

			// Check Special chars 3rd part
		case n >= 91 && n <= 96:
			num := b[0] - 91
			p.conn.Write([]byte(specialChars3[num]))
			p.command = append(p.command, specialChars3[num])
			p.position++

			// Check Special chars 4th part
		case n >= 123 && n <= 126:
			num := b[0] - 123
			p.conn.Write([]byte(specialChars4[num]))
			p.command = append(p.command, specialChars4[num])
			p.position++

		// Check uppercase letters
		case n >= UPPER_ALPHA && n <= UPPER_OMEGA:
			num := b[0] - 65
			p.conn.Write([]byte(strings.ToUpper(alphabet[num])))
			p.command = append(p.command, strings.ToUpper(alphabet[num]))
			p.position++

		// Check for lowercase letters
		case n >= LOW_ALPHA && n <= LOW_OMEGA:
			num := b[0] - 97
			p.conn.Write([]byte(alphabet[num]))
			p.command = append(p.command, alphabet[num])
			p.position++

		// Check for numbers
		case n >= NUM_0 && n <= NUM_9:
			num := b[0] - 48
			p.conn.Write([]byte(fmt.Sprintf("%d", num)))
			p.command = append(p.command, fmt.Sprintf("%d", num))
			p.position++

		// Enter key
		case n == ENTER_KEY:
			if len(p.command) > 0 {
				p.enterKey(s)
			}
		// Space key
		case n == SPACE_KEY:
			p.position++
			if p.position < len(p.command) {
				p.command = InsertInSlice(p.command, p.position-1, " ")
				p.clearPromptBar()
				p.conn.Write([]byte(p.getCommandAsString()))
				p.conn.Write(ansi.Goto(uint16(p.h)-1, uint16(p.position)+1))

			} else {
				p.conn.Write([]byte(" "))
				p.command = append(p.command, " ")
			}

		// Backspace key
		case n == BACKSPACE_KEY:
			if p.position > 0 {
				p.deletePartofCommand(p.position - 1)
				p.position--
				p.clearPromptBar()
				p.conn.Write([]byte(p.getCommandAsString()))
				p.conn.Write(ansi.Goto(uint16(p.h)-1, uint16(p.position)+1))
			}
			// Delete Key
		case n == DELETE_KEY && b[2] == 51:
			if p.position < len(p.command) {
				p.deletePartofCommand(p.position)
				p.clearPromptBar()
				p.conn.Write([]byte(p.getCommandAsString()))
				p.conn.Write(ansi.Goto(uint16(p.h)-1, uint16(p.position)+1))
			}

		//  Key ] only for debuging purpose.
		case n == 93:
			log.Info(fmt.Sprintf("%#v", p.commandHistory))
			log.Info(fmt.Sprintf("%#v", p.command))

		}
		//log.Debug(fmt.Sprintf("Position %d", p.position))
	}
}

func (p *Player) getCommandAsString() string {
	cmd := ""
	for i := range p.command {
		cmd += p.command[i]
	}
	return cmd
}

func (p *Player) convertCommadHistoryToArray(command string) {
	for i := 0; i < len(command); i++ {
		p.command = append(p.command, string(command[i]))
	}
}

func (p *Player) fillPromptBar() string {
	promptBar := ""
	for i := 0; i < p.w; i++ {
		promptBar += "~"
	}
	return promptBar
}

func (p *Player) drawPromptBar() {
	p.conn.Write([]byte(string(ansi.Goto(uint16(p.h)-2, 1)) + p.fillPromptBar()))
	p.conn.Write([]byte(string(ansi.Goto(uint16(p.h), 1)) + p.fillPromptBar()))
	p.conn.Write(ansi.Goto(uint16(p.h)-1, 1))
}

// Travel backwards through the history of commands
func (p *Player) arrowUp() {
	p.clearPromptBar()
	p.rollback++

	log.Debug(fmt.Sprintf("Len %d , Rollback %d", len(p.commandHistory), p.rollback))
	if len(p.commandHistory)-p.rollback > 0 {
		// Clear command array to re-use it again.
		p.command = []string{}
		if len(p.commandHistory)-p.rollback == 1 {
			p.rollback--
			p.conn.Write([]byte(p.commandHistory[1]))
			p.convertCommadHistoryToArray(p.commandHistory[1])
			p.position = len(p.command)

		} else if len(p.commandHistory)-p.rollback > 1 {
			p.conn.Write([]byte(p.commandHistory[len(p.commandHistory)-p.rollback]))
			p.convertCommadHistoryToArray(p.commandHistory[len(p.commandHistory)-p.rollback])
			p.position = len(p.command)

		}
	} else {
		log.Debug("No command history")
		p.wantHistory = false
		p.rollback--
	}
}

// Travel forwards through the history of commands
func (p *Player) arrowDown() {
	p.clearPromptBar()
	p.rollback--

	log.Debug(fmt.Sprintf("Len %d , Rollback %d", len(p.commandHistory), p.rollback))
	if len(p.commandHistory)-p.rollback < len(p.commandHistory) {
		// Clear command array to re-use it again.
		p.command = []string{}
		if len(p.commandHistory)-p.rollback == 1 {
			p.rollback++
			p.conn.Write([]byte(p.commandHistory[1]))
			p.convertCommadHistoryToArray(p.commandHistory[1])
			p.position = len(p.command)
		} else {
			p.conn.Write([]byte(p.commandHistory[len(p.commandHistory)-p.rollback]))
			p.convertCommadHistoryToArray(p.commandHistory[len(p.commandHistory)-p.rollback])
			p.position = len(p.command)
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
	p.clearPromptBar()
	p.conn.Write(ansi.CursorHide)

	p.commandHistory = append(p.commandHistory, p.getCommandAsString())

	for _, onlineClient := range s.onlinePlayers {
		if s.lines == onlineClient.h-2 {
			onlineClient.conn.Write(ansi.EraseScreen)
			s.lines = 1
		}
		onlineClient.conn.Write([]byte(string(ansi.Goto(uint16(onlineClient.h-onlineClient.h+s.lines), 1)) + p.Name + " : " + p.getCommandAsString()))
		onlineClient.conn.Write(ansi.Goto(uint16(onlineClient.h)-1, uint16(onlineClient.position)+1))
	}
	s.lines++
	p.conn.Write(ansi.CursorShow)

	p.drawPromptBar()
	event := Event{Player: p, EventType: p.getCommandAsString()}
	s.Events <- event
	// Clear command array to re-use it again.
	p.command = []string{}

	// Clear position
	p.position = 0
}

func (p *Player) deletePartofCommand(position int) {
	p.command = append(p.command[:position], p.command[position+1:]...)
}

func InsertInSlice(original []string, position int, value string) []string {
	//we'll grow by 1
	target := make([]string, len(original)+1)

	//copy everything up to the position
	copy(target, original[:position])

	//set the new value at the desired position
	target[position] = value

	//copy everything left over
	copy(target[position+1:], original[position:])

	return target
}

func (p *Player) clearPromptBar() {
	p.conn.Write(ansi.EraseLine)
	p.conn.Write(ansi.Goto(uint16(p.h)-1, 1))
}
