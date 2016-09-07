package game

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"

	"github.com/gothyra/toml"
)

type Config struct {
	host      string
	port      int
	staticDir string
}

type Server struct {
	players       map[string]Player
	Areas         map[string]Area
	staticDir     string
	DefaultArea   Area
	Config        Config
	onlineLock    sync.RWMutex
	onlinePlayers map[string]struct{}
}

func NewServer(staticDir string) *Server {
	return &Server{
		players:       make(map[string]Player),
		onlinePlayers: make(map[string]struct{}),
		Areas:         make(map[string]Area),
		staticDir:     staticDir,
	}
}

func (s *Server) LoadConfig() error {
	log.Println("Loading config ...")

	configFileName := filepath.Join(s.staticDir, "/server.toml")
	fileContent, fileIoErr := ioutil.ReadFile(configFileName)
	if fileIoErr != nil {
		log.Printf("%s could not be loaded: %v\n", configFileName, fileIoErr)
		return fileIoErr
	}

	config := Config{}
	if _, err := toml.Decode(string(fileContent), &config); err != nil {
		log.Printf("%s could not be unmarshaled: %v\n", configFileName, err)
		return err
	}

	s.Config = config
	log.Println("Config loaded.")
	return nil
}

func (s *Server) LoadAreas() error {
	log.Println("Loading areas ...")
	areaWalker := func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		fileContent, fileIoErr := ioutil.ReadFile(path)
		if fileIoErr != nil {
			log.Printf("%s could not be loaded: %v\n", path, fileIoErr)
			return fileIoErr
		}

		area := Area{}
		if _, err := toml.Decode(string(fileContent), &area); err != nil {
			log.Printf("%s could not be unmarshaled: %v\n", path, err)
			return err
		}

		log.Printf("Loaded area %q\n", area.Name)
		s.addArea(area)

		return nil
	}

	return filepath.Walk(s.staticDir+"/areas/", areaWalker)
}

func (s *Server) addArea(area Area) {
	s.Areas[area.Name] = area
}

func (s *Server) getPlayerFileName(playerName string) (bool, string) {
	if !s.IsValidUsername(playerName) {
		return false, ""
	}
	return true, s.staticDir + "/player/" + playerName + ".toml"
}

func (s *Server) IsValidUsername(playerName string) bool {
	r, err := regexp.Compile(`^[a-zA-Z0-9_-]{1,40}$`)
	if err != nil {
		return false
	}
	if !r.MatchString(playerName) {
		return false
	}
	return true
}

func (s *Server) LoadPlayer(playerName string) (bool, error) {
	ok, playerFileName := s.getPlayerFileName(playerName)
	if !ok {
		return false, nil
	}
	if _, err := os.Stat(playerFileName); err != nil {
		return false, nil
	}

	fileContent, fileIoErr := ioutil.ReadFile(playerFileName)
	if fileIoErr != nil {
		log.Printf("%s could not be loaded: %v\n", playerFileName, fileIoErr)
		return true, fileIoErr
	}

	player := Player{}
	if _, err := toml.Decode(string(fileContent), &player); err != nil {
		log.Printf("%s could not be unmarshaled: %v\n", playerFileName, err)
		return true, err
	}

	log.Printf("Loaded player %q\n", player.Nickname)
	s.addPlayer(player)

	return true, nil
}

func (s *Server) addPlayer(player Player) error {
	s.players[player.Nickname] = player
	return nil
}

func (s *Server) GetPlayerByNick(nickname string) (Player, bool) {
	player, ok := s.players[nickname]
	return player, ok
}

func (s *Server) CreatePlayer(nick string) {
	ok, playerFileName := s.getPlayerFileName(nick)
	if !ok {
		return
	}
	if _, err := os.Stat(playerFileName); err == nil {
		log.Printf("Player %q does already exist.\n", nick)
		if _, err := s.LoadPlayer(nick); err != nil {
			log.Printf("Player %s cannot be loaded: %v", nick, err)
		}
		return
	}
	player := Player{
		Nickname: nick,
		PC:       *NewPC(),
		Area:     "City",
		Room:     "Inn",
		Position: "1",
	}
	s.addPlayer(player)
}

func (s *Server) SavePlayer(player Player) bool {
	data := &bytes.Buffer{}
	encoder := toml.NewEncoder(data)
	err := encoder.Encode(player)
	if err == nil {
		ok, playerFileName := s.getPlayerFileName(player.Nickname)
		if !ok {
			return false
		}

		if ioerror := ioutil.WriteFile(playerFileName, data.Bytes(), 0644); ioerror != nil {
			log.Println(ioerror)
			return true
		}
	} else {
		log.Println(err)
	}
	return false
}

func (s *Server) OnExit(client Client) {
	s.SavePlayer(*client.Player)
	s.PlayerLoggedOut(client.Nickname)
	client.WriteLineToUser(fmt.Sprintf("Good bye %s", client.Player.Nickname))
}

func (s *Server) PlayerLoggedIn(nickname string) {
	s.onlineLock.Lock()
	s.onlinePlayers[nickname] = struct{}{}
	s.onlineLock.Unlock()
}

func (s *Server) PlayerLoggedOut(nickname string) {
	s.onlineLock.Lock()
	delete(s.onlinePlayers, nickname)
	s.onlineLock.Unlock()
}

func (s *Server) OnlinePlayers() []string {
	s.onlineLock.RLock()
	defer s.onlineLock.RUnlock()

	online := []string{}
	for nick := range s.onlinePlayers {
		online = append(online, nick)
	}

	return online
}

func (s *Server) MapList() []string {
	s.onlineLock.RLock()
	defer s.onlineLock.RUnlock()

	maplist := []string{}
	for area := range s.Areas {
		maplist = append(maplist, area)
	}

	return maplist
}

func (s *Server) WriteLinesFrom(conn net.Conn, ch <-chan string) {
	for msg := range ch {
		_, err := io.WriteString(conn, msg)
		if err != nil {
			return
		}
	}
}

func (s *Server) CreateRoom(area, room string) [][]int {

	biggestx := 0
	biggesty := 0
	biggest := 0

	roomCubes := []Cube{}
	for i := range s.Areas[area].Rooms {
		if s.Areas[area].Rooms[i].Name == room {
			roomCubes = s.Areas[area].Rooms[i].Cubes
			break
		}
	}

	for nick := range roomCubes {
		posx, _ := strconv.Atoi(roomCubes[nick].POSX)
		if posx > biggestx {
			biggestx = posx
		}

		posy, _ := strconv.Atoi(roomCubes[nick].POSY)
		if posy > biggesty {
			biggesty = posy
		}

	}

	if biggestx < biggesty {
		biggest = biggesty
	} else {
		biggest = biggestx
	}
	biggest++

	maparray := make([][]int, biggest)
	for i := range maparray {
		maparray[i] = make([]int, biggest)
	}

	for z := range roomCubes {
		posx, _ := strconv.Atoi(roomCubes[z].POSX)
		posy, _ := strconv.Atoi(roomCubes[z].POSY)
		if roomCubes[z].ID != "" {
			id, _ := strconv.Atoi(roomCubes[z].ID)
			maparray[posx][posy] = id
		}
	}

	return maparray
}

func (s *Server) CreateRoom_as_cubes(area, room string) [][]Cube {

	biggestx := 0
	biggesty := 0
	biggest := 0

	roomCubes := []Cube{}
	for i := range s.Areas[area].Rooms {
		if s.Areas[area].Rooms[i].Name == room {
			roomCubes = s.Areas[area].Rooms[i].Cubes
			break
		}
	}

	for nick := range roomCubes {
		posx, _ := strconv.Atoi(roomCubes[nick].POSX)
		if posx > biggestx {
			biggestx = posx
		}

		posy, _ := strconv.Atoi(roomCubes[nick].POSY)
		if posy > biggesty {
			biggesty = posy
		}

	}

	if biggestx < biggesty {
		biggest = biggesty
	} else {
		biggest = biggestx
	}
	biggest++

	maparray := make([][]Cube, biggest)
	for i := range maparray {
		maparray[i] = make([]Cube, biggest)
	}

	for z := range roomCubes {
		posx, _ := strconv.Atoi(roomCubes[z].POSX)
		posy, _ := strconv.Atoi(roomCubes[z].POSY)
		if roomCubes[z].ID != "" {
			maparray[posx][posy] = roomCubes[z]
		}
	}

	return maparray
}

func (s *Server) HandleCommand(c Client, command string, roomsMap map[string]map[string][][]Cube) {
	map_array := roomsMap[c.Player.Area][c.Player.Room]

	switch command {
	case "l", "look", "map":
		posarray := s.FindExits(map_array, c.Player.Area, c.Player.Room, c.Player.Position)
		printToUser(s, c, map_array, posarray, c.Player.Area, c.Player.Room)

	case "e", "east":
		newpos, _ := strconv.Atoi(s.FindExits(map_array, c.Player.Area, c.Player.Room, c.Player.Position)[0][1])
		posarray := s.FindExits(map_array, c.Player.Area, c.Player.Room, c.Player.Position)
		if newpos > 0 {

			c.Player.Position = strconv.Itoa(newpos)
			c.Player.Area = posarray[0][0]
			c.Player.Room = posarray[0][2]
			map_array := roomsMap[c.Player.Area][c.Player.Room]

			posarray := s.FindExits(map_array, c.Player.Area, c.Player.Room, c.Player.Position)
			printToUser(s, c, map_array, posarray, c.Player.Area, c.Player.Room)
		} else {
			c.WriteToUser("You can't go that way\n")
		}

	case "w", "west":
		newpos, _ := strconv.Atoi(s.FindExits(map_array, c.Player.Area, c.Player.Room, c.Player.Position)[1][1])
		posarray := s.FindExits(map_array, c.Player.Area, c.Player.Room, c.Player.Position)
		if newpos > 0 {

			c.Player.Position = strconv.Itoa(newpos)
			c.Player.Area = posarray[1][0]
			c.Player.Room = posarray[1][2]
			map_array := roomsMap[c.Player.Area][c.Player.Room]
			posarray := s.FindExits(map_array, c.Player.Area, c.Player.Room, c.Player.Position)
			printToUser(s, c, map_array, posarray, c.Player.Area, c.Player.Room)
		} else {
			c.WriteToUser("You can't go that way\n")
		}

	case "n", "north":
		newpos, _ := strconv.Atoi(s.FindExits(map_array, c.Player.Area, c.Player.Room, c.Player.Position)[2][1])
		posarray := s.FindExits(map_array, c.Player.Area, c.Player.Room, c.Player.Position)
		if newpos > 0 {

			c.Player.Position = strconv.Itoa(newpos)
			c.Player.Area = posarray[2][0]
			c.Player.Room = posarray[2][2]
			map_array := roomsMap[c.Player.Area][c.Player.Room]
			posarray := s.FindExits(map_array, c.Player.Area, c.Player.Room, c.Player.Position)
			printToUser(s, c, map_array, posarray, c.Player.Area, c.Player.Room)

		} else {
			c.WriteToUser("You can't go that way\n")
		}

	case "s", "south":
		newpos, _ := strconv.Atoi(s.FindExits(map_array, c.Player.Area, c.Player.Room, c.Player.Position)[3][1])
		posarray := s.FindExits(map_array, c.Player.Area, c.Player.Room, c.Player.Position)
		if newpos > 0 {

			c.Player.Position = strconv.Itoa(newpos)
			c.Player.Area = posarray[3][0]
			c.Player.Room = posarray[3][2]
			map_array := roomsMap[c.Player.Area][c.Player.Room]
			posarray := s.FindExits(map_array, c.Player.Area, c.Player.Room, c.Player.Position)
			printToUser(s, c, map_array, posarray, c.Player.Area, c.Player.Room)

		} else {
			c.WriteToUser("You can't go that way\n")
		}

	case "q", "quit":
		s.OnExit(c)
		c.Conn.Close()

	case "online":
		for _, nickname := range s.OnlinePlayers() {
			c.WriteToUser(nickname + "\n")
		}

	case "fight":
		do_fight(c)

	case "where":
		updateMap(s, c, c.Player.Position, map_array)

	case "list":
		for i := range map_array {
			for y := range map_array {

				if map_array[y][i].ID != "" {
					id, _ := strconv.Atoi(map_array[y][i].ID)
					if id < 10 {
						c.WriteToUser("|  " + map_array[y][i].ID + " |")

					} else {
						c.WriteToUser("| " + map_array[y][i].ID + " |")

					}

				} else {
					c.WriteToUser("| XX |")

				}

			}
			c.WriteToUser("\n")

		}

	default:
		c.WriteLineToUser("Huh?")
	}
	c.WriteToUser("\n" + c.Player.Nickname + ": ")
}

// Briskei ta exits analoga me to position tou user, default possition otan kanei register einai 1
func (server *Server) FindExits(s [][]Cube, area, room, pos string) [][]string {

	exitarr := [][]string{}
	east := []string{area, "0", room}
	west := []string{area, "0", room}
	north := []string{area, "0", room}
	south := []string{area, "0", room}

	exitarr = append(exitarr, east)
	exitarr = append(exitarr, west)
	exitarr = append(exitarr, north)
	exitarr = append(exitarr, south)

	//roomCubes := server.Areas[area].Rooms[room].Cubes

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

	//	fmt.Printf("%v", exitarr)

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

func updateMap(server *Server, c Client, pos string, s [][]Cube) bytes.Buffer {

	var buffer bytes.Buffer

	for y := 0; y < len(s); y++ {

		buffer.WriteString("|")

		for x := 0; x < len(s); x++ {

			if s[x][y].ID != "" && s[x][y].ID != pos {

				if s[x][y].Type == "door" {
					buffer.WriteString("{O}|")
				} else {
					buffer.WriteString("___|")
				}

			} else if s[x][y].ID == pos {

				buffer.WriteString("_*_|")
			} else {
				buffer.WriteString("XXX|")

			}

		}
		buffer.WriteString("\n")

	}

	return buffer
}

func printIntro(s *Server, c Client, areaID, room string) bytes.Buffer { // Print to intro tis area

	var buffer bytes.Buffer

	areaIntro := s.Areas[areaID].Rooms[room].Description
	buffer.WriteString(areaIntro)

	return buffer
}

func printToUser(s *Server, c Client, map_array [][]Cube, posarray [][]string, areaID, room string) {

	buffexits := printExits(c, posarray)
	buff := updateMap(s, c, c.Player.Position, map_array)
	buffintro := printIntro(s, c, areaID, room)

	c.WriteToUser(buffintro.String() + "\n\n" + buffexits.String() + "\n\n" + buff.String())

}
