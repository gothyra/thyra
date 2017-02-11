package server

import (
	"fmt"
	"math"
	"sync"

	"github.com/droslean/thyraNew/area"
	"github.com/jpillora/ansi"
	"golang.org/x/crypto/ssh"
	log "gopkg.in/inconshreveable/log15.v2"
)

type resize struct {
	width, height uint32
}

// A Player represents a live TCP connection from a client
type Client struct {
	id                   ID     // identification
	hash                 string //hash of public key
	SSHName, Name, cname string
	w, h                 int // terminal size
	ready                bool
	resizes              chan resize
	screen               *Screen
	conn                 *ansi.Ansi
	promptBar            *PromptBar
	Player               *area.Player
}

// NewPlayer returns an initialized Player.
func NewClient(id ID, sshName, name, hash string, conn ssh.Channel, player *area.Player) *Client {
	if hash == "" {
		hash = name //finally, hash fallsback to name
	}
	p := &Client{
		id:        id,
		hash:      hash,
		SSHName:   sshName,
		Name:      name,
		ready:     false,
		resizes:   make(chan resize),
		conn:      ansi.Wrap(conn),
		promptBar: NewPromptBar(),
		Player:    player,
	}
	return p
}

var resizeTmpl = string(ansi.Goto(2, 5)) +
	string(ansi.Set(ansi.Blue)) +
	"Please resize your terminal to %dx%d (+%dx+%d)" + string(ansi.Set(ansi.Default))

func (c *Client) receiveActions(stopCh <-chan struct{}, wg *sync.WaitGroup) {
	// defer wg.Done()

	buff := make([]byte, 3)

	for {
		log.Debug(fmt.Sprintf("read buff is : %v", buff))
		n, err := c.conn.Read(buff)

		if err != nil {
			break
		}
		b := buff[:n]
		if b[0] == 3 {
			break
		}

		// Ignore until terminal size is more than requested.
		if !c.ready {
			continue
		}

		// Send byte array to Prompt bar channel
		select {
		case c.promptBar.promptChan <- b:
		case <-stopCh:
			log.Info("receiveActions is exiting.")
			return
		}
	}

}

func (c *Client) writeString(message string) {
	c.conn.Write([]byte(message))
}

func (c *Client) writeGoto(x, y int) {
	c.conn.Write(ansi.Goto(uint16(x), uint16(y)))
}

func (c *Client) prepareClient(events chan Event, stopCh <-chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()

	// wg.Add(1)
	go c.receiveActions(stopCh, wg)

	wg.Add(1)
	go c.promptBar.promptBar(c, events, stopCh, wg)

	wg.Add(1)
	go c.resizeWatch(stopCh, wg)

	log.Info("prepareClient complete.")
}

func (c *Client) resetScreen() {

	for w := 0; w < c.w; w++ {
		for h := 0; h < c.h-3; h++ {
			c.screen.screenRunes[w][h] = ' '
			c.screen.screenColors[w][h] = ID(255)
		}
	}
}

func (c *Client) resizeWatch(stopCh <-chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case <-stopCh:
			log.Info("resizeWatch is exiting.")
			return
		case r := <-c.resizes:
			c.w = int(r.width)
			c.h = int(r.height)
			log.Info(fmt.Sprintf("%s: Width :%d  Height:%d", c.Name, c.w, c.h))

			// fits?
			if c.w >= 30 && c.h >= 30 {
				c.conn.EraseScreen()
				// send updates!
				c.ready = true
				c.screen = NewScreen(c.w, c.h)
			} else {
				// doesnt fit
				c.conn.EraseScreen()
				c.conn.Write([]byte(fmt.Sprintf(resizeTmpl, 10, 10,
					int(math.Max(float64(10-c.w), 0)),
					int(math.Max(float64(10-c.h), 0)))))
				c.ready = false
			}
		}
	}
}
