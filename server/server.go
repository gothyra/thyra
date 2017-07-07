package server

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/gothyra/thyra/area"
	"github.com/gothyra/thyra/game"
	"github.com/gothyra/toml"

	"golang.org/x/crypto/ssh"
	log "gopkg.in/inconshreveable/log15.v2"
)

type ID uint16

var (
	matchip    = regexp.MustCompile(`^\d+\.\d+\.\d+\.\d+`) // TODO: make correct
	filtername = regexp.MustCompile(`\W`)                  // non-words
)

type Server struct {
	sync.RWMutex
	port          int
	addresses     string
	idPool        <-chan ID
	logf          func(format string, args ...interface{})
	privateKey    ssh.Signer
	onlineClients map[string]*Client
	Players       map[string]area.Player
	Events        chan Event
	Areas         map[string]area.Area
	staticDir     string
}

// TODO: Use a .thyra.toml file for client configuration.
func NewServer(port int) (*Server, error) {
	// Environment variables
	staticDir := os.Getenv("THYRA_STATIC")
	if len(staticDir) == 0 {
		pwd, _ := os.Getwd()
		staticDir = filepath.Join(pwd, "static")
		log.Warn("Set THYRA_STATIC if you wish to configure the directory for static content")
	}
	log.Info(fmt.Sprintf("Using %s for static content", staticDir))

	idPool := make(chan ID, 100)
	for id := 1; id <= 100; id++ {
		idPool <- ID(id)
	}

	s := &Server{
		port:          port,
		idPool:        idPool,
		onlineClients: make(map[string]*Client),
		Events:        make(chan Event),
		Areas:         make(map[string]area.Area),
		staticDir:     staticDir,
		Players:       make(map[string]area.Player),
	}

	if err := s.loadAreas(); err != nil {
		os.Exit(1)
	}

	db, err := newDatabase(filepath.Join(os.TempDir(), "thyra.db"), true)
	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}

	if err := db.getPrivateKey(s); err != nil {
		return nil, err
	}
	if addrs, err := net.InterfaceAddrs(); err == nil {
		joins := []string{}
		for _, a := range addrs {
			ipv4 := matchip.FindString(a.String())
			if ipv4 != "" {
				joins = append(joins, fmt.Sprintf(" ssh %s -p %d", ipv4, s.port))
			}
		}
		s.addresses = strings.Join(joins, "\n")
	}
	return s, nil
}

func (s *Server) StartServer() {
	// bind to provided port
	server, err := net.ListenTCP("tcp4", &net.TCPAddr{Port: s.port})
	if err != nil {
		log.Info(fmt.Sprintf("%v", err))
	}
	log.Info(fmt.Sprintf("Listening for incoming connections on localhost:%d", s.port))

	// Channel for gracefully shutting down all the rest of the threads.
	stopCh := make(chan struct{})
	wg := &sync.WaitGroup{}

	// God has all the server-side logic.
	wg.Add(1)
	go s.God(stopCh, wg)

	// accept connections
	// wg.Add(1)
	go func() {
		// defer wg.Done()

		for {
			// TODO: Timeout after some time to unblock the loop occasionally
			// and check for graceful termination.
			tcpConn, err := server.AcceptTCP()
			if err != nil {
				log.Warn(err.Error())
				continue
			}
			wg.Add(1)
			go s.handle(tcpConn, stopCh, wg)
		}
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, os.Kill)
	select {
	case <-signals:
		log.Warn("Server is terminating...")
		close(stopCh)
	}

	wg.Wait()
	log.Warn("Server shutdown.")
}

func (s *Server) handle(tcpConn *net.TCPConn, stopCh <-chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()

	// TODO: Revisit everything ssh-related.
	//extract these from connection
	var sshName, hash string
	// perform handshake
	config := &ssh.ServerConfig{
		PublicKeyCallback: func(conn ssh.ConnMetadata, publicKey ssh.PublicKey) (*ssh.Permissions, error) {
			sshName = conn.User()
			if publicKey != nil {
				m := md5.Sum(publicKey.Marshal())
				hash = hex.EncodeToString(m[:])
			}
			return nil, nil
		},
	}
	config.AddHostKey(s.privateKey)
	sshConn, chans, globalReqs, err := ssh.NewServerConn(tcpConn, config)
	if err != nil {
		log.Warn(fmt.Sprintf("new connection handshake failed (%s)", err))
		return
	}
	defer sshConn.Close()

	// global requests must be serviced - discard
	go ssh.DiscardRequests(globalReqs)
	// protect against XTR (cross terminal renderering) attacks
	name := filtername.ReplaceAllString(sshName, "")
	// trim name
	maxlen := 100
	if len(name) > maxlen {
		name = string([]rune(name)[:maxlen])
	}
	// get the first channel
	var c ssh.NewChannel
	select {
	case c = <-chans:
	case <-stopCh:
		return
	}
	// channel requests must be serviced - reject rest
	go func() {
		for c := range chans {
			c.Reject(ssh.Prohibited, "only 1 channel allowed")
		}
	}()
	// must be a 'session'
	if t := c.ChannelType(); t != "session" {
		c.Reject(ssh.UnknownChannelType, fmt.Sprintf("unknown channel type: %s", t))
		return
	}
	conn, chanReqs, err := c.Accept()
	if err != nil {
		log.Warn(fmt.Sprintf("could not accept channel (%s)", err))
		return
	}
	// non-blocking pull off the id pool
	id := ID(0)
	select {
	case id, _ = <-s.idPool:
	case <-stopCh:
		return
	default:
	}
	// show fullgame error
	if id == 0 {
		conn.Write([]byte("This game is full.\r\n"))
		return
	}
	// default name using id
	if name == "" {
		name = fmt.Sprintf("player-%d", id)
	}
	// if user has no public key for some strange reason, use their ip as their unique id
	if hash == "" {
		if ip, _, err := net.SplitHostPort(tcpConn.RemoteAddr().String()); err == nil {
			hash = ip
		}
	}
	log.Info(fmt.Sprintf("Creating new client %q: id: %d, hash: %s", name, id, hash))

	player, err := s.createOrLoadPlayer(name)
	if err != nil {
		log.Warn(fmt.Sprintf("Cannot load player %q: %v", name, err))
		return
	}

	client := NewClient(id, sshName, name, hash, conn, player)
	s.clientLoggedIn(client)

	// Client threads that handle all the output from the server are started here.
	wg.Add(1)
	client.prepareClient(s.Events, stopCh, wg)

	for {
		select {
		case <-stopCh:
			log.Info(fmt.Sprintf("[%s] handle exiting.", client.Name))
			return
		case r := <-chanReqs:
			if r == nil {
				continue
			}
			log.Info(fmt.Sprintf("[%s] request type: %s", client.Name, r.Type))

			ok := false
			switch r.Type {
			case "shell":
				// We don't accept any commands (Payload),
				// only the default shell.
				if len(r.Payload) == 0 {
					ok = true
				}
			case "pty-req":
				// Responding 'ok' here will let the client
				// know we have a pty ready for input
				ok = true
				strlen := r.Payload[3]
				client.resizes <- parseDims(r.Payload[strlen+4:])
			case "window-change":
				client.resizes <- parseDims(r.Payload)
				continue // no response
			}
			log.Info(fmt.Sprintf("[%s] replying ok to a %q request", client.Name, r.Type))
			r.Reply(ok, nil)
		}
	}
}

// parseDims extracts two uint32s from the provided buffer.
func parseDims(b []byte) resize {
	if len(b) < 8 {
		return resize{
			width:  0,
			height: 0,
		}
	}
	w := binary.BigEndian.Uint32(b)
	h := binary.BigEndian.Uint32(b[4:])
	return resize{
		width:  w,
		height: h,
	}
}

// OnlineClients returns all the online players in the server.
func (s *Server) OnlineClients() []Client {
	s.RLock()
	defer s.RUnlock()

	var online []Client
	for _, onlineClient := range s.onlineClients {
		online = append(online, *onlineClient)
	}

	return online
}

// clientLoggedIn stores the logged in player into an internal cache that holds
// all online players.
func (s *Server) clientLoggedIn(client *Client) {
	s.Lock()
	s.onlineClients[client.Name] = client
	s.Unlock()
}

// clientLoggedOut removes the logged out player from the internal cache that
// holds all online players.
func (s *Server) clientLoggedOut(name string) {
	s.Lock()
	delete(s.onlineClients, name)
	s.Unlock()
}

// loadAreas loads all the areas from the static directory into memory.
// TODO: Change the way we load rooms into memory. We should load rooms
// wherever online players are. We should also change our schema to hold
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

// OnlineClientsGetByRoom returns all the online players in the given room.
func (s *Server) OnlineClientsGetByRoom(area, room string) []Client {
	clients := s.OnlineClients()
	var clientsSameRoom []Client

	for i := range clients {
		c := clients[i]
		if area == c.Player.Area && room == c.Player.Room {
			clientsSameRoom = append(clientsSameRoom, c)
		}
	}

	return clientsSameRoom
}

func CreateRandomRoom(x, y int) {
	var buffer bytes.Buffer
	var buffer2 bytes.Buffer
	id := 0

	for w := 0; w < x; w++ {
		for h := 0; h < y; h++ {
			id++
			buffer.WriteString(fmt.Sprintf("{ id = \"%d\", posx = \"%d\", posy = \"%d\" },\n", id, w, h))

		}
	}

	buf := bytes.NewBuffer(buffer.Bytes())

	for i := 0; i < buf.Len(); i++ {
		random := rand.Intn(10)
		line, _ := buf.ReadString('\n')
		if random < 8 {
			buffer2.WriteString(line)
		}
	}

	log.Debug(buffer2.String())
}

// CreateRoom creates a 2-d array of cubes that essentially consists of a room.
func (s *Server) CreateRoom(areaName, room string) [][]area.Cube {

	biggestx := 0
	biggesty := 0
	biggest := 0

	roomCubes := []area.Cube{}
	// TODO: Remove Areas from Server
	for i := range s.Areas[areaName].Rooms {
		if s.Areas[areaName].Rooms[i].Name == room {
			roomCubes = s.Areas[areaName].Rooms[i].Cubes
			break
		}
	}

	for z := range roomCubes {
		posx, _ := strconv.Atoi(roomCubes[z].POSX)
		if posx > biggestx {
			biggestx = posx
		}

		posy, _ := strconv.Atoi(roomCubes[z].POSY)
		if posy > biggesty {
			biggesty = posy
		}

	}

	if biggestx < biggesty {
		biggest = biggesty
	} else {
		biggest = biggestx
	}

	// TODO: Figure out why this needs to happen and remove it.
	if biggest < 5 {
		biggest = biggest + 5
	}
	biggest++

	maparray := make([][]area.Cube, biggest+20)
	for i := range maparray {
		maparray[i] = make([]area.Cube, biggest+20)
	}

	for z := range roomCubes {
		posx, _ := strconv.Atoi(roomCubes[z].POSX)
		posy, _ := strconv.Atoi(roomCubes[z].POSY)
		if roomCubes[z].ID != "" {
			maparray[posx+2][posy+2] = roomCubes[z]
		}
	}

	return maparray
}

// getPlayerFileName constructs the path for the player file.
func (s *Server) getPlayerFileName(playerName string) (string, error) {
	if !isValidUsername(playerName) {
		return "", fmt.Errorf("invalid username: %s", playerName)
	}
	return s.staticDir + "/player/" + playerName + ".toml", nil
}

// isValidUsername checks if the given player name is a valid one.
// TODO: Revisit what we want for a valid username.
func isValidUsername(playerName string) bool {
	r, err := regexp.Compile(`^[a-zA-Z0-9_-]{1,40}$`)
	if err != nil {
		return false
	}
	if !r.MatchString(playerName) {
		return false
	}
	return true
}

// createOrLoadPlayer creates or loads a player with the given nickname.
func (s *Server) createOrLoadPlayer(nick string) (*area.Player, error) {
	s.Lock()
	defer s.Unlock()

	playerFileName, err := s.getPlayerFileName(nick)
	if err != nil {
		// Invalid username.
		return nil, err
	}

	// If the player already exists, load it.
	var player area.Player
	if _, err := os.Stat(playerFileName); err != nil {
		log.Info(fmt.Sprintf("Creating new player %q.", nick))
		// TODO: Create a generator for players.
		player = area.Player{
			Nickname: nick,
			PC:       *game.NewPC(),
			Area:     "City",
			Room:     "Inn",
			Position: "1",
		}
	} else {
		log.Info(fmt.Sprintf("Player %q already exists.", nick))
		fileContent, err := ioutil.ReadFile(playerFileName)
		if err != nil {
			return nil, err
		}
		if _, err := toml.Decode(string(fileContent), &player); err != nil {
			return nil, err
		}
	}

	s.Players[player.Nickname] = player
	return &player, nil
}

// savePlayer saves the player back to the static directory.
// TODO: Add an autosave mechanism instead of saving Players
// once they quit.
func (s *Server) savePlayer(player area.Player) error {
	data := &bytes.Buffer{}
	encoder := toml.NewEncoder(data)
	err := encoder.Encode(player)
	if err != nil {
		return err
	}

	playerFileName, err := s.getPlayerFileName(player.Nickname)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(playerFileName, data.Bytes(), 0644)
}
