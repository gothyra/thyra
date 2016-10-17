package game

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
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
	Conn    net.Conn
	Player  *Player
	Cmd     chan<- ClientRequest
	Buff    bytes.Buffer
	Reply   chan Reply
	Bbuffer *Cellbuf
	Fbuffer *Cellbuf
}

func NewClient(c net.Conn, player *Player, cmd chan<- ClientRequest) *Client {
	return &Client{
		Conn:    c,
		Player:  player,
		Cmd:     cmd,
		Reply:   make(chan Reply, 1),
		Bbuffer: new(Cellbuf),
		Fbuffer: new(Cellbuf),
	}
}

func (c Client) WriteToUser(msg string) {
	io.WriteString(c.Conn, msg)
}

func (c Client) WriteLineToUser(msg string) {
	io.WriteString(c.Conn, msg+"\n")
}

func (c Client) do_tell(client []Client, msg string, name string) {
	for i := range client {
		io.WriteString(client[i].Conn, "\n"+name+": "+msg+"\n")
	}
}

func (c Client) ReadLinesInto(quit <-chan struct{}) {
	bufc := bufio.NewReader(c.Conn)

	for {
		line, err := bufc.ReadString('\n')
		if err != nil {
			log.Info(fmt.Sprintf("%#v", err))
			return
		}
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			select {
			case <-quit:
				log.Info(fmt.Sprintf("Player %q quit", c.Player.Nickname))
				return
			default:
			}
			continue
		}

		select {
		case c.Cmd <- ClientRequest{Client: &c, Cmd: line}:
		case <-quit:
			log.Info(fmt.Sprintf("Player %q quit", c.Player.Nickname))
			return
		}
	}
}
