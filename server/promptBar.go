package server

import (
	"fmt"
	"strings"

	"github.com/jpillora/ansi"
	log "gopkg.in/inconshreveable/log15.v2"
)

type PromptBar struct {
	command        []string
	commandHistory []string
	rollback       int
	wantHistory    bool
	position       int
	promptChan     chan []byte
}

func NewPromptBar() *PromptBar {
	promptBar := &PromptBar{
		command:        make([]string, 0),
		commandHistory: make([]string, 0),
		rollback:       0,
		wantHistory:    false,
		position:       0,
		promptChan:     make(chan []byte, 3),
	}
	return promptBar
}

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
	DELETE_KEY    = 27
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

	//from 33 to 47
	specialChars1 = []string{
		"!",
		"\"",
		"#",
		"$",
		"%",
		"&",
		"'",
		"(",
		")",
		"*",
		"+",
		",",
		"-",
		".",
		"/",
	}

	//from 58 to 64
	specialChars2 = []string{
		":",
		";",
		"<",
		"=",
		">",
		"?",
		"@",
	}

	//from 91 to 95
	specialChars3 = []string{
		"[",
		"\\",
		"]",
		"^",
		"_",
		"`",
	}

	//from 123 to 126
	specialChars4 = []string{
		"{",
		"|",
		"}",
		"~",
	}
)

func (p *PromptBar) promptBar(s *Server, player *Client) {

	for {
		b := <-p.promptChan

		// Parse Arrows
		if len(b) == 3 && b[0] == ansi.Esc && b[1] == 91 {
			cursorBehavor := []byte{0, 0, 0}
			switch c := b[2]; {

			// We use ARROW_UP to go back in command history.
			case c == ARROW_UP:
				p.wantHistory = true
				p.arrowUp(player)

			// We use ARROW_DOWN to go forward in command history.
			case c == ARROW_DOWN:
				p.wantHistory = true
				p.arrowDown(player)

			// We use ARROW_RIGHT to move right through the command for backspace and delete purpose.
			case c == ARROW_RIGHT:
				if len(p.command) > p.position {
					cursorBehavor = []byte{ansi.Esc, 91, 67}
					p.position++
				}
			// We use ARROW_LEFT to move left through the command for backspace and delete purpose.
			case c == ARROW_LEFT:
				if len(p.command) > 0 {
					cursorBehavor = []byte{ansi.Esc, 91, 68}
					p.position--
				}
			}
			player.conn.Write(cursorBehavor)
		} else {
			p.rollback = 0
		}

		switch n := b[0]; {

		// Check Special chars 1st part
		case n >= 33 && n <= 47:
			num := b[0] - 33
			player.writeString(specialChars1[num])
			p.command = append(p.command, specialChars1[num])
			p.position++

			// Check Special chars 2nd part
		case n >= 58 && n <= 64:
			num := b[0] - 58
			player.writeString(specialChars2[num])
			p.command = append(p.command, specialChars2[num])
			p.position++

			// Check Special chars 3rd part
		case n >= 91 && n <= 96:
			num := b[0] - 91
			player.writeString(specialChars3[num])
			p.command = append(p.command, specialChars3[num])
			p.position++

			// Check Special chars 4th part
		case n >= 123 && n <= 126:
			num := b[0] - 123
			player.writeString(specialChars4[num])
			p.command = append(p.command, specialChars4[num])
			p.position++

		// Check uppercase letters
		case n >= UPPER_ALPHA && n <= UPPER_OMEGA:
			num := b[0] - 65
			player.writeString(strings.ToUpper(alphabet[num]))
			p.command = append(p.command, strings.ToUpper(alphabet[num]))
			p.position++

		// Check for lowercase letters
		case n >= LOW_ALPHA && n <= LOW_OMEGA:
			num := b[0] - 97
			player.writeString(alphabet[num])
			p.command = append(p.command, alphabet[num])
			p.position++

		// Check for numbers
		case n >= NUM_0 && n <= NUM_9:
			num := b[0] - 48
			player.writeString(fmt.Sprintf("%d", num))
			p.command = append(p.command, fmt.Sprintf("%d", num))
			p.position++

		// Enter key
		case n == ENTER_KEY:
			if len(p.command) > 0 {
				p.enterKey(s, player)
			}
		// Space key
		case n == SPACE_KEY:
			p.position++
			if p.position < len(p.command) {
				p.command = InsertInSlice(p.command, p.position-1, " ")
				p.clearPromptBar(player)
				player.writeString(p.getCommandAsString())
				player.writeGoto(player.h-1, p.position+1)

			} else {
				player.writeString(" ")
				p.command = append(p.command, " ")
			}

		// Backspace key
		case n == BACKSPACE_KEY:
			if p.position > 0 {
				p.deletePartofCommand(p.position - 1)
				p.position--
				p.clearPromptBar(player)
				player.writeString(p.getCommandAsString())
				player.writeGoto(player.h-1, p.position+1)
			}
			// Delete Key
		case n == DELETE_KEY && b[2] == 51:
			if p.position < len(p.command) {
				p.deletePartofCommand(p.position)
				p.clearPromptBar(player)
				player.writeString(p.getCommandAsString())
				player.writeGoto(player.h-1, p.position+1)
			}

		//  Key ] only for debuging purpose.
		case n == 93:
			log.Info(fmt.Sprintf("%#v", p.commandHistory))
			log.Info(fmt.Sprintf("%#v", p.command))

		}
		//event := Event{Player: player, EventType: p.getCommandAsString()}
		//s.Events <- event
	}
}

func (p *PromptBar) getCommandAsString() string {
	cmd := ""
	for i := range p.command {
		cmd += p.command[i]
	}
	return cmd
}

func (p *PromptBar) convertCommadHistoryToArray(command string) {
	for i := 0; i < len(command); i++ {
		p.command = append(p.command, string(command[i]))
	}
}

func (p *PromptBar) fillPromptBar(player *Client) string {
	promptBar := ""
	for i := 0; i < player.w; i++ {
		promptBar += "~"
	}
	return promptBar
}

func (p *PromptBar) drawPromptBar(player *Client) {
	player.conn.Write([]byte(string(ansi.Goto(uint16(player.h)-2, 1)) + p.fillPromptBar(player)))
	player.conn.Write([]byte(string(ansi.Goto(uint16(player.h), 1)) + p.fillPromptBar(player)))
	player.conn.Write(ansi.Goto(uint16(player.h)-1, 1))
}

// Travel backwards through the history of commands
func (p *PromptBar) arrowUp(player *Client) {
	p.clearPromptBar(player)
	p.rollback++
	log.Debug(fmt.Sprintf("Len %d , Rollback %d", len(p.commandHistory), p.rollback))
	if len(p.commandHistory)-p.rollback >= 0 {
		// Clear command array to re-use it again.
		p.command = []string{}

		player.writeString(p.commandHistory[len(p.commandHistory)-p.rollback])
		p.convertCommadHistoryToArray(p.commandHistory[len(p.commandHistory)-p.rollback])

		p.position = len(p.command)

	} else {
		log.Debug("No command history")
		// Clear command array to re-use it again.
		p.command = []string{}
		p.wantHistory = false
		p.rollback--
	}
}

// Travel forwards through the history of commands
func (p *PromptBar) arrowDown(player *Client) {
	p.clearPromptBar(player)
	p.rollback--

	log.Debug(fmt.Sprintf("Len %d , Rollback %d", len(p.commandHistory), p.rollback))
	if p.rollback > 0 {
		// Clear command array to re-use it again.
		p.command = []string{}
		player.writeString(p.commandHistory[len(p.commandHistory)-p.rollback])
		p.convertCommadHistoryToArray(p.commandHistory[len(p.commandHistory)-p.rollback])
		p.position = len(p.command)

	} else {
		log.Debug("No command history")
		// Clear command array to re-use it again.
		p.command = []string{}
		p.wantHistory = false
		p.rollback++
	}
}

// This function sends an event to s.Events channel.
// GOD thread will handle those events.
func (p *PromptBar) enterKey(s *Server, player *Client) {
	p.clearPromptBar(player)
	player.conn.Write(ansi.CursorHide)

	p.commandHistory = append(p.commandHistory, p.getCommandAsString())

	player.conn.Write(ansi.CursorShow)

	p.drawPromptBar(player)
	event := Event{Player: player, EventType: p.getCommandAsString()}
	s.Events <- event

	// Clear command array to re-use it again.
	p.command = []string{}

	// Clear position
	p.position = 0
}

func (p *PromptBar) deletePartofCommand(position int) {
	p.command = append(p.command[:position], p.command[position+1:]...)
}

func InsertInSlice(original []string, position int, value string) []string {
	//grow by 1
	target := make([]string, len(original)+1)

	//copy everything up to the position
	copy(target, original[:position])

	//set the new value at the desired position
	target[position] = value

	//copy everything left over
	copy(target[position+1:], original[position:])

	return target
}

func (p *PromptBar) clearPromptBar(player *Client) {
	player.conn.Write(ansi.EraseLine)
	player.conn.Write(ansi.Goto(uint16(player.h)-1, 1))
}
