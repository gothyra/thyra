package game

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
)

type ServerConfig struct {
	Name      string `xml:"name"`
	Interface string `xml:"interface"`
	Motd      string `xml:"motd"`
}

type Server struct {
	players       map[string]Player
	levels        map[string]Level
	staticDir     string
	DefaultLevel  Level
	Config        ServerConfig
	onlineLock    sync.RWMutex
	onlinePlayers map[string]struct{}
}

func NewServer(staticDir string) *Server {
	return &Server{
		players:       make(map[string]Player),
		onlinePlayers: make(map[string]struct{}),
		levels:        make(map[string]Level),
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

func (s *Server) LoadLevels() error {
	log.Println("Loading levels ...")
	levelWalker := func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		fileContent, fileIoErr := ioutil.ReadFile(path)
		if fileIoErr != nil {
			log.Printf("%s could not be loaded: %v\n", path, fileIoErr)
			return fileIoErr
		}

		level := Level{}
		if xmlerr := xml.Unmarshal(fileContent, &level); xmlerr != nil {
			log.Printf("%s could not be unmarshaled: %v\n", path, xmlerr)
			return xmlerr
		}

		log.Printf("Loaded level %q\n", info.Name())
		s.addLevel(level)

		return nil
	}

	return filepath.Walk(s.staticDir+"/levels/", levelWalker)
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

func (s *Server) GetRoom(key string) (Level, bool) {
	level, ok := s.levels[key]
	return level, ok
}

func (s *Server) GetName() string {
	return s.Config.Name
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
	s.SavePlayer(client.Player)
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
	for level := range s.levels {
		maplist = append(maplist, level)
	}

	return maplist
}

func (s *Server) addLevel(level Level) error {
	if level.Tag == "default" {
		log.Printf("default level loaded: %s\n", level.Key)
		s.DefaultLevel = level
	}
	s.levels[level.Key] = level
	return nil
}
