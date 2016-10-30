package server

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
	"time"

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
func NewServer() *Server {
	// Environment variables
	staticDir := os.Getenv("THYRA_STATIC")
	if len(staticDir) == 0 {
		pwd, _ := os.Getwd()
		staticDir = filepath.Join(pwd, "static")
		log.Warn("Set THYRA_STATIC if you wish to configure the directory for static content")
	}
	log.Info(fmt.Sprintf("Using %s for static content", staticDir))

	s := &Server{
		Players:       make(map[string]area.Player),
		onlineClients: make(map[string]*client.Client),
		Areas:         make(map[string]area.Area),
		staticDir:     staticDir,
		Events:        make(chan client.Event, 1000),
	}

	if err := s.loadConfig(); err != nil {
		os.Exit(1)
	}

	if err := s.loadAreas(); err != nil {
		os.Exit(1)
	}

	return s
}

// loadConfig loads in memory the server configuration from server.toml found in
// the static directory.
func (s *Server) loadConfig() error {
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

// loadAreas loads all the areas from the static directory into memory.
// TODO: Change the way we load rooms into memory. We should load rooms
// where online players are. We should also change our schema to hold
// rooms in separate files.
func (s *Server) loadAreas() error {
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

func (s *Server) Start(port int64) {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Info(err.Error())
		os.Exit(1)
	}
	log.Info(fmt.Sprintf("Listen on: %s", ln.Addr()))

	wg := &sync.WaitGroup{}
	quit := make(chan struct{})
	regRequest := make(chan client.LoginRequest, 1000)
	clientRequest := make(chan client.Request, 1000)

	wg.Add(1)
	go handleRegistrations(s, wg, quit, regRequest)

	wg.Add(1)
	go acceptConnections(ln, s, wg, quit, clientRequest, regRequest)

	wg.Add(1)
	go broadcast(s, wg, quit, clientRequest)

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, os.Kill)
	select {
	case <-signals:
		log.Warn("Server is terminating...")
		close(quit)
	}

	wg.Wait()
	log.Warn("Server shutdown.")
}

// handleRegistrations accepts requests for registration and replies back if the requested
// username exists or not.
func handleRegistrations(s *Server, wg *sync.WaitGroup, quit chan struct{}, regRequest chan client.LoginRequest) {
	log.Info("handleRegistrations started")
	defer wg.Done()

	for {
		exists := false
		var err error

		select {
		case <-quit:
			log.Warn("handleRegistrations quit")
			return
		case request := <-regRequest:
			exists, err = s.loadPlayer(request.Username)
			if err != nil {
				io.WriteString(request.Conn, fmt.Sprintf("%s\n", err.Error()))
				continue
			}

			select {
			case request.Reply <- exists:
			case <-quit:
				log.Warn("handleRegistrations quit")
				return
			}

		}
	}
}

func acceptConnections(
	ln net.Listener,
	s *Server,
	wg *sync.WaitGroup,
	quit <-chan struct{},
	clientCh chan<- client.Request,
	regRequest chan client.LoginRequest,
) {
	log.Info("acceptConnections started")
	defer wg.Done()

	for {
		select {
		case <-quit:
			log.Warn("acceptConnections quit")
			return
		default:
		}

		ln.(*net.TCPListener).SetDeadline(time.Now().Add(10 * time.Second))
		conn, err := ln.Accept()
		if err != nil {
			if opErr, ok := err.(*net.OpError); !ok || !opErr.Timeout() {
				log.Info(err.Error())
			}
			continue
		}

		// TODO: handleConnection is not terminating gracefully right now because it blocks on waiting
		// ReadLinesInto to quit which in turn is blocked on user input.
		go handleConnection(conn, s, wg, quit, clientCh, regRequest)
	}
}

// handleConnection should be invoked as a goroutine.
func handleConnection(
	conn net.Conn,
	s *Server,
	wg *sync.WaitGroup,
	quit <-chan struct{},
	clientCh chan<- client.Request,
	regRequest chan<- client.LoginRequest,
) {
	log.Info("handleConnection started")
	defer wg.Done()

	bufc := bufio.NewReader(conn)
	defer conn.Close()

	log.Info(fmt.Sprintf("New connection open: %s", conn.RemoteAddr()))

	io.WriteString(conn, welcomePage)

	var username string
	questions := 0

out:
	for {
		if questions >= 3 {
			io.WriteString(conn, "See you\n")
			return
		}

		username = promptMessage(conn, bufc, "Whats your Nick?\n")
		isValidName := IsValidUsername(username)
		if !isValidName {
			questions++
			io.WriteString(conn, fmt.Sprintf("Username %s is not valid (0-9a-z_-).\n", username))
			continue
		}

		exists := false
		replyCh := make(chan bool, 1)

		select {
		case regRequest <- client.LoginRequest{Username: username, Conn: conn, Reply: replyCh}:
		case <-quit:
			return
		}

		select {
		case exists = <-replyCh:
		case <-quit:
			return
		}

		if exists {
			break out
		}

		questions++
		io.WriteString(conn, fmt.Sprintf("Username %s does not exists.\n", username))
		answer := promptMessage(conn, bufc, "Do you want to create that user? [y|n] ")

		if answer == "y" || answer == "yes" {
			s.CreatePlayer(username)
			break
		}
	}

	player, _ := s.GetPlayerByNick(username)
	c := client.NewClient(conn, &player, clientCh)
	log.Info(fmt.Sprintf("Player %q got connected", c.Player.Nickname))
	s.clientLoggedIn(c.Player.Nickname, *c)

	wg.Add(1)
	go c.Redraw(wg, quit)

	// TODO: Main client thread is not terminating gracefully right now because it blocks on waiting
	// for the user to hit Enter before proceeding to check for quit.
	c.ReadLinesInto(quit)
	log.Info(fmt.Sprintf("Connection from %v closed.", conn.RemoteAddr()))
}

func promptMessage(c net.Conn, bufc *bufio.Reader, message string) string {
	for {
		io.WriteString(c, message)
		answer, _, _ := bufc.ReadLine()
		if string(answer) != "" {
			return string(answer)
		}
	}
}

// TODO: Maybe parallelize this so that each client request is handled on a separate routine.
func broadcast(s *Server, wg *sync.WaitGroup, quit <-chan struct{}, reqChan <-chan client.Request) {
	log.Info("broadcast started")
	defer wg.Done()

	wg.Add(1)
	go God(s, wg, quit)

	for {
		select {
		case request := <-reqChan:
			s.HandleCommand(*request.Client, request.Cmd)
		case <-quit:
			log.Warn("broadcast quit")
			return
		}
	}
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

// loadPlayer loads the player into memory.
func (s *Server) loadPlayer(playerName string) (bool, error) {
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
		if _, err := s.loadPlayer(nick); err != nil {
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

// savePlayer saves the player back to the static directory.
// TODO: Add an autosave mechanism instead of saving Players
// once they quit.
func (s *Server) savePlayer(player area.Player) bool {
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
	s.savePlayer(*client.Player)
	s.clientLoggedOut(client.Player.Nickname)
}

// clientLoggedIn stores the logged in player into an internal cache that holds
// all online players.
func (s *Server) clientLoggedIn(name string, client client.Client) {
	s.Lock()
	s.onlineClients[name] = &client
	s.Unlock()
}

// clientLoggedOut removes the logged out player from the internal cache that
// holds all online players.
func (s *Server) clientLoggedOut(name string) {
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
