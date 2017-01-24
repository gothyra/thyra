package server

import (
	"fmt"

	"github.com/jpillora/ansi"
	log "gopkg.in/inconshreveable/log15.v2"
)

type Event struct {
	Player    *Player
	EventType string
}

func God(s *Server) {

	for {
		select {

		case ev := <-s.Events:
			switch ev.EventType {
			case "quit":
				ev.Player.conn.Write(ansi.EraseScreen)
				ev.Player.conn.Close()
			}
			log.Info(fmt.Sprintf("%s : %s", ev.Player.Name, ev.EventType))

			for _, onlineClient := range s.onlinePlayers {
				if s.lines == onlineClient.h-2 {
					onlineClient.conn.Write(ansi.EraseScreen)
					s.lines = 1
				}
				onlineClient.conn.Write([]byte(string(ansi.Goto(uint16(onlineClient.h-onlineClient.h+s.lines), 1)) + ev.Player.Name + " : " + ev.EventType))
				onlineClient.conn.Write(ansi.Goto(uint16(onlineClient.h)-1, uint16(onlineClient.promptBar.position)+1))
			}
			s.lines++

		}

	}
}
