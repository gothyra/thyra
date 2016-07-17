package game

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strings"
)

type Client struct {
	Conn     net.Conn
	Nickname string
	Player   *Player
	Cmd      chan<- ClientRequest
}

func NewClient(c net.Conn, player *Player, cmd chan<- ClientRequest) Client {
	return Client{
		Conn:     c,
		Nickname: player.Nickname,
		Player:   player,
		Cmd:      cmd,
	}
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
	io.WriteString(c.Conn, fmt.Sprintf("Welcome, %s!\n", c.Player.Nickname))

	bufc := bufio.NewReader(c.Conn)
	for {
		line, err := bufc.ReadString('\n')
		if err != nil {
			break
		}
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		select {
		case c.Cmd <- ClientRequest{Client: c, Cmd: line}:
		case <-stopCh:
			return
		}

	}
}
