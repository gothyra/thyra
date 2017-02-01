package area

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/droslean/thyraNew/game"
	"github.com/jpillora/ansi"
	log "gopkg.in/inconshreveable/log15.v2"
)

type Area struct {
	Name  string          `toml:"name"`
	Intro string          `toml:"intro"`
	Rooms map[string]Room `toml:"rooms"`
}

type Room struct {
	Name        string `toml:"name"`
	Description string `toml:"description"`
	Cubes       []Cube `toml:"cubes"`
}

// Player holds all variables for a character.
type Player struct {
	Nickname string `toml:"nickname"`
	game.PC
	Area         string `toml:"area"`
	Room         string `toml:"room"`
	Position     string `toml:"position"`
	PreviousRoom string `toml:"previousRoom"`
	PreviousArea string `toml:"previousArea"`
}

type Cube struct {
	ID    string `toml:"id"`
	POSX  string `toml:"posx"`
	POSY  string `toml:"posy"`
	Exits []Exit `toml:"exits"`
	Type  string `toml:"type"`
}

type Exit struct {
	ToArea   string `toml:"toarea"`
	ToRoom   string `toml:"toroom"`
	ToCubeID string `toml:"tocubeid"`
}

func FindExits(s [][]Cube, area, room, pos string) [][]string {
	//TODO : Randomize door exit

	ctype := "cube"
	exitarr := [][]string{}
	east := []string{area, "0", room, ctype}
	west := []string{area, "0", room, ctype}
	north := []string{area, "0", room, ctype}
	south := []string{area, "0", room, ctype}

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
						exitarr[0][3] = "door"
					} else {
						exitarr[0][1] = s[x+1][y].ID //EAST

					}
				}

				if west_id > 0 {

					if s[x-1][y].Type == "door" {
						exitarr[1][0] = s[x-1][y].Exits[0].ToArea
						exitarr[1][1] = s[x-1][y].Exits[0].ToCubeID
						exitarr[1][2] = s[x-1][y].Exits[0].ToRoom
						exitarr[1][3] = "door"
					} else {

						exitarr[1][1] = s[x-1][y].ID //WEST
					}
				}

				if north_id > 0 {

					if s[x][y-1].Type == "door" {
						exitarr[2][0] = s[x][y-1].Exits[0].ToArea
						exitarr[2][1] = s[x][y-1].Exits[0].ToCubeID
						exitarr[2][2] = s[x][y-1].Exits[0].ToRoom
						exitarr[2][3] = "door"
					} else {
						exitarr[2][1] = s[x][y-1].ID //NORTH
					}
				}

				if south_id > 0 {

					if s[x][y+1].Type == "door" {
						exitarr[3][0] = s[x][y+1].Exits[0].ToArea
						exitarr[3][1] = s[x][y+1].Exits[0].ToCubeID
						exitarr[3][2] = s[x][y+1].Exits[0].ToRoom
						exitarr[3][3] = "door"
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

func PrintMap(p *Player, online map[string]bool, s [][]Cube) bytes.Buffer {
	var buffer bytes.Buffer

	for y := 0; y < len(s); y++ {
		for x := 0; x < len(s); x++ {
			current, ok := online[s[x][y].ID]
			switch {
			case s[x][y].Type == "door":
				buffer.WriteString(string(ansi.Attribute(398)) + " ")
			case ok && current:
				buffer.WriteString(string(ansi.Attribute(198)) + " ")
			case ok && !current:
				buffer.WriteString(string(ansi.Attribute(165)) + " ")
			case s[x][y].ID == "":

				if hasEmptyNeighbours(s, x, y) {
					buffer.WriteString("  ")
				} else {
					buffer.WriteString(string(ansi.Attribute(182)) + " ")
				}

			default:
				buffer.WriteString(string(ansi.Attribute(183)) + " ")
			}
		}
		buffer.WriteString("\n")
	}

	return buffer
}

// Print to intro tis area
func PrintIntro(desc string) bytes.Buffer {
	var buffer bytes.Buffer
	buffer.WriteString(desc)
	return buffer
}

func hasEmptyNeighbours(array [][]Cube, x, y int) bool {

	//CHECK FOR CORNERS :
	// DOWN RIGHT CORNER
	if x == len(array)-1 && y == len(array)-1 {
		up := y - 1
		left := x - 1
		if array[x][up].ID == "" && // UP
			array[left][y].ID == "" && // LEFT
			array[left][up].ID == "" { // UP LEFT
			return true
		}
	}

	// DOWN LEFT CORNER
	if x == 0 && y == len(array)-1 {
		log.Debug(fmt.Sprintf("CORNER ! : X:%d  Y:%d ", x, y))
		up := y - 1
		right := x + 1

		if array[x][up].ID == "" && // UP
			array[right][up].ID == "" && // UP RIGHT
			array[right][y].ID == "" { // RIGHT
			return true
		}
	}

	// UP RIGHT CORNER
	if x == len(array)-1 && y == 0 {
		down := y + 1
		left := x - 1
		if array[x][down].ID == "" && // DOWN
			array[left][y].ID == "" && // LEFT
			array[left][down].ID == "" { // DOWN LEFT
			return true
		}
	}

	// UP LEFT CORNER
	if x == 0 && y == 0 {
		down := y + 1
		right := x + 1
		if array[x][down].ID == "" && // DOWN
			array[right][y].ID == "" && // RIGHT
			array[right][down].ID == "" { // DOWN RIGHT
			return true
		}
	}

	//Check Borders
	// LEFT BORDER
	if x == 0 && y > 0 && y < len(array)-1 {

		up := y - 1
		down := y + 1
		right := x + 1
		if array[x][up].ID == "" && // UP
			array[x][down].ID == "" && // DOWN
			array[right][up].ID == "" && // UP RIGHT
			array[right][down].ID == "" && // DOWN RIGHT
			array[right][y].ID == "" { // RIGHT
			return true
		}
	}

	// RIGHT BORDER
	if x == len(array)-1 && y > 0 && y < len(array)-1 {
		up := y - 1
		down := y + 1
		left := x - 1
		if array[x][up].ID == "" && // UP
			array[x][down].ID == "" && // DOWN
			array[left][up].ID == "" && // UP LEFT
			array[left][down].ID == "" && // DOWN LEFT
			array[left][y].ID == "" { // LEFT
			return true
		}
	}

	// UP BORDER
	if y == 0 && x > 0 && x < len(array)-1 {
		right := x + 1
		left := x - 1
		down := y + 1
		if array[x][down].ID == "" && // DOWN
			array[right][y].ID == "" && // RIGHT
			array[left][y].ID == "" { // LEFT
			return true
		}
	}

	// DOWN BORDER
	if y == len(array)-1 && x > 0 && x < len(array)-1 {
		up := y - 1
		right := x + 1
		left := x - 1
		if array[x][up].ID == "" && // UP
			array[right][y].ID == "" && // RIGHT
			array[left][y].ID == "" { // LEFT
			return true
		}
	}

	// INSIDE FRAME
	if x > 0 && x < len(array)-1 && y > 0 && y < len(array)-1 {
		up := y - 1
		down := y + 1
		right := x + 1
		left := x - 1

		if array[x][up].ID == "" && // UP
			array[x][down].ID == "" && // DOWN
			array[right][y].ID == "" && // RIGHT
			array[left][y].ID == "" && // LEFT
			array[right][up].ID == "" && // UP RIGHT
			array[right][down].ID == "" && // DOWN RIGHT
			array[left][up].ID == "" && // UP LEFT
			array[left][down].ID == "" { // DOWN LEFT
			return true
		}
	}
	return false
}
