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
		}
	}
}
