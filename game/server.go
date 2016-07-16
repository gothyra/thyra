package game

import (
	"bytes"
	"encoding/xml"
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
)

type ServerConfig struct {
	Name string `xml:"name"`
	Motd string `xml:"motd"`
}

type Server struct {
	players       map[string]Player
	Areas         map[string]Area
	staticDir     string
	DefaultArea   Area
	Config        ServerConfig
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

	configFileName := filepath.Join(s.staticDir, "/server.xml")
	fileContent, fileIoErr := ioutil.ReadFile(configFileName)
	if fileIoErr != nil {
		log.Printf("%s could not be loaded: %v\n", configFileName, fileIoErr)
		return fileIoErr
	}

	config := ServerConfig{}
	if xmlerr := xml.Unmarshal(fileContent, &config); xmlerr != nil {
		log.Printf("%s could not be unmarshaled: %v\n", configFileName, xmlerr)
		return xmlerr
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
		if err := xml.Unmarshal(fileContent, &area); err != nil {
			log.Printf("%s could not be unmarshaled: %v\n", path, err)
			return err
		}

		log.Printf("Loaded area %q\n", info.Name())
		s.addArea(area)

		return nil
	}

	return filepath.Walk(s.staticDir+"/areas/", areaWalker)
}

func (s *Server) getPlayerFileName(playerName string) (bool, string) {
	if !s.IsValidUsername(playerName) {
		return false, ""
	}
	return true, s.staticDir + "/player/" + playerName + ".player"
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
	log.Printf("Loading player %q\n", playerFileName)

	fileContent, fileIoErr := ioutil.ReadFile(playerFileName)
	if fileIoErr != nil {
		log.Printf("%s could not be loaded: %v\n", playerFileName, fileIoErr)
		return true, fileIoErr
	}

	player := Player{}
	if xmlerr := xml.Unmarshal(fileContent, &player); xmlerr != nil {
		log.Printf("%s could not be unmarshaled: %v\n", playerFileName, xmlerr)
		return true, xmlerr
	}

	log.Printf("Loaded player %q\n", player.Gamename)
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
		Position: strconv.Itoa(1),
		// TODO: Make this configurable
		Area: "City",
	}
	s.addPlayer(player)
}

func (s *Server) SavePlayer(player Player) bool {
	data, err := xml.MarshalIndent(player, "", "    ")
	if err == nil {
		ok, playerFileName := s.getPlayerFileName(player.Nickname)
		if !ok {
			return false
		}

		if ioerror := ioutil.WriteFile(playerFileName, data, 0666); ioerror != nil {
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
	client.WriteLineToUser(fmt.Sprintf("Good bye %s", client.Player.Gamename))
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

func (s *Server) addArea(area Area) error {
	if area.Tag == "default" {
		log.Printf("default area loaded: %s\n", area.Key)
		s.DefaultArea = area
	}
	s.Areas[area.Key] = area
	return nil
}

func (s *Server) WriteLinesFrom(conn net.Conn, ch <-chan string) {
	for msg := range ch {
		_, err := io.WriteString(conn, msg)
		if err != nil {
			return
		}
	}
}

func (s *Server) CreateRoom(areaID string, roomID string) [][]int {

	biggestx := 0
	biggesty := 0
	biggest := 0

	areaCubes := []Cube{}
	for i := range s.Areas[areaID].Rooms {

		if s.Areas[areaID].Rooms[i].ID == roomID {
			areaCubes = s.Areas[areaID].Rooms[i].Cubes
			break
		}

	}

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

func (s *Server) HandleCommand(c Client, command string, roomsMap map[string]map[string][][]int) {
	map_array := roomsMap[c.Player.Area][c.Player.RoomId]

	switch command {
	case "l", "look", "map":
		posarray := s.FindExits(map_array, c, c.Player.Position, c.Player.Area)
		printToUser(s, c, map_array, posarray, c.Player.Area)

	case "e", "east":
		newpos, _ := strconv.Atoi(s.FindExits(map_array, c, c.Player.Position, c.Player.Area)[0][1])
		posarray := s.FindExits(map_array, c, c.Player.Position, c.Player.Area)
		if newpos > 0 {

			c.Player.Position = strconv.Itoa(newpos)
			c.Player.Area = posarray[0][0]
			c.Player.RoomId = posarray[0][2]
			map_array := roomsMap[c.Player.Area][c.Player.RoomId]

			posarray := s.FindExits(map_array, c, c.Player.Position, c.Player.Area)
			printToUser(s, c, map_array, posarray, c.Player.Area)
		} else {
			c.WriteToUser("You can't go that way\n")
		}

	case "w", "west":
		newpos, _ := strconv.Atoi(s.FindExits(map_array, c, c.Player.Position, c.Player.Area)[1][1])
		posarray := s.FindExits(map_array, c, c.Player.Position, c.Player.Area)
		if newpos > 0 {

			c.Player.Position = strconv.Itoa(newpos)
			c.Player.Area = posarray[1][0]
			c.Player.RoomId = posarray[1][2]
			map_array := roomsMap[c.Player.Area][c.Player.RoomId]
			posarray := s.FindExits(map_array, c, c.Player.Position, c.Player.Area)
			printToUser(s, c, map_array, posarray, c.Player.Area)

		} else {
			c.WriteToUser("You can't go that way\n")
		}

	case "n", "north":
		newpos, _ := strconv.Atoi(s.FindExits(map_array, c, c.Player.Position, c.Player.Area)[2][1])
		posarray := s.FindExits(map_array, c, c.Player.Position, c.Player.Area)
		if newpos > 0 {

			c.Player.Position = strconv.Itoa(newpos)
			c.Player.Area = posarray[2][0]
			c.Player.RoomId = posarray[2][2]
			map_array := roomsMap[c.Player.Area][c.Player.RoomId]
			posarray := s.FindExits(map_array, c, c.Player.Position, c.Player.Area)
			printToUser(s, c, map_array, posarray, c.Player.Area)

		} else {
			c.WriteToUser("You can't go that way\n")
		}

	case "s", "south":
		newpos, _ := strconv.Atoi(s.FindExits(map_array, c, c.Player.Position, c.Player.Area)[3][1])
		posarray := s.FindExits(map_array, c, c.Player.Position, c.Player.Area)
		if newpos > 0 {

			c.Player.Position = strconv.Itoa(newpos)
			c.Player.Area = posarray[3][0]
			c.Player.RoomId = posarray[3][2]
			map_array := roomsMap[c.Player.Area][c.Player.RoomId]
			posarray := s.FindExits(map_array, c, c.Player.Position, c.Player.Area)
			printToUser(s, c, map_array, posarray, c.Player.Area)

		} else {
			c.WriteToUser("You can't go that way\n")
		}

	case "quit":
		s.OnExit(c)
		c.Conn.Close()

	case "online":
		for _, nickname := range s.OnlinePlayers() {
			c.WriteToUser(nickname + "\n")
		}
	case "fight":
		do_fight(c)

	default:
		c.WriteLineToUser("Huh?")
	}
	c.WriteToUser("\n\n" + c.Player.Nickname + " : ")
}

//Briskei ta exits analoga me to position tou user, default possition otan kanei register einai 1
func (server *Server) FindExits(s [][]int, c Client, pos string, area string) [][]string {
	intpos, _ := strconv.Atoi(pos) //to exw balei etsi prwsorina.
	exitarr := [][]string{}        // 2d array s   ,einai o xartis.

	east := []string{area, "0", c.Player.RoomId}
	west := []string{area, "0", c.Player.RoomId}
	north := []string{area, "0", c.Player.RoomId}
	south := []string{area, "0", c.Player.RoomId}

	exitarr = append(exitarr, east)
	exitarr = append(exitarr, west)
	exitarr = append(exitarr, north)
	exitarr = append(exitarr, south)

	roomCubes := server.Areas[area].Rooms[0].Cubes
	for i := range server.Areas[area].Rooms {
		if server.Areas[area].Rooms[i].ID == c.Player.RoomId {
			roomCubes = server.Areas[area].Rooms[i].Cubes
		}
	}

	for x := 0; x < len(s); x++ {
		for y := 0; y < len(s); y++ {
			if s[x][y] == intpos {

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

	//Finding Exits that belongs to different area or room.
	for i := range roomCubes {

		if roomCubes[i].ID == pos && roomCubes[i].Exits != nil {

			for z := range roomCubes[i].Exits {

				if roomCubes[i].Exits[z].ToCubeId != "" && roomCubes[i].Exits[z].FromExit == "EAST" {
					exitarr[0][0] = roomCubes[i].Exits[z].ToArea
					exitarr[0][1] = roomCubes[i].Exits[z].ToCubeId
					exitarr[0][2] = roomCubes[i].Exits[z].ToRoomId

				}

				if roomCubes[i].Exits[z].ToCubeId != "" && roomCubes[i].Exits[z].FromExit == "WEST" {
					exitarr[1][0] = roomCubes[i].Exits[z].ToArea
					exitarr[1][1] = roomCubes[i].Exits[z].ToCubeId
					exitarr[1][2] = roomCubes[i].Exits[z].ToRoomId

				}

				if roomCubes[i].Exits[z].ToCubeId != "" && roomCubes[i].Exits[z].FromExit == "NORTH" {
					exitarr[2][0] = roomCubes[i].Exits[z].ToArea
					exitarr[2][1] = roomCubes[i].Exits[z].ToCubeId
					exitarr[2][2] = roomCubes[i].Exits[z].ToRoomId

				}

				if roomCubes[i].Exits[z].ToCubeId != "" && roomCubes[i].Exits[z].FromExit == "SOUTH" {
					exitarr[3][0] = roomCubes[i].Exits[z].ToArea
					exitarr[3][1] = roomCubes[i].Exits[z].ToCubeId
					exitarr[3][2] = roomCubes[i].Exits[z].ToRoomId

				}
			}
		}

	}
	c.WriteToUser("\n")

	// return 2d string , morfi [0][0] Area , [0][1] Cubeid [0][2] Roomid
	// [0] East , [1] West , [2] North ,[3] South
	//TODO : make cube have multiple exits.now cube can lead only from 1 exit to different area-roomid.
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

	areaIntro := s.Areas[area].Rooms[0].Description
	buffer.WriteString(areaIntro)

	return buffer
}

func printToUser(s *Server, c Client, map_array [][]int, posarray [][]string, areaname string) {

	buffexits := printExits(c, posarray)
	buff := updateMap(s, c, c.Player.Position, map_array, posarray)
	buffintro := printIntro(s, c, areaname)

	c.WriteToUser(buffintro.String() + "\n\n" + buffexits.String() + "\n\n" + buff.String())

}
