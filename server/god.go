package server

import (
	"bytes"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/gothyra/thyra/area"

	"github.com/jpillora/ansi"
	log "gopkg.in/inconshreveable/log15.v2"
)

type Event struct {
	Client    *Client
	EventType string
}

func (s *Server) God(stopCh <-chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()

	roomsMap := make(map[string]map[string][][]area.Cube)
	for _, a := range s.Areas {
		roomsMap[a.Name] = make(map[string][][]area.Cube)
		for _, room := range a.Rooms {
			roomsMap[a.Name][room.Name] = s.CreateRoom(a.Name, room.Name)
		}
	}

	msg := ""

	for {
		select {
		case <-stopCh:
			log.Info("God is exiting.")
			return
		case ev := <-s.Events:
			log.Debug(fmt.Sprintf("Player: %s, event type: %s", ev.Client.Name, ev.EventType))
			c := ev.Client
			online := s.OnlineClientsGetByRoom(c.Player.Area, c.Player.Room)
			log.Debug(fmt.Sprintf("Clients in room %s: %s", c.Player.Room, Clients(online)))

			switch ev.EventType {
			case "e", "east":
				msg = doMove(c, online, roomsMap, 0)

			case "w", "west":
				msg = doMove(c, online, roomsMap, 1)

			case "n", "north":
				msg = doMove(c, online, roomsMap, 2)

			case "s", "south":
				msg = doMove(c, online, roomsMap, 3)

			case "quit":
				c.conn.Write(ansi.EraseScreen)
				c.conn.Close()
				s.savePlayer(*c.Player)
				s.clientLoggedOut(c.Player.Nickname)
			}

			log.Info(fmt.Sprintf("msg: %s, player: %#v", msg, c.Player))

			onlineCurrentRoom := s.OnlineClientsGetByRoom(c.Player.Area, c.Player.Room)

			var globalMsg string
			if msg == "door" {
				globalMsg = fmt.Sprintf("%s enters the room.", c.Player.Nickname)

				onlinePreviousRoom := s.OnlineClientsGetByRoom(c.Player.PreviousArea, c.Player.PreviousRoom)
				log.Info(fmt.Sprintf("Online clients in previous room (%s/%s): %s", c.Player.PreviousArea, c.Player.PreviousRoom, Clients(onlinePreviousRoom)))
				if onlinePreviousRoom != nil {
					s.godPrintRoom(onlinePreviousRoom, roomsMap, "", fmt.Sprintf("%s left the room.", c.Player.Nickname))
				}
			}

			if ev.EventType != "quit" {
				// TODO: Sort out msg
				log.Info(fmt.Sprintf("Online clients in room (%s/%s) for player %s: %s", c.Player.Area, c.Player.Room, c.Player.Nickname, Clients(onlineCurrentRoom)))
				s.godPrintRoom(onlineCurrentRoom, roomsMap, "", globalMsg)
			}
		}
	}
}

// godPrintRoom updates the map, intros, and exits for all the provided clients in a room.
// msg is a private message for a player and globalMsg is a global message in the room.
func (s *Server) godPrintRoom(clients []Client, roomsMap map[string]map[string][][]area.Cube, msg, globalMsg string) {
	now := time.Now()
	log.Debug(fmt.Sprintf("godPrintRoom start: %v", now))

	positionToCurrent := map[string]bool{}
	mapArray := roomsMap[clients[0].Player.Area][clients[0].Player.Room]

	for i := range clients {
		c := clients[i]
		positionToCurrent[c.Player.Position] = false
	}

	for i := range clients {
		c := clients[i]
		p := c.Player
		log.Debug(fmt.Sprintf("Player: %s, Area: %s, Room: %s, CubeID: %s", c.Player.Nickname, c.Player.Area, c.Player.Room, c.Player.Position))

		posToCurr := copyMapWithNewPos(positionToCurrent, c.Player.Position)

		// Re-create the Screen. Instead of clear
		c.screen = NewScreen(c.w, c.h)

		// Create map
		bufMap := area.PlayerCentricMap(p, posToCurr, mapArray)
		c.screen.updateScreenRunes("map", bufMap)

		// Create Available movement
		bufExits := area.PrintExits(area.FindExits(mapArray, c.Player.Area, c.Player.Room, c.Player.Position))
		c.screen.updateScreenRunes("exits", bufExits)

		// Create Name and Description of Room
		buffIntro := area.PrintIntro(s.Areas[c.Player.Area].Rooms[c.Player.Room])
		c.screen.updateScreenRunes("intro", buffIntro)

		// TODO : Now messages are global. Separate private messages.
		// Create Messages
		c.screen.updateScreenRunes("message", *bytes.NewBufferString(msg))

		// Finally Draw Screen
		DrawScreen(c)

		// Return cursor to prompt bar
		c.writeGoto(c.h-1, c.promptBar.position+1)

		// Show cursor again
		c.conn.Write(ansi.CursorShow)
	}

	reallyNow := time.Now()
	log.Debug(fmt.Sprintf("godPrintRoom end: %v", reallyNow))
	log.Debug(fmt.Sprintf("Printed after %f ms", reallyNow.Sub(now).Seconds()*1000))
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

// Initiate the movement to the desired direction. Returns
func doMove(c *Client, online []Client, roomsMap map[string]map[string][][]area.Cube, direction int) string {
	mapArray := roomsMap[c.Player.Area][c.Player.Room]
	posArray := area.FindExits(mapArray, c.Player.Area, c.Player.Room, c.Player.Position)
	newPosType := posArray[direction][3]
	newArea := posArray[direction][0]
	newRoom := posArray[direction][2]
	newPos := posArray[direction][1]

	log.Info(fmt.Sprintf("Player: %s, pos: %s->%s, pos type: %s, area: %s->%s, room: %s->%s",
		c.Player.Nickname, c.Player.Position, newPos, newPosType, c.Player.Area, newArea, c.Player.Room, newRoom))

	// Check if the destination cube is available.
	isAvailable, info := isCubeAvailable(*c, online, newArea, newRoom, newPos)

	msg := ""
	if newPosType == "door" {
		msg = "door"
	}
	if isAvailable {
		c.Player.PreviousArea = c.Player.Area
		c.Player.PreviousRoom = c.Player.Room
		c.Player.Position = newPos
		c.Player.Area = newArea
		c.Player.Room = newRoom
		return msg
	}

	return info
}

// TODO: Switch cube to a Cube struct.
// Check if the given cube is available,
// otherwise includes info about what or who is occupying it.
func isCubeAvailable(client Client, online []Client, area string, room string, cube string) (bool, string) {
	if cubeNum, _ := strconv.Atoi(cube); cubeNum <= 0 {
		return false, "You can't go that way\n"
	}

	for i := range online {
		c := online[i]

		if c.Player.Area == area &&
			c.Player.Room == room &&
			c.Player.Position == cube &&
			client.Player.Nickname != c.Player.Nickname {
			return false, c.Player.Nickname + " is blocking the way\n"
		}
	}
	return true, ""
}

// TODO : Divine by percentage all the Canvas to fit dynamicly to ScreenRune
// TODO : Check for Canvas offset.
// Append all Canvas to final ScreenRune and print it to user.
func DrawScreen(c Client) {
	u := make([]byte, 0)

	// Add mapCanvas to screenRunes
	for h := 0; h < len(c.screen.mapCanvas); h++ {
		for w := 0; w < len(c.screen.mapCanvas[h]); w++ {
			c.screen.screenRunes[c.h-30+h][c.w-20+w] = c.screen.mapCanvas[h][w]
		}
	}

	// Clear mapCanvas
	c.screen.mapCanvas = [][]rune{}

	// Add exitCanvas to screenRunes
	for ex := 0; ex < len(c.screen.exitCanvas); ex++ {
		c.screen.screenRunes[c.h-10][c.w-30+ex] = c.screen.exitCanvas[ex]
	}

	// Add Intro to screenRunes
	for h := 0; h < len(c.screen.introCanvas); h++ {
		for w := 0; w < len(c.screen.introCanvas[h]); w++ {
			c.screen.screenRunes[h][w] = c.screen.introCanvas[h][w]
		}
	}

	// Add Messages to screenRunes
	for msgCh := 0; msgCh < len(c.screen.messagesCanvas); msgCh++ {
		c.screen.screenRunes[c.h-8][c.w-50+msgCh] = c.screen.messagesCanvas[msgCh]
	}

	// Hide Cursor and go to 0,0 potition of the screen.
	// With this way user won't keep terminal history while
	// demostrating frame per second illustration.
	c.conn.Write(ansi.CursorHide)
	c.conn.Write(ansi.Goto(0, 0))

	// Write all the screen data.
	for x := 0; x < len(c.screen.screenRunes)-1; x++ {
		u = append(u, []byte(string("\r"))...)
		for y := 0; y < len(c.screen.screenRunes[x]); y++ {
			u = append(u, []byte(string(c.screen.screenRunes[x][y]))...)
		}
		u = append(u, []byte(string("\n"))...)
	}
	c.conn.Write(u)
}
