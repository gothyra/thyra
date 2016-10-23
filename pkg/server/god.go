package server

import (
	"fmt"
	"strconv"
	"sync"

	log "gopkg.in/inconshreveable/log15.v2"

	"github.com/gothyra/thyra/pkg/area"
	"github.com/gothyra/thyra/pkg/client"
)

func God(s *Server, wg *sync.WaitGroup, quit <-chan struct{}) {
	log.Info("god started")
	defer wg.Done()

	roomsMap := make(map[string]map[string][][]area.Cube)
	for _, a := range s.Areas {
		roomsMap[a.Name] = make(map[string][][]area.Cube)
		for _, room := range a.Rooms {
			roomsMap[a.Name][room.Name] = s.CreateRoom(a.Name, room.Name)
		}
	}

	for {
		select {
		case <-quit:
			log.Warn("God quit")
			return

		case ev := <-s.Events:
			cl := ev.Client
			c := s.OnlineClientsGetByRoom(cl.Player.Area, cl.Player.Room)

			switch ev.Etype {
			case "look":
				wg.Add(1)
				godPrintRoom(s, *cl, c, wg, quit, roomsMap, "", "")

			case "move_east":
				msg := doMove(s, *cl, roomsMap, 0)
				wg.Add(1)
				godPrintRoom(s, *cl, c, wg, quit, roomsMap, msg, "")

			case "move_west":
				msg := doMove(s, *cl, roomsMap, 1)
				wg.Add(1)
				godPrintRoom(s, *cl, c, wg, quit, roomsMap, msg, "")

			case "move_north":
				msg := doMove(s, *cl, roomsMap, 2)
				wg.Add(1)
				godPrintRoom(s, *cl, c, wg, quit, roomsMap, msg, "")

			case "move_south":
				msg := doMove(s, *cl, roomsMap, 3)
				wg.Add(1)
				godPrintRoom(s, *cl, c, wg, quit, roomsMap, msg, "")
			case "enter_door":
				currentroom := s.OnlineClientsGetByRoom(cl.Player.Area, cl.Player.Room)
				wg.Add(1)
				godPrintRoom(s, *cl, currentroom, wg, quit, roomsMap, "", fmt.Sprintf("%s enter the room.", cl.Player.Nickname))

				previousroom := s.OnlineClientsGetByRoom(cl.Player.PreviousArea, cl.Player.PreviousRoom)
				if previousroom != nil {
					wg.Add(1)
					godPrintRoom(s, *cl, previousroom, wg, quit, roomsMap, "", fmt.Sprintf("%s left the room.", cl.Player.Nickname))
				}

			case "quit":
				//TODO :
				//godPrint(s, c, wg, quit, roomsMap, fmt.Sprintf("%s has quit.", c.Player.Nickname))
				//clients := s.OnlineClientsGetByRoom(c.Player.Area, c.Player.Room)
				//for i := range clients {
				//	log.Info(fmt.Sprintf("Clients same room : %s", clients[i].Player.Nickname))
				//}
				s.OnExit(*cl)
				cl.Conn.Close()

			case "unknown":
				wg.Add(1)
				godPrintRoom(s, *cl, c, wg, quit, roomsMap, "Huh?", "")

			}
		}
	}
}

func godPrintRoom(
	s *Server,
	cl client.Client,
	clients []client.Client,
	wg *sync.WaitGroup,
	quit <-chan struct{},
	roomsMap map[string]map[string][][]area.Cube,
	msg string,
	globalMsg string,
) {
	defer wg.Done()

	positionToCurrent := map[string]bool{}

	mapArray := roomsMap[clients[0].Player.Area][clients[0].Player.Room]
	for i := range clients {
		c := clients[i]
		positionToCurrent[c.Player.Position] = false
	}

	for i := range clients {
		c := clients[i]
		p := c.Player

		posToCurr := copyMapWithNewPos(positionToCurrent, c.Player.Position)

		buffintro := area.PrintIntro(s.Areas[c.Player.Area].Rooms[c.Player.Room].Description)
		bufmap := area.PrintMap(p, posToCurr, mapArray)
		bufexits := area.PrintExits(area.FindExits(mapArray, c.Player.Area, c.Player.Room, c.Player.Position))

		reply := client.Reply{
			World: bufmap.Bytes(),
			Intro: buffintro.Bytes(),
			Exits: bufexits.String(),
		}

		if cl.Player.Nickname == p.Nickname {
			reply.Events = msg
		} else {
			reply.Events = globalMsg
		}

		select {
		case c.Reply <- reply:
		case <-quit:
			return
		}
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

func doMove(s *Server, c client.Client, roomsMap map[string]map[string][][]area.Cube, direction int) string {
	event := client.Event{
		Client: &c,
	}

	mapArray := roomsMap[c.Player.Area][c.Player.Room]
	posarray := area.FindExits(mapArray, c.Player.Area, c.Player.Room, c.Player.Position)

	newPosType := posarray[direction][3]
	if newPosType == "door" {
		event.Etype = "enter_door"
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
func isCubeAvailable(s *Server, client client.Client, area string, room string, cube int) (bool, string) {

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
