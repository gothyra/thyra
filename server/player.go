package server

import (
	"fmt"
	"math"

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
	w, h                 int      // terminal size
	screenRunes          [][]rune // the player's view of the screen
	screenColors         [][]ID   // the player's view of the screen
	ready                bool
	resizes              chan resize
	conn                 *ansi.Ansi
	promptBar            *PromptBar
}

// NewPlayer returns an initialized Player.
func NewPlayer(id ID, sshName, name, hash string, conn ssh.Channel) *Player {
	if hash == "" {
		hash = name //finally, hash fallsback to name
	}
	p := &Player{
		id:        id,
		hash:      hash,
		SSHName:   sshName,
		Name:      name,
		ready:     false,
		resizes:   make(chan resize),
		conn:      ansi.Wrap(conn),
		promptBar: NewPromptBar(),
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

func (p *Player) receiveActions(s *Server, player *Player) {

	// Start Prompt Bar
	go p.promptBar.promptBar(s, player)

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

		p.promptBar.promptChan <- b
	}

}
