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

type Client struct {
	Conn    net.Conn
	Player  *Player
	Cmd     chan<- ClientRequest
	Buff    bytes.Buffer
	Reply   chan Reply
	Bbuffer *Cellbuf
	Fbuffer *Cellbuf
}

type Clients []Client

func (cl Clients) String() string {
	var clients []string
	for _, c := range cl {
		clients = append(clients, c.Player.Nickname)
	}
	return fmt.Sprintf("%#v", clients)
}

func NewClient(c net.Conn, player *Player, cmd chan<- ClientRequest, reply chan Reply) *Client {
	client := &Client{
		Conn:    c,
		Player:  player,
		Cmd:     cmd,
		Reply:   reply,
		Bbuffer: new(Cellbuf),
		Fbuffer: new(Cellbuf),
	}

	return client

}

func (c Client) WriteToUser(msg string) {
	io.WriteString(c.Conn, msg)
}

func (c Client) WriteLineToUser(msg string) {
	io.WriteString(c.Conn, msg+"\n\r")
}

func (c Client) do_tell(client []Client, msg string, name string) {
	for i := range client {
		io.WriteString(client[i].Conn, "\n"+name+": "+msg+"\n\r")
	}
}

func (c Client) ReadLinesInto(stopCh <-chan struct{}) {

	bufc := bufio.NewReader(c.Conn)
	for {
		line, err := bufc.ReadString('\n')
		if err != nil {
			log.Info(fmt.Sprintf("%#v", err))
			return
		}
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		select {
		case c.Cmd <- ClientRequest{Client: &c, Cmd: line}:
		case <-stopCh:
			return
		}

	}
}
