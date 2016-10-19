package game

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"strings"

	log "gopkg.in/inconshreveable/log15.v2"
)

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
	Player *Player
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
}

func NewClient(c net.Conn, player *Player, req chan<- Request) *Client {
	return &Client{
		Conn:    c,
		Player:  player,
		Request: req,
		Reply:   make(chan Reply, 1),
		Bbuffer: new(Cellbuf),
		Fbuffer: new(Cellbuf),
		intbuf:  make([]byte, 0, 16),
	}
}

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
