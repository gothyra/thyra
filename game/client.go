package game

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"strings"
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

func NewClient(c net.Conn, player *Player, cmd chan<- ClientRequest, reply chan Reply) Client {
	client := Client{
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
	c.Cmd <- ClientRequest{Client: c, Cmd: "empty"}
	bufc := bufio.NewReader(c.Conn)
	for {
		line, err := bufc.ReadString('\n')
		if err != nil {

			break

		}
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			line = "empty"
		}

		select {
		case c.Cmd <- ClientRequest{Client: c, Cmd: line}:
		case <-stopCh:
			return
		}

	}
}
