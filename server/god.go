package server

import (
	"bytes"
	"fmt"
	"strconv"
	"time"

	"github.com/droslean/thyraNew/area"

	"github.com/jpillora/ansi"
	log "gopkg.in/inconshreveable/log15.v2"
)

type Event struct {
	Client    *Client
	EventType string
}

func (s *Server) God() {
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

		case ev := <-s.Events:
			log.Debug(fmt.Sprintf("Event type : %s", ev.EventType))
			cl := ev.Client
			online := s.OnlineClientsGetByRoom(cl.Player.Area, cl.Player.Room)
			for i := range online {
				log.Debug(fmt.Sprintf("Clients in room %s", online[i].Player.Nickname))
			}

			switch ev.EventType {
			case "e", "east":
				msg = doMove(*cl, online, roomsMap, 0)

			case "w", "west":
				msg = doMove(*cl, online, roomsMap, 1)

			case "n", "north":
				msg = doMove(*cl, online, roomsMap, 2)

			case "s", "south":
				msg = doMove(*cl, online, roomsMap, 3)

			case "quit":
				ev.Client.conn.Write(ansi.EraseScreen)
				ev.Client.conn.Close()
				s.clientLoggedOut(ev.Client.Name)
			}

			if msg == "door" {
				log.Info("Enter door")
				onlineCurrentRoom := s.OnlineClientsGetByRoom(cl.Player.Area, cl.Player.Room)
				s.godPrintRoom(onlineCurrentRoom, roomsMap, "", fmt.Sprintf("%s enter the room.\n", cl.Player.Nickname))

				onlinePreviousRoom := s.OnlineClientsGetByRoom(cl.Player.PreviousArea, cl.Player.PreviousRoom)
				if onlinePreviousRoom != nil {
					s.godPrintRoom(onlinePreviousRoom, roomsMap, "", fmt.Sprintf("%s left the room.\n", cl.Player.Nickname))
				}
			} else {
				s.godPrintRoom(online, roomsMap, msg, "")
			}
			log.Debug(fmt.Sprintf("%s : %s", ev.Client.Name, ev.EventType))
		}
	}
}

func (s *Server) godPrintRoom(
	clients []Client,
	roomsMap map[string]map[string][][]area.Cube,
	msg string,
	globalMsg string,
) {

	now := time.Now()
	log.Debug(fmt.Sprintf("Start of print: %v", now))

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

		// Re-create the Screen. Instead of clear
		c.screen = NewScreen(c.w, c.h)

		// Create map
		bufmap := area.PlayerCentricMap(p, posToCurr, mapArray)
		c.screen.updateScreen("map", bufmap)

		// Create Available movement
		bufexits := area.PrintExits(area.FindExits(mapArray, c.Player.Area, c.Player.Room, c.Player.Position))
		c.screen.updateScreen("exits", bufexits)

		// Create Name and Description of Room
		buffintro := area.PrintIntro(s.Areas[c.Player.Area].Rooms[c.Player.Room])
		c.screen.updateScreen("intro", buffintro)

		// TODO : Now messages are global. Seperate private messages.
		// Create Messages
		c.screen.updateScreen("message", *bytes.NewBufferString(msg))

		// Finally Draw Screen
		DrawScreen(c)

		// Return cursor to prompt bar
		c.writeGoto(c.h-1, c.promptBar.position+1)

		// Show cursor again
		c.conn.Write(ansi.CursorShow)
	}

	reallyNow := time.Now()
	log.Debug(fmt.Sprintf("End of print: %v", reallyNow))
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

// Initiate the movement to the desired direction.
func doMove(c Client, online []Client, roomsMap map[string]map[string][][]area.Cube, direction int) string {

	mapArray := roomsMap[c.Player.Area][c.Player.Room]
	posarray := area.FindExits(mapArray, c.Player.Area, c.Player.Room, c.Player.Position)
	newPosType := posarray[direction][3]
	newarea := posarray[direction][0]
	newroom := posarray[direction][2]
	newpos, _ := strconv.Atoi(posarray[direction][1])

	// Check if the destination cube is available.
	isAvailable, info := isCubeAvailable(c, online, newarea, newroom, newpos)

	if isAvailable {
		c.Player.PreviousArea = c.Player.Area
		c.Player.PreviousRoom = c.Player.Room
		c.Player.Position = strconv.Itoa(newpos)
		c.Player.Area = newarea
		c.Player.Room = newroom
		return ""
	}

	if newPosType == "door" {
		return "door"
	}

	return info
}

// TODO : After finilize with all cube types , create a check in this function for all types.
// Check if the given cube is available,
// otherwise includes info about what or who is occupying it.
func isCubeAvailable(client Client, online []Client, area string, room string, cube int) (bool, string) {

	if cube <= 0 {
		return false, "You can't go that way\n"
	}

	for i := range online {
		c := online[i]

		if c.Player.Area == area &&
			c.Player.Room == room &&
			c.Player.Position == strconv.Itoa(cube) &&
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
