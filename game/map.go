package game

import (
	"bytes"
	"strconv"
)

func FindExits(s [][]Cube, area, room, pos string) [][]string {
	exitarr := [][]string{}
	east := []string{area, "0", room}
	west := []string{area, "0", room}
	north := []string{area, "0", room}
	south := []string{area, "0", room}

	exitarr = append(exitarr, east)
	exitarr = append(exitarr, west)
	exitarr = append(exitarr, north)
	exitarr = append(exitarr, south)

	east_id := 0
	west_id := 0
	north_id := 0
	south_id := 0

	for y := 0; y < len(s); y++ {
		for x := 0; x < len(s); x++ {
			if s[x][y].ID == pos {

				if x < len(s)-1 {
					east_id, _ = strconv.Atoi(s[x+1][y].ID)
				}
				if x > 0 {
					west_id, _ = strconv.Atoi(s[x-1][y].ID)
				}
				if y > 0 {

					north_id, _ = strconv.Atoi(s[x][y-1].ID)
				}
				if y < len(s)-1 {
					south_id, _ = strconv.Atoi(s[x][y+1].ID)

				}

				if east_id > 0 {

					if s[x+1][y].Type == "door" {
						exitarr[0][0] = s[x+1][y].Exits[0].ToArea
						exitarr[0][1] = s[x+1][y].Exits[0].ToCubeID
						exitarr[0][2] = s[x+1][y].Exits[0].ToRoom
					} else {
						exitarr[0][1] = s[x+1][y].ID //EAST

					}
				}

				if west_id > 0 {

					if s[x-1][y].Type == "door" {
						exitarr[1][0] = s[x-1][y].Exits[0].ToArea
						exitarr[1][1] = s[x-1][y].Exits[0].ToCubeID
						exitarr[1][2] = s[x-1][y].Exits[0].ToRoom
					} else {

						exitarr[1][1] = s[x-1][y].ID //WEST
					}
				}

				if north_id > 0 {

					if s[x][y-1].Type == "door" {
						exitarr[2][0] = s[x][y-1].Exits[0].ToArea
						exitarr[2][1] = s[x][y-1].Exits[0].ToCubeID
						exitarr[2][2] = s[x][y-1].Exits[0].ToRoom
					} else {
						exitarr[2][1] = s[x][y-1].ID //NORTH
					}
				}

				if south_id > 0 {

					if s[x][y+1].Type == "door" {
						exitarr[3][0] = s[x][y+1].Exits[0].ToArea
						exitarr[3][1] = s[x][y+1].Exits[0].ToCubeID
						exitarr[3][2] = s[x][y+1].Exits[0].ToRoom
					} else {

						exitarr[3][1] = s[x][y+1].ID //SOUTH
					}

				}
			}

		}
	}

	// First field denotes direction:
	// [0] East, [1] West, [2] North, [3] South
	// Second array holds the cube we will end up following the direction
	// [][0] ToArea, [][1] ToCubeID, [][2] ToRoom

	return exitarr
}

func PrintExits(exit_array [][]string) bytes.Buffer { //Print exits,From returned [5]string findExits
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

func PrintMap(server *Server, p *Player, s [][]Cube) bytes.Buffer {
	var buffer bytes.Buffer

	hasPlayer := map[string]string{}
	//	pos := p.Position

	for _, players := range server.OnlineClients() {

		if players.Player.Room == p.Room {
			if players.Player.Nickname == p.Nickname {
				//pos = p.Position
			}
			hasPlayer[players.Player.Position] = players.Player.Nickname
		}

	}

	for y := 0; y < len(s); y++ {

		buffer.WriteString("|")

		for x := 0; x < len(s); x++ {
			_, ok := hasPlayer[s[x][y].ID]
			switch {
			case s[x][y].Type == "door":
				buffer.WriteString("O|")
			case ok:
				buffer.WriteString("*|")
			case s[x][y].ID == "":
				buffer.WriteString("X|")
			default:
				buffer.WriteString("_|")
			}
		}
		buffer.WriteString("\n")
	}

	return buffer
}

// Print to intro tis area
func PrintIntro(s *Server, areaID, room string) bytes.Buffer {
	var buffer bytes.Buffer

	areaIntro := s.Areas[areaID].Rooms[room].Description
	buffer.WriteString(areaIntro)

	return buffer
}
