package game

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
)

type Client struct {
	Conn     net.Conn
	Nickname string
	Player   Player
	Ch       chan string
}

func NewClient(c net.Conn, player Player) Client {
	return Client{
		Conn:     c,
		Nickname: player.Nickname,
		Player:   player,
		Ch:       make(chan string),
	}
}

func (c Client) WriteToUser(msg string) {
	io.WriteString(c.Conn, msg)
}

func (c Client) WriteLineToUser(msg string) {
	io.WriteString(c.Conn, msg+"\n\r")
}

func (c Client) ReadLinesInto(ch chan<- string, server *Server) {
	bufc := bufio.NewReader(c.Conn)

	for {
		line, err := bufc.ReadString('\n')
		if err != nil {
			break
		}

		userLine := strings.TrimSpace(line)

		if userLine == "" {
			continue
		}
		lineParts := strings.SplitN(userLine, " ", 2)

		var command, commandText string
		if len(lineParts) > 0 {
			command = lineParts[0]
		}
		if len(lineParts) > 1 {
			commandText = lineParts[1]
		}

		log.Printf("Command by %s: %s  -  %s", c.Player.Nickname, command, commandText)

		map_array := populate_maparray(server, c.Player.Area)

		printIntro(server, c, c.Player.Area)

		switch command {
		case "l":
			fallthrough
		case "look":
			fallthrough
		case "map":
			posarray := findExits(server, map_array, c, c.Player.Position, c.Player.Area)
			printToUser(server, c, map_array, posarray, c.Player.Area)
		case "go":
		case "exits":
			posarray := findExits(server, map_array, c, c.Player.Position, c.Player.Area)
			printToUser(server, c, map_array, posarray, c.Player.Area)

		case "e":
			fallthrough
		case "east":
			newpos, _ := strconv.Atoi(findExits(server, map_array, c, c.Player.Position, c.Player.Area)[0][1])
			posarray := findExits(server, map_array, c, c.Player.Position, c.Player.Area)
			if newpos > 0 {

				c.Player.Position = strconv.Itoa(newpos)
				c.Player.Area = posarray[0][0]
				map_array := populate_maparray(server, c.Player.Area)
				posarray := findExits(server, map_array, c, c.Player.Position, c.Player.Area)
				printToUser(server, c, map_array, posarray, c.Player.Area)
			} else {
				c.WriteToUser("You can't go that way\n")
			}
		case "w":
			fallthrough
		case "west":
			newpos, _ := strconv.Atoi(findExits(server, map_array, c, c.Player.Position, c.Player.Area)[1][1])
			posarray := findExits(server, map_array, c, c.Player.Position, c.Player.Area)
			if newpos > 0 {

				c.Player.Position = strconv.Itoa(newpos)
				c.Player.Area = posarray[1][0]
				map_array := populate_maparray(server, c.Player.Area)
				posarray := findExits(server, map_array, c, c.Player.Position, c.Player.Area)
				printToUser(server, c, map_array, posarray, c.Player.Area)

			} else {
				c.WriteToUser("You can't go that way\n")
			}
		case "n":
			fallthrough
		case "north":
			newpos, _ := strconv.Atoi(findExits(server, map_array, c, c.Player.Position, c.Player.Area)[2][1])
			posarray := findExits(server, map_array, c, c.Player.Position, c.Player.Area)
			if newpos > 0 {

				c.Player.Position = strconv.Itoa(newpos)
				c.Player.Area = posarray[2][0]
				map_array := populate_maparray(server, c.Player.Area)
				posarray := findExits(server, map_array, c, c.Player.Position, c.Player.Area)
				printToUser(server, c, map_array, posarray, c.Player.Area)

			} else {
				c.WriteToUser("You can't go that way\n")
			}
		case "s":
			fallthrough
		case "south":
			newpos, _ := strconv.Atoi(findExits(server, map_array, c, c.Player.Position, c.Player.Area)[3][1])
			posarray := findExits(server, map_array, c, c.Player.Position, c.Player.Area)
			if newpos > 0 {

				c.Player.Position = strconv.Itoa(newpos)
				c.Player.Area = posarray[3][0]
				map_array := populate_maparray(server, c.Player.Area)
				posarray := findExits(server, map_array, c, c.Player.Position, c.Player.Area)
				printToUser(server, c, map_array, posarray, c.Player.Area)

			} else {
				c.WriteToUser("You can't go that way\n")
			}
		case "say":
			// TODO: implement channel wide communication
			io.WriteString(c.Conn, "\033[1F\033[K") // up one line so we overwrite the say command typed with the result
			ch <- fmt.Sprintf("%s: %s", c.Player.Gamename, commandText)
		case "quit":
			server.OnExit(c)
			c.Conn.Close()
		case "online":
			for _, nickname := range server.OnlinePlayers() {
				c.WriteToUser(nickname + "\n")
			}
		default:
			c.WriteLineToUser("Huh?")
			continue

			c.WriteToUser("\033[1F\033[K")

		}
	}
}

func (c Client) WriteLinesFrom(ch <-chan string) {
	for msg := range ch {
		_, err := io.WriteString(c.Conn, msg)
		if err != nil {
			return
		}
	}
}

func findExits(server *Server, s [][]int, c Client, pos string, area string) [][]string { //Briskei ta exits analoga me to position tou user  ,default possition otan kanei register einai 1
	intpos, _ := strconv.Atoi(pos) //to exw balei etsi prwsorina.
	exitarr := [][]string{}        // 2d array s   ,einai o xartis.

	east := []string{area, "0"}
	west := []string{area, "0"}
	north := []string{area, "0"}
	south := []string{area, "0"}

	exitarr = append(exitarr, east)
	exitarr = append(exitarr, west)
	exitarr = append(exitarr, north)
	exitarr = append(exitarr, south)

	areaCubes := server.levels[area].Rooms[0].Cubes

	for x := 0; x < len(s); x++ {
		for y := 0; y < len(s); y++ { // Kanei return ena array [4]int me ta exits se morfi  {EAST , WEST , NORTH , SOUTH}
			if s[x][y] == intpos { //P.X an kanei return  {50,0,40,0} simainei oti apo to possition pou eisai exei exits EAST kai NORTH
				// EAST se paei sto cube me ID 50 kai NORTH se paei sto cube me ID 40

				if x < len(s)-1 && s[x+1][y] > 0 {
					exitarr[0][1] = strconv.Itoa((s[x+1][y])) //EAST
				}
				if x > 0 && s[x-1][y] > 0 {
					exitarr[1][1] = strconv.Itoa(s[x-1][y]) //WEST
				}
				if y > 0 && s[x][y-1] > 0 {
					exitarr[2][1] = strconv.Itoa(s[x][y-1]) //NORTH
				}
				if y < len(s)-1 && s[x][y+1] > 0 {
					exitarr[3][1] = strconv.Itoa(s[x][y+1]) //SOUTH
				}
			}

		}
	}
	//Finding Exits that belongs to different area.
	for i := range areaCubes {

		if areaCubes[i].ID == pos {
			if areaCubes[i].ToId != "" && areaCubes[i].FromExit == "EAST" {
				exitarr[0][1] = areaCubes[i].ToId
				exitarr[0][0] = areaCubes[i].ToArea

			}

			if areaCubes[i].ToId != "" && areaCubes[i].FromExit == "WEST" {
				exitarr[1][1] = areaCubes[i].ToId
				exitarr[1][0] = areaCubes[i].ToArea

			}

			if areaCubes[i].ToId != "" && areaCubes[i].FromExit == "NORTH" {
				exitarr[2][1] = areaCubes[i].ToId
				exitarr[2][0] = areaCubes[i].ToArea

			}

			if areaCubes[i].ToId != "" && areaCubes[i].FromExit == "SOUTH" {
				exitarr[3][1] = areaCubes[i].ToId
				exitarr[3][0] = areaCubes[i].ToArea

			}
		}

	}
	c.WriteToUser("\n")
	return exitarr
}

func printExits(c Client, exit_array [][]string) bytes.Buffer { //Print exits,From returned [5]string findExits

	var buffer bytes.Buffer
	buffer.WriteString("Exits  : [ ")

	if exit_array[0][1] != "0" {
		buffer.WriteString("East ")
	}

	if exit_array[1][1] != "0" {
		buffer.WriteString("West ")
	}
	if exit_array[2][1] != "0" {
		buffer.WriteString("North ")
	}
	if exit_array[3][1] != "0" {
		buffer.WriteString("South ")
	}
	buffer.WriteString("]\n")
	return buffer

}

func populate_maparray(s *Server, area string) [][]int {

	biggestx := 0
	biggesty := 0
	biggest := 0
	areaCubes := s.levels[area].Rooms[0].Cubes

	for nick := range areaCubes {
		posx, _ := strconv.Atoi(areaCubes[nick].POSX)
		if posx > biggestx {
			biggestx = posx
		}

		posy, _ := strconv.Atoi(areaCubes[nick].POSY)
		if posy > biggesty {
			biggesty = posy
		}

	}
	if biggestx > biggesty {
		biggest = biggestx
	}
	if biggestx < biggesty {
		biggest = biggesty
	} else {
		biggest = biggestx
	}

	maparray := make([][]int, 0)

	for i := 0; i <= biggest; i++ {

		tmp := make([]int, 0)
		for j := 0; j <= biggest; j++ {
			tmp = append(tmp, 0)
		}
		maparray = append(maparray, tmp)

	}
	for z := range areaCubes {
		posx, _ := strconv.Atoi(areaCubes[z].POSX)
		posy, _ := strconv.Atoi(areaCubes[z].POSY)
		if areaCubes[z].ID != "" {
			id, _ := strconv.Atoi(areaCubes[z].ID)
			maparray[posx][posy] = id
		}

	}

	return maparray
}

func updateMap(server *Server, c Client, pos string, s [][]int, posarray [][]string) bytes.Buffer { //Print ton xarti

	var buffer bytes.Buffer
	intpos, _ := strconv.Atoi(pos)

	//TODO : Print "}" when the exit goes to different room-area.
	for x := 0; x < len(s); x++ {

		buffer.WriteString("|")

		for y := 0; y < len(s); y++ {

			if s[y][x] != 0 && s[y][x] != intpos {
				buffer.WriteString("___|")
			} else if s[y][x] == intpos {

				buffer.WriteString("_*_|")
			} else {
				buffer.WriteString("XXX|")

			}
		}

		buffer.WriteString("\n")
	}

	return buffer
}

func printIntro(s *Server, c Client, area string) bytes.Buffer { // Print to intro tis area

	var buffer bytes.Buffer

	areaIntro := s.levels[area].Rooms[0].Description
	buffer.WriteString(areaIntro)

	return buffer
}

func printToUser(s *Server, c Client, map_array [][]int, posarray [][]string, areaname string) {

	buffexits := printExits(c, posarray)
	buff := updateMap(s, c, c.Player.Position, map_array, posarray)
	buffintro := printIntro(s, c, areaname)

	c.WriteToUser(buffintro.String() + "\n\n" + buffexits.String() + "\n\n" + buff.String())

}
