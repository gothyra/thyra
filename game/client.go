package game

import (
	"bufio"
	"io"
	"net"
	"strings"
)

type Client struct {
	Conn     net.Conn
	Nickname string
	Player   *Player
	Cmd      chan string
}

func NewClient(c net.Conn, player *Player, cmd chan string) Client {
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

func (c Client) ReadLinesInto(ch chan<- string, server *Server) {
	bufc := bufio.NewReader(c.Conn)

	for {
		line, err := bufc.ReadString('\n')
		if err != nil {
			break
		}

		userLine := strings.TrimSpace(line)

		if userLine == "" {
			continue
		}

		select {
		case c.Cmd <- userLine:
		}

	}
}
