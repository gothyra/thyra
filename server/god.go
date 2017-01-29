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
				s.clientLoggedOut(ev.Player.Name)
			}
			//	p := ev.Player
			log.Info(fmt.Sprintf("%s : %s", ev.Player.Name, ev.EventType))

			//online := s.OnlineClients()

			for _, onlineClient := range s.onlineClients {
				if s.lines == onlineClient.h-2 {
					onlineClient.conn.Write(ansi.EraseScreen)
					s.lines = 1
				}
				onlineClient.conn.Write([]byte(string(ansi.Goto(uint16(onlineClient.h-onlineClient.h+s.lines), 1)) + ev.Player.Name + " : " + ev.EventType))
				onlineClient.conn.Write(ansi.Goto(uint16(onlineClient.h)-2, uint16(onlineClient.promptBar.position)+1))

			}
			s.lines++

		}
	}
}

func printToClient(p *Player) {

	drawScreen(p)

}

func drawScreen(p *Player) {
	for i := 0; i < p.w; i++ {
		p.writeGoto(1, p.w+i-p.w)
		p.writeString(string(ansi.Set(ansi.Yellow)) + "-" + string(ansi.Set(ansi.White)))
	}
	p.writeGoto(p.h+2-p.h, 1)
	p.writeString(p.Name)
	p.writeGoto(p.h-1, p.promptBar.position+1)

}
