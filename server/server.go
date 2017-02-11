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
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/droslean/thyraNew/area"
	"github.com/droslean/thyraNew/game"
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
	newPlayers    chan *Client
	onlineClients map[string]*Client
	Players       map[string]area.Player
	Events        chan Event
	Areas         map[string]area.Area
	staticDir     string
}

func NewServer(db *Database, port int) (*Server, error) {
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

	if err := db.GetPrivateKey(s); err != nil {
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
	go s.God()
	// accept all tcp
	for {
		tcpConn, err := server.AcceptTCP()
		if err != nil {
			log.Warn(fmt.Sprintf("accept error (%s)", err))
			continue
		}
		go s.handle(tcpConn)
	}

}

func (s *Server) handle(tcpConn *net.TCPConn) {
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
	c := <-chans
	// channel requests must be serviced - reject rest
	go func() {
		for c := range chans {
			c.Reject(ssh.Prohibited, "only 1 channel allowed")
		}
	}()
	// must be a 'session'
	if t := c.ChannelType(); t != "session" {
		c.Reject(ssh.UnknownChannelType, fmt.Sprintf("unknown channel type: %s", t))
		sshConn.Close()
		return
	}
	conn, chanReqs, err := c.Accept()
	if err != nil {
		log.Warn(fmt.Sprintf("could not accept channel (%s)", err))
		sshConn.Close()
		return
	}
	// non-blocking pull off the id pool
	id := ID(0)
	select {
	case id, _ = <-s.idPool:
	default:
	}
	// show fullgame error
	if id == 0 {
		conn.Write([]byte("This game is full.\r\n"))
		sshConn.Close()
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

	exists, err := s.loadPlayer(name)
	if !exists {
		log.Info(fmt.Sprintf("Player %s doesn't exists", name))
		sshConn.Close()
	}

	player, _ := s.GetPlayerByNick(name)
	client := NewClient(id, sshName, name, hash, conn, &player)
	s.clientLoggedIn(client)

	client.prepareClient(s)

	go func() {
		for r := range chanReqs {
			ok := false
			log.Warn(fmt.Sprintf("[%s] response: %#v", r.Type, r))

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
			log.Info(fmt.Sprintf("replying ok to a %q request", r.Type))
			r.Reply(ok, nil)
		}
	}()
	s.newPlayers <- client
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

func fingerprintKey(k ssh.PublicKey) string {
	bytes := md5.Sum(k.Marshal())
	strbytes := make([]string, len(bytes))
	for i, b := range bytes {
		strbytes[i] = fmt.Sprintf("%02x", b)
	}
	return strings.Join(strbytes, ":")
}

// OnlineClients returns all the online players in the server.
func (s *Server) OnlineClients() []Client {
	s.RLock()
	defer s.RUnlock()

	online := []Client{}
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

func (s *Server) CreateRandomRoom(x, y int) {

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

// GetPlayerByNick returns the player by nickname.
func (s *Server) GetPlayerByNick(nickname string) (area.Player, bool) {
	player, ok := s.Players[nickname]
	return player, ok
}
