package game

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/gothyra/toml"
)

type Config struct {
	host string
	port int
}

type Server struct {
	sync.RWMutex
	players       map[string]Player
	onlineClients map[string]*Client
	Areas         map[string]Area

	staticDir string
	Config    Config
}

func NewServer(staticDir string) *Server {
	return &Server{
		players:       make(map[string]Player),
		onlineClients: make(map[string]*Client),
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
		// TODO: Lock
		s.Areas[area.Name] = area

		return nil
	}

	return filepath.Walk(s.staticDir+"/areas/", areaWalker)
}

func (s *Server) getPlayerFileName(playerName string) (bool, string) {
	if !IsValidUsername(playerName) {
		return false, ""
	}
	return true, s.staticDir + "/player/" + playerName + ".toml"
}

func IsValidUsername(playerName string) bool {
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
	// TODO: Lock
	s.players[player.Nickname] = player

	return true, nil
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
	// TODO: Lock
	s.players[player.Nickname] = player
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
	s.ClientLoggedOut(client.Nickname)
	client.WriteLineToUser(fmt.Sprintf("Good bye %s", client.Player.Nickname))
}

func (s *Server) ClientLoggedIn(name string, client Client) {
	s.Lock()
	s.onlineClients[name] = &client
	s.Unlock()
}

func (s *Server) ClientLoggedOut(name string) {
	s.Lock()
	delete(s.onlineClients, name)
	s.Unlock()
}

func (s *Server) OnlineClients() []Client {
	s.RLock()
	defer s.RUnlock()

	online := []Client{}
	for _, online_clients := range s.onlineClients {
		online = append(online, *online_clients)
	}

	return online
}

func (s *Server) MapList() []string {
	s.RLock()
	defer s.RUnlock()

	maplist := []string{}
	// TODO: Remove Areas from Server
	for area := range s.Areas {
		maplist = append(maplist, area)
	}

	return maplist
}

// TODO: Remove from Server
func (s *Server) CreateRoom_as_cubes(area, room string) [][]Cube {

	biggestx := 0
	biggesty := 0
	biggest := 0

	roomCubes := []Cube{}
	// TODO: Remove Areas from Server
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

	if biggest < 5 {
		biggest = biggest + 5
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

// TODO: Remove from Server
func (s *Server) HandleCommand(c Client, command string, roomsMap map[string]map[string][][]Cube) {

	map_array := roomsMap[c.Player.Area][c.Player.Room]

	lineParts := strings.SplitN(command, " ", 2)

	var args string
	if len(lineParts) > 0 {
		command = lineParts[0]
	}
	if len(lineParts) > 1 {
		args = lineParts[1]
	}

	switch command {
	case "l", "look", "map":
		posarray := FindExits(map_array, c.Player.Area, c.Player.Room, c.Player.Position)
		printToUser(s, c, map_array, posarray, "")

	case "e", "east":
		newpos, _ := strconv.Atoi(FindExits(map_array, c.Player.Area, c.Player.Room, s.players[c.Player.Nickname].Position)[0][1])
		posarray := FindExits(map_array, c.Player.Area, c.Player.Room, s.players[c.Player.Nickname].Position)
		if newpos > 0 {

			c.Player.Position = strconv.Itoa(newpos)

			delete(s.players, c.Player.Nickname)
			c.Player.Area = posarray[0][0]
			c.Player.Room = posarray[0][2]
			s.players[c.Player.Nickname] = *c.Player

			map_array := roomsMap[c.Player.Area][c.Player.Room]

			posarray := FindExits(map_array, c.Player.Area, c.Player.Room, s.players[c.Nickname].Position)
			printToUser(s, c, map_array, posarray, "")
		} else {

			msg := "You can't go that way"
			printToUser(s, c, map_array, posarray, msg)
		}

	case "w", "west":
		newpos, _ := strconv.Atoi(FindExits(map_array, c.Player.Area, c.Player.Room, c.Player.Position)[1][1])
		posarray := FindExits(map_array, c.Player.Area, c.Player.Room, c.Player.Position)

		if newpos > 0 {
			c.Player.Position = strconv.Itoa(newpos)

			delete(s.players, c.Player.Nickname)
			s.players[c.Player.Nickname] = *c.Player
			c.Player.Area = posarray[1][0]
			c.Player.Room = posarray[1][2]
			map_array := roomsMap[c.Player.Area][c.Player.Room]
			posarray := FindExits(map_array, c.Player.Area, c.Player.Room, c.Player.Position)
			printToUser(s, c, map_array, posarray, "")
		} else {
			msg := "You can't go that way"
			printToUser(s, c, map_array, posarray, msg)
		}

	case "n", "north":
		newpos, _ := strconv.Atoi(FindExits(map_array, c.Player.Area, c.Player.Room, c.Player.Position)[2][1])
		posarray := FindExits(map_array, c.Player.Area, c.Player.Room, c.Player.Position)
		if newpos > 0 {
			c.Player.Position = strconv.Itoa(newpos)

			delete(s.players, c.Player.Nickname)
			s.players[c.Player.Nickname] = *c.Player
			c.Player.Area = posarray[2][0]
			c.Player.Room = posarray[2][2]
			map_array := roomsMap[c.Player.Area][c.Player.Room]
			posarray := FindExits(map_array, c.Player.Area, c.Player.Room, c.Player.Position)
			printToUser(s, c, map_array, posarray, "")
		} else {
			msg := "You can't go that way"
			printToUser(s, c, map_array, posarray, msg)
		}

	case "s", "south":
		newpos, _ := strconv.Atoi(FindExits(map_array, c.Player.Area, c.Player.Room, c.Player.Position)[3][1])
		posarray := FindExits(map_array, c.Player.Area, c.Player.Room, c.Player.Position)
		if newpos > 0 {
			c.Player.Position = strconv.Itoa(newpos)

			delete(s.players, c.Player.Nickname)
			c.Player.Area = posarray[3][0]
			c.Player.Room = posarray[3][2]
			s.players[c.Player.Nickname] = *c.Player

			map_array := roomsMap[c.Player.Area][c.Player.Room]
			posarray := FindExits(map_array, c.Player.Area, c.Player.Room, c.Player.Position)
			printToUser(s, c, map_array, posarray, "")
		} else {
			msg := "You can't go that way"
			printToUser(s, c, map_array, posarray, msg)
			log.Println(msg)
		}
	case "q", "quit":
		s.OnExit(c)
		c.Conn.Close()

	case "fight":
		do_fight(c)
	case "create":
		create_character()

	case "clients":
		for _, players := range s.OnlineClients() {
			fmt.Print("Name : " + players.Player.Nickname + "\n")

		}

	case "tell":
		c.do_tell(s.OnlineClients(), args, c.Player.Nickname)

	/*case "list":
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

	}*/
	case "empty":
		posarray := FindExits(map_array, c.Player.Area, c.Player.Room, c.Player.Position)
		printToUser(s, c, map_array, posarray, "")

	default:
		posarray := FindExits(map_array, c.Player.Area, c.Player.Room, c.Player.Position)
		msg := "Huh?"
		printToUser(s, c, map_array, posarray, msg)
	}
}

func printToUser(s *Server, client Client, map_array [][]Cube, posarray [][]string, event string) {

	buffexits := printExits(client, posarray)

	online := s.OnlineClients()
	e := ""

	for i := range online {

		c := online[i]
		bufmap := updateMap(s, c.Player, map_array)
		buffintro := printIntro(s, c, c.Player.Area, c.Player.Room)

		if client.Player.Nickname == c.Player.Nickname {
			e = event
		}
		c.Reply <- Reply{world: bufmap.Bytes(), events: e, intro: buffintro.Bytes(), exits: buffexits.String()}

	}

}
