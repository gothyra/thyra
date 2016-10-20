package game

import (
	"strconv"
	"sync"

	log "gopkg.in/inconshreveable/log15.v2"
)

func God(
	s *Server,
	wg *sync.WaitGroup,
	quit <-chan struct{},
	map_array map[string]map[string][][]Cube,
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
				msg := do_move(s, *c, map_array, 0)
				wg.Add(1)
				godPrint(s, c, wg, quit, map_array, msg)

			case "move_west":
				msg := do_move(s, *c, map_array, 1)
				wg.Add(1)
				godPrint(s, c, wg, quit, map_array, msg)

			case "move_north":
				msg := do_move(s, *c, map_array, 2)
				wg.Add(1)
				godPrint(s, c, wg, quit, map_array, msg)

			case "move_south":
				msg := do_move(s, *c, map_array, 3)
				wg.Add(1)
				godPrint(s, c, wg, quit, map_array, msg)

			case "quit":
				s.OnExit(*c)
				c.Conn.Close()
			case "unknown":
				godPrint(s, c, wg, quit, map_array, "Huh?")

			}
		}
	}
}

func godPrint(s *Server, client *Client, wg *sync.WaitGroup, quit <-chan struct{}, roomsMap map[string]map[string][][]Cube, msg string) {
	defer wg.Done()

	room := client.Player.Room
	preroom := client.Player.PreviousRoom
	map_array := roomsMap[client.Player.Area][client.Player.Room]
	map_array_pre := roomsMap[client.Player.PreviousArea][client.Player.PreviousRoom]

	var onlineSameRoom []Client
	var previousSameRoom []Client
	online := s.OnlineClients()
	for i := range online {
		c := online[i]

		if c.Player.Room == room {
			onlineSameRoom = append(onlineSameRoom, c)
		} else if c.Player.Room == preroom {
			previousSameRoom = append(previousSameRoom, c)
		}

	}
	p := client.Player

	for i := range onlineSameRoom {
		c := onlineSameRoom[i]

		buffintro := PrintIntro(s, p.Area, p.Room)
		bufmap := PrintMap(s, p, map_array)
		bufexits := PrintExits(FindExits(map_array, c.Player.Area, c.Player.Room, c.Player.Position))

		reply := Reply{
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

		buffexits := PrintExits(FindExits(map_array_pre, c.Player.Area, c.Player.Room, c.Player.Position))

		bufmap := PrintMap(s, c.Player, map_array_pre)
		buffintro := PrintIntro(s, c.Player.Area, c.Player.Room)

		reply := Reply{
			World: bufmap.Bytes(),
			Intro: buffintro.Bytes(),
			Exits: buffexits.String(),
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
}

func do_move(s *Server, c Client, roomsMap map[string]map[string][][]Cube, direction int) string {

	map_array := roomsMap[c.Player.Area][c.Player.Room]
	newpos, _ := strconv.Atoi(FindExits(map_array, c.Player.Area, c.Player.Room, c.Player.Position)[direction][1])
	posarray := FindExits(map_array, c.Player.Area, c.Player.Room, c.Player.Position)

	newarea := posarray[direction][0]
	newroom := posarray[direction][2]
	is_avail, pexist := is_cube_available(s, c, newarea, newroom, newpos)

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

func is_cube_available(s *Server, client Client, area string, room string, cube int) (bool, string) {

	if cube > 0 {
		online := s.OnlineClients()
		for i := range online {
			c := online[i]
			if c.Player.Area == area &&
				c.Player.Room == room &&
				c.Player.Position == strconv.Itoa(cube) &&
				client.Player.Nickname != c.Player.Nickname {
				return false, c.Player.Nickname + " is blocking the way"
			} else {
				continue
			}

		}

	} else {
		return false, "You can't go that way"
	}

	return true, ""
}
