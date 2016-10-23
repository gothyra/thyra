package server

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"

	"github.com/gothyra/toml"
	log "gopkg.in/inconshreveable/log15.v2"

	"github.com/gothyra/thyra/pkg/area"
	"github.com/gothyra/thyra/pkg/client"
	"github.com/gothyra/thyra/pkg/game"
)

// Config holds the server configuration.
type Config struct {
	host string
	port int
}

// Server holds all the required fields for running a simple game server.
type Server struct {
	sync.RWMutex
	Players       map[string]area.Player
	onlineClients map[string]*client.Client
	Areas         map[string]area.Area
	Events        chan client.Event

	staticDir string
	Config    Config
}

// NewServer creates a new Server.
func NewServer(staticDir string) *Server {
	return &Server{
		Players:       make(map[string]area.Player),
		onlineClients: make(map[string]*client.Client),
		Areas:         make(map[string]area.Area),
		staticDir:     staticDir,
		Events:        make(chan client.Event, 1000),
	}
}

// LoadConfig loads in memory the server configuration from server.toml found in
// the static directory.
func (s *Server) LoadConfig() error {
	log.Info("Loading config ...")

	configFileName := filepath.Join(s.staticDir, "/server.toml")
	fileContent, fileIoErr := ioutil.ReadFile(configFileName)
	if fileIoErr != nil {

		log.Info(fmt.Sprintf("%s could not be loaded: %v\n", configFileName, fileIoErr))
		return fileIoErr
	}

	config := Config{}
	if _, err := toml.Decode(string(fileContent), &config); err != nil {
		log.Info(fmt.Sprintf("%s could not be unmarshaled: %v\n", configFileName, err))
		return err
	}

	s.Config = config
	log.Info("Config loaded.")
	return nil
}

// LoadAreas loads all the areas from the static directory into memory.
// TODO: Change the way we load rooms into memory. We should load rooms
// where online players are. We should also change our schema to hold
// rooms in separate files.
func (s *Server) LoadAreas() error {
	log.Info("Loading areas ...")
	areaWalker := func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		fileContent, fileIoErr := ioutil.ReadFile(path)
		if fileIoErr != nil {
			log.Info(fmt.Sprintf("%s could not be loaded: %v", path, fileIoErr))
			return fileIoErr
		}

		area := area.Area{}
		if _, err := toml.Decode(string(fileContent), &area); err != nil {
			log.Info(fmt.Sprintf("%s could not be unmarshaled: %v", path, err))
			return err
		}

		log.Info(fmt.Sprintf("Loaded area %q", area.Name))
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

// IsValidUsername checks if the given player name is a valid one.
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

// LoadPlayer loads the player into memory.
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
		log.Info(fmt.Sprintf("%s could not be loaded: %v", playerFileName, fileIoErr))
		return true, fileIoErr
	}

	player := area.Player{}
	if _, err := toml.Decode(string(fileContent), &player); err != nil {
		log.Info(fmt.Sprintf("%s could not be unmarshaled: %v", playerFileName, err))
		return true, err
	}

	log.Info(fmt.Sprintf("Loaded player %q", player.Nickname))
	// TODO: Lock
	s.Players[player.Nickname] = player

	return true, nil
}

// GetPlayerByNick returns the player by nickname.
func (s *Server) GetPlayerByNick(nickname string) (area.Player, bool) {
	player, ok := s.Players[nickname]
	return player, ok
}

// CreatePlayer creates a player with the given nickname.
func (s *Server) CreatePlayer(nick string) {
	ok, playerFileName := s.getPlayerFileName(nick)
	if !ok {
		return
	}
	if _, err := os.Stat(playerFileName); err == nil {
		log.Info(fmt.Sprintf("Player %q does already exist.\n", nick))
		if _, err := s.LoadPlayer(nick); err != nil {
			log.Info(fmt.Sprintf("Player %q cannot be loaded: %v", nick, err))
		}
		return
	}
	player := area.Player{
		Nickname: nick,
		PC:       *game.NewPC(),
		Area:     "City",
		Room:     "Inn",
		Position: "1",
	}
	// TODO: Lock
	s.Players[player.Nickname] = player
}

// SavePlayer saves the player back to the static directory.
// TODO: Add an autosave mechanism instead of saving Players
// once they quit.
func (s *Server) SavePlayer(player area.Player) bool {
	data := &bytes.Buffer{}
	encoder := toml.NewEncoder(data)
	err := encoder.Encode(player)
	if err == nil {
		ok, playerFileName := s.getPlayerFileName(player.Nickname)
		if !ok {
			return false
		}

		if ioerror := ioutil.WriteFile(playerFileName, data.Bytes(), 0644); ioerror != nil {
			log.Info(ioerror.Error())
			return true
		}
	} else {
		log.Info(err.Error())
	}
	return false
}

// OnExit is a handler run by the server every time a player quits.
func (s *Server) OnExit(client client.Client) {
	s.SavePlayer(*client.Player)
	s.ClientLoggedOut(client.Player.Nickname)
}

// ClientLoggedIn stores the logged in player into an internal cache that holds
// all online players.
func (s *Server) ClientLoggedIn(name string, client client.Client) {
	s.Lock()
	s.onlineClients[name] = &client
	s.Unlock()
}

// ClientLoggedOut removes the logged out player from the internal cache that
// holds all online players.
func (s *Server) ClientLoggedOut(name string) {
	s.Lock()
	delete(s.onlineClients, name)
	s.Unlock()
}

// OnlineClients returns all the online players in the server.
func (s *Server) OnlineClients() []client.Client {
	s.RLock()
	defer s.RUnlock()

	online := []client.Client{}
	for _, onlineClient := range s.onlineClients {
		online = append(online, *onlineClient)
	}

	return online
}

// OnlineClientsGetByRoom returns all the online players in the given room.
func (s *Server) OnlineClientsGetByRoom(area, room string) []client.Client {
	clients := s.OnlineClients()
	var clientsSameRoom []client.Client

	for i := range clients {
		c := clients[i]
		if area == c.Player.Area && room == c.Player.Room {
			clientsSameRoom = append(clientsSameRoom, c)
		}
	}

	return clientsSameRoom
}

// CreateRoom creates a 2-d array of cubes that essentially consists of a room.
func (s *Server) CreateRoom(a, room string) [][]area.Cube {

	biggestx := 0
	biggesty := 0
	biggest := 0

	roomCubes := []area.Cube{}
	// TODO: Remove Areas from Server
	for i := range s.Areas[a].Rooms {
		if s.Areas[a].Rooms[i].Name == room {
			roomCubes = s.Areas[a].Rooms[i].Cubes
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

	maparray := make([][]area.Cube, biggest)
	for i := range maparray {
		maparray[i] = make([]area.Cube, biggest)
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

// HandleCommand processes commands received by clients.
func (s *Server) HandleCommand(c client.Client, command string) {
	//TODO split command to get arguments

	event := client.Event{
		Client: &c,
	}

	switch command {

	case "l", "look", "map":
		event.Etype = "look"
	case "e", "east":
		event.Etype = "move_east"
	case "w", "west":
		event.Etype = "move_west"
	case "n", "north":
		event.Etype = "move_north"
	case "s", "south":
		event.Etype = "move_south"
	case "quit", "exit":
		event.Etype = "quit"
	default:
		event.Etype = "unknown"
	}
	s.Events <- event
}
