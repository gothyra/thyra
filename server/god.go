package server

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/droslean/thyraNew/area"

	"github.com/jpillora/ansi"
	log "gopkg.in/inconshreveable/log15.v2"
)

type Event struct {
	Player    *Client
	EventType string
}

func God(s *Server) {
	roomsMap := make(map[string]map[string][][]area.Cube)
	for _, a := range s.Areas {
		roomsMap[a.Name] = make(map[string][][]area.Cube)
		for _, room := range a.Rooms {
			roomsMap[a.Name][room.Name] = s.CreateRoom(a.Name, room.Name)
		}
	}

	for {
		select {

		case ev := <-s.Events:
			cl := ev.Player
			//c := s.OnlineClientsGetByRoom(cl.Player.Area, cl.Player.Room)
			c := s.OnlineClients()
			switch ev.EventType {
			case "e", "east":
				doMove(s, *cl, roomsMap, 0)
				godPrintRoom(s, *cl, c, roomsMap, "", "")
			case "w", "west":
				doMove(s, *cl, roomsMap, 1)
				godPrintRoom(s, *cl, c, roomsMap, "", "")
			case "n", "north":
				doMove(s, *cl, roomsMap, 2)
				godPrintRoom(s, *cl, c, roomsMap, "", "")
			case "s", "south":
				msg := doMove(s, *cl, roomsMap, 3)
				godPrintRoom(s, *cl, c, roomsMap, msg, "")

			case "quit":
				ev.Player.conn.Write(ansi.EraseScreen)
				ev.Player.conn.Close()
				s.clientLoggedOut(ev.Player.Name)
			}

			log.Info(fmt.Sprintf("%s : %s", ev.Player.Name, ev.EventType))

			/*for _, onlineClient := range s.onlineClients {
				if s.lines == onlineClient.h-2 {
					onlineClient.conn.Write(ansi.EraseScreen)
					s.lines = 1
				}

				onlineClient.writeGoto(onlineClient.h-onlineClient.h+s.lines, 1)
				onlineClient.writeString(ev.Player.Name + " : " + ev.EventType)
				onlineClient.writeGoto(onlineClient.h-1, onlineClient.promptBar.position+1)
			}
			s.lines++*/

		}
	}
}

func godPrintRoom(
	s *Server,
	cl Client,
	clients []Client,
	roomsMap map[string]map[string][][]area.Cube,
	msg string,
	globalMsg string,
) {

	positionToCurrent := map[string]bool{}

	//log.Debug(fmt.Sprintf("%#v", clients[0]))
	//log.Debug(fmt.Sprintf("Player : %#v", clients[0].Player))

	mapArray := roomsMap[clients[0].Player.Area][clients[0].Player.Room]

	for i := range clients {
		c := clients[i]
		positionToCurrent[c.Player.Position] = false
	}

	for i := range clients {
		c := clients[i]
		p := c.Player

		posToCurr := copyMapWithNewPos(positionToCurrent, c.Player.Position)
		bufmap := area.PrintMap(p, posToCurr, mapArray)

		counter := 20
		buf := bytes.NewBuffer(bufmap.Bytes())
		for {
			line, err := buf.ReadString('\n')
			if err != nil {
				// TODO: Log errors other than io.EOF
				// log.Info("world buffer read error: %v", err)
				break
			}

			c.writeGoto(c.h-counter, c.w-40)
			c.writeString(line)
			counter--
		}

		c.writeGoto(c.h-1, c.promptBar.position+1)

	}

}

func copyMapWithNewPos(m map[string]bool, currentPos string) map[string]bool {
	copied := map[string]bool{}
	for k, v := range m {
		copied[k] = v
		if k == currentPos {
			copied[k] = true
		}
	}
	return copied
}

func doMove(s *Server, c Client, roomsMap map[string]map[string][][]area.Cube, direction int) string {
	event := Event{
		Player: &c,
	}

	mapArray := roomsMap[c.Player.Area][c.Player.Room]
	posarray := area.FindExits(mapArray, c.Player.Area, c.Player.Room, c.Player.Position)

	newPosType := posarray[direction][3]
	if newPosType == "door" {
		event.EventType = "enter_door"
		s.Events <- event
	}

	newarea := posarray[direction][0]
	newroom := posarray[direction][2]
	newpos, _ := strconv.Atoi(posarray[direction][1])
	isAvailable, info := isCubeAvailable(s, c, newarea, newroom, newpos)

	if isAvailable {
		c.Player.PreviousArea = c.Player.Area
		c.Player.PreviousRoom = c.Player.Room
		c.Player.Position = strconv.Itoa(newpos)
		c.Player.Area = newarea
		c.Player.Room = newroom
		return ""
	}

	return info

}

// isCubeAvailable returns if the given cube is available, otherwise includes info about what or who is
// occupying it.
func isCubeAvailable(s *Server, client Client, area string, room string, cube int) (bool, string) {

	if cube <= 0 {
		return false, "You can't go that way"
	}

	online := s.OnlineClients()
	for i := range online {
		c := online[i]

		if c.Player.Area == area && c.Player.Room == room && c.Player.Position == strconv.Itoa(cube) && client.Player.Nickname != c.Player.Nickname {
			return false, c.Player.Nickname + " is blocking the way"
		}
	}

	return true, ""
}
