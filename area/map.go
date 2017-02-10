package area

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/droslean/thyraNew/game"
	log "gopkg.in/inconshreveable/log15.v2"

	"github.com/jpillora/ansi"
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

// Find Available Movement
func FindExits(s [][]Cube, area, room, pos string) [][]string {
	// TODO : Randomize door exit
	// TODO : ADD  NE , NW , SE , SW

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
		for x := 0; x < len(s[y]); x++ {
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

// Print Available Movement
func PrintExits(exit_array [][]string) bytes.Buffer {
	var buffer bytes.Buffer

	buffer.WriteString("Movement: [ ")

	if exit_array[0][1] != "0" {
		buffer.WriteString("→ ")
	}

	if exit_array[1][1] != "0" {
		buffer.WriteString("← ")
	}

	if exit_array[2][1] != "0" {
		buffer.WriteString("↑ ")
	}

	if exit_array[3][1] != "0" {
		buffer.WriteString("↓ ")
	}

	buffer.WriteString("]\n")
	return buffer
}

func PlayerCentricMap(p *Player, online map[string]bool, s [][]Cube) bytes.Buffer {
	var buffer bytes.Buffer
	var buffer2 bytes.Buffer

	r := 8
	px := 0
	py := 0

	// Find X,Y from Player's position
	for x := 0; x < len(s); x++ {
		for y := 0; y < len(s[x]); y++ {
			if p.Position == s[x][y].ID {
				px, _ = strconv.Atoi(s[x][y].POSX)
				py, _ = strconv.Atoi(s[x][y].POSY)
				break
			}
		}
	}

	log.Debug(fmt.Sprintf("Position X:%d Y:%d", px, py))

	for y1 := 0; y1 < len(s); y1++ {
		for x1 := 0; x1 < len(s[y1]); x1++ {

			if ((x1-px)*(x1-px) + (y1-py)*(y1-py)) <= r*r {
				current, ok := online[s[x1][y1].ID]
				switch {
				case s[x1][y1].Type == "door":
					buffer.WriteString(string(ansi.Attribute(398)))
				case ok && current:
					buffer.WriteString(string(ansi.Attribute(198)))
				case ok && !current:
					buffer.WriteString(string(ansi.Attribute(165)))
				case s[x1][y1].ID == "":
					if hasEmptyNeighbours(s, x1, y1) {
						buffer.WriteString("")
					} else {
						buffer.WriteString(string(ansi.Attribute(182)))
					}
				default:
					buffer.WriteString(string(ansi.Attribute(183)))
				}
			}
		}
		buffer.WriteString("\n")
	}

	// Clear empty lines.
	buf := bytes.NewBuffer(buffer.Bytes())
	for i := 0; i < buffer.Len(); i++ {
		line, _ := buf.ReadString('\n')
		if len(line) > 1 {
			buffer2.WriteString(line)
		}
	}
	return buffer2

}

// Generate Map
func PrintMap(p *Player, online map[string]bool, s [][]Cube) bytes.Buffer {
	var buffer bytes.Buffer

	for y := 0; y < len(s); y++ {
		for x := 0; x < len(s); x++ {
			current, ok := online[s[x][y].ID]
			switch {
			case s[x][y].Type == "door":
				buffer.WriteString(string(ansi.Attribute(398)))
			case ok && current:
				buffer.WriteString(string(ansi.Attribute(198)))
			case ok && !current:
				buffer.WriteString(string(ansi.Attribute(165)))
			case s[x][y].ID == "":
				if hasEmptyNeighbours(s, x, y) {
					buffer.WriteString("")
				} else {
					buffer.WriteString(string(ansi.Attribute(182)))
				}
			default:
				buffer.WriteString(string(ansi.Attribute(183)))
			}
		}
		buffer.WriteString("\n")
	}

	return buffer
}

// Print Name and Description of a Room
func PrintIntro(room Room) bytes.Buffer {
	var buffer bytes.Buffer
	buffer.WriteString("| " + room.Name + " |\n\n")
	buffer.WriteString(room.Description)
	return buffer
}

func hasEmptyNeighbours(array [][]Cube, x, y int) bool {

	// Check for corners

	// Down Right corner
	if x == len(array)-1 && y == len(array)-1 {
		up := y - 1
		left := x - 1
		if array[x][up].ID == "" && // UP
			array[left][y].ID == "" && // LEFT
			array[left][up].ID == "" { // UP LEFT
			return true
		}
	}

	// Down left Corner
	if x == 0 && y == len(array)-1 {
		up := y - 1
		right := x + 1

		if array[x][up].ID == "" && // UP
			array[right][up].ID == "" && // UP RIGHT
			array[right][y].ID == "" { // RIGHT
			return true
		}
	}

	// Up Right Corner
	if x == len(array)-1 && y == 0 {
		down := y + 1
		left := x - 1
		if array[x][down].ID == "" && // DOWN
			array[left][y].ID == "" && // LEFT
			array[left][down].ID == "" { // DOWN LEFT
			return true
		}
	}

	// Up left Corner
	if x == 0 && y == 0 {
		down := y + 1
		right := x + 1
		if array[x][down].ID == "" && // DOWN
			array[right][y].ID == "" && // RIGHT
			array[right][down].ID == "" { // DOWN RIGHT
			return true
		}
	}

	// Check Borders

	// Left Side
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

	// Right Side
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

	// Up Side
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

	// Down Side
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

	// Inner Frame
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
