package server

import (
	"strconv"
	"sync"

	log "gopkg.in/inconshreveable/log15.v2"

	"github.com/gothyra/thyra/pkg/area"
	"github.com/gothyra/thyra/pkg/client"
)

func God(
	s *Server,
	wg *sync.WaitGroup,
	quit <-chan struct{},
	map_array map[string]map[string][][]area.Cube,
) {
	log.Info("god started")
	defer wg.Done()

	for {
		select {
		case <-quit:
			log.Warn("God quit")
			return

		case ev := <-s.Events:
			c := ev.Client

			switch ev.Etype {
			case "look":
				wg.Add(1)
				godPrint(s, c, wg, quit, map_array, "")

			case "move_east":
				msg := doMove(s, *c, map_array, 0)
				wg.Add(1)
				godPrint(s, c, wg, quit, map_array, msg)

			case "move_west":
				msg := doMove(s, *c, map_array, 1)
				wg.Add(1)
				godPrint(s, c, wg, quit, map_array, msg)

			case "move_north":
				msg := doMove(s, *c, map_array, 2)
				wg.Add(1)
				godPrint(s, c, wg, quit, map_array, msg)

			case "move_south":
				msg := doMove(s, *c, map_array, 3)
				wg.Add(1)
				godPrint(s, c, wg, quit, map_array, msg)

			case "quit":
				s.OnExit(*c)
				c.Conn.Close()

			case "unknown":
				wg.Add(1)
				godPrint(s, c, wg, quit, map_array, "Huh?")

			}
		}
	}
}

func godPrint(s *Server, cl *client.Client, wg *sync.WaitGroup, quit <-chan struct{}, roomsMap map[string]map[string][][]area.Cube, msg string) {
	defer wg.Done()

	room := cl.Player.Room
	preroom := cl.Player.PreviousRoom
	map_array := roomsMap[cl.Player.Area][cl.Player.Room]
	map_array_pre := roomsMap[cl.Player.PreviousArea][cl.Player.PreviousRoom]

	var onlineSameRoom []client.Client
	var previousSameRoom []client.Client
	// positionToCurrent maps the positions of online players inside the room of the current
	// player to a boolean which denotes if that position is held by the current player.
	positionToCurrent := map[string]bool{}
	// previousPositionToCurrent creates the respective positionToCurrent map for players in
	// the previous room from the one the current player is.
	previousPositionToCurrent := map[string]bool{}
	online := s.OnlineClients()
	for i := range online {
		c := online[i]

		if c.Player.Room == room {
			positionToCurrent[c.Player.Position] = false
			onlineSameRoom = append(onlineSameRoom, c)
		} else if c.Player.Room == preroom {
			previousPositionToCurrent[c.Player.Position] = false
			previousSameRoom = append(previousSameRoom, c)
		}
	}
	p := cl.Player

	for i := range onlineSameRoom {
		c := onlineSameRoom[i]

		posToCurr := copyMapWithNewPos(positionToCurrent, c.Player.Position)

		buffintro := area.PrintIntro(s.Areas[c.Player.Area].Rooms[c.Player.Room].Description)
		bufmap := area.PrintMap(p, posToCurr, map_array)
		bufexits := area.PrintExits(area.FindExits(map_array, c.Player.Area, c.Player.Room, c.Player.Position))

		reply := client.Reply{
			World: bufmap.Bytes(),
			Intro: buffintro.Bytes(),
			Exits: bufexits.String(),
		}

		if c.Player.Nickname == p.Nickname {
			reply.Events = msg
		}

		select {
		case c.Reply <- reply:
		case <-quit:
			return
		}
	}

	for i := range previousSameRoom {
		c := previousSameRoom[i]

		posToCurr := copyMapWithNewPos(previousPositionToCurrent, c.Player.Position)

		buffintro := area.PrintIntro(s.Areas[c.Player.Area].Rooms[c.Player.Room].Description)
		bufmap := area.PrintMap(c.Player, posToCurr, map_array_pre)
		buffexits := area.PrintExits(area.FindExits(map_array_pre, c.Player.Area, c.Player.Room, c.Player.Position))

		reply := client.Reply{
			World: bufmap.Bytes(),
			Intro: buffintro.Bytes(),
			Exits: buffexits.String(),
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

	map_array := roomsMap[c.Player.Area][c.Player.Room]
	newpos, _ := strconv.Atoi(area.FindExits(map_array, c.Player.Area, c.Player.Room, c.Player.Position)[direction][1])
	posarray := area.FindExits(map_array, c.Player.Area, c.Player.Room, c.Player.Position)

	newarea := posarray[direction][0]
	newroom := posarray[direction][2]
	is_avail, pexist := isCubeAvailable(s, c, newarea, newroom, newpos)

	if is_avail {

		c.Player.PreviousArea = c.Player.Area
		c.Player.PreviousRoom = c.Player.Room
		c.Player.Position = strconv.Itoa(newpos)
		c.Player.Area = newarea
		c.Player.Room = newroom
		return ""
	}

	return pexist

}

func isCubeAvailable(s *Server, client client.Client, area string, room string, cube int) (bool, string) {

	if cube <= 0 {
		return false, "You can't go that way"
	}

	online := s.OnlineClients()
	for i := range online {
		c := online[i]
		if c.Player.Area == area &&
			c.Player.Room == room &&
			c.Player.Position == strconv.Itoa(cube) &&
			client.Player.Nickname != c.Player.Nickname {
			return false, c.Player.Nickname + " is blocking the way"
		}
	}

	return true, ""
}
