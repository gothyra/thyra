package server

import (
	"fmt"

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
			log.Info(fmt.Sprintf("%s : %s", ev.Player.Name, ev.EventType))
		}
	}
}
