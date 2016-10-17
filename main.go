package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/gothyra/thyra/game"
	log "gopkg.in/inconshreveable/log15.v2"
	"gopkg.in/inconshreveable/log15.v2/stack"
)

func customFormat() log.Format {
	return log.FormatFunc(func(r *log.Record) []byte {
		b := &bytes.Buffer{}
		call := stack.Call(r.CallPC[0])
		fmt.Fprintf(b, "[%s %s:%d] %s\n", r.Time.Format("2006-01-02|15:04:05.000"), call, call, r.Msg)
		return b.Bytes()
	})
}

func init() {
	h := log.StreamHandler(os.Stdout, customFormat())
	log.Root().SetHandler(h)
}

func main() {
	// Environment variables
	flag.Parse()
	staticDir := os.Getenv("THYRA_STATIC")
	if len(staticDir) == 0 {
		pwd, _ := os.Getwd()
		staticDir = filepath.Join(pwd, "static")
		log.Info("Set THYRA_STATIC if you wish to configure the directory for static content")
	}
	log.Info(fmt.Sprintf("Using %s for static content", staticDir))

	// Flags
	port := flag.Int64("port", 4000, "Port to listen on incoming connections")
	//flag.Parse()

	// Setup and start the server
	server := game.NewServer(staticDir)

	if err := server.LoadConfig(); err != nil {
		os.Exit(1)
	}

	if err := server.LoadAreas(); err != nil {
		os.Exit(1)
	}

	roomsMap := make(map[string]map[string][][]game.Cube)
	for _, area := range server.Areas {
		roomsMap[area.Name] = make(map[string][][]game.Cube)
		for _, room := range area.Rooms {
			roomsMap[area.Name][room.Name] = server.CreateRoom_as_cubes(area.Name, room.Name)
		}
	}

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Info(err.Error())
		os.Exit(1)
	}
	log.Info(fmt.Sprintf("Listen on: %s", ln.Addr()))

	var wg sync.WaitGroup
	quit := make(chan struct{})
	regRequest := make(chan game.LoginRequest, 1000)
	clientRequest := make(chan game.ClientRequest, 1000)

	wg.Add(1)
	go handleRegistrations(*server, wg, quit, regRequest)

	wg.Add(1)
	go acceptConnections(ln, server, wg, quit, clientRequest, regRequest)

	wg.Add(1)
	go broadcast(*server, wg, quit, clientRequest, roomsMap)

	wg.Wait()
}

// handleRegistrations accepts requests for registration and replies back if the requested
// username exists or not.
func handleRegistrations(server game.Server, wg sync.WaitGroup, quit chan struct{}, regRequest chan game.LoginRequest) {
	defer wg.Done()

	for {
		exists := false
		var err error

		select {
		case <-quit:
			return
		case request := <-regRequest:
			exists, err = server.LoadPlayer(request.Username)
			if err != nil {
				io.WriteString(request.Conn, fmt.Sprintf("%s\n", err.Error()))
				continue
			}

			select {
			case request.Reply <- exists:
			case <-quit:
				return
			}

		}
	}
}

func acceptConnections(
	ln net.Listener,
	server *game.Server,
	wg sync.WaitGroup,
	quit <-chan struct{},
	clientCh chan<- game.ClientRequest,
	regRequest chan game.LoginRequest,
) {

	defer wg.Done()

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Info(err.Error())
			continue
		}
		wg.Add(1)
		go handleConnection(conn, server, wg, quit, clientCh, regRequest)

		select {
		case <-quit:
			return
		default:
		}
	}
}

// handleConnection should be invoked as a goroutine.
func handleConnection(
	c net.Conn,
	server *game.Server,
	wg sync.WaitGroup,
	quit <-chan struct{},
	clientCh chan<- game.ClientRequest,
	regRequest chan<- game.LoginRequest,
) {

	defer wg.Done()

	bufc := bufio.NewReader(c)
	defer c.Close()

	log.Info(fmt.Sprintf("New connection open: %s", c.RemoteAddr()))

	io.WriteString(c, WelcomePage)

	var username string
	questions := 0

out:
	for {
		if questions >= 3 {
			io.WriteString(c, "See you\n")
			return
		}

		username = promptMessage(c, bufc, "Whats your Nick?\n")
		isValidName := game.IsValidUsername(username)
		if !isValidName {
			questions++
			io.WriteString(c, fmt.Sprintf("Username %s is not valid (0-9a-z_-).\n", username))
			continue
		}

		exists := false
		replyCh := make(chan bool, 1)

		select {
		case regRequest <- game.LoginRequest{Username: username, Conn: c, Reply: replyCh}:
		case <-quit:
		}

		select {
		case exists = <-replyCh:
		case <-quit:
		}

		if exists {
			break out
		}

		questions++
		io.WriteString(c, fmt.Sprintf("Username %s does not exists.\n", username))
		answer := promptMessage(c, bufc, "Do you want to create that user? [y|n] ")

		if answer == "y" || answer == "yes" {
			server.CreatePlayer(username)
			break
		}
	}

	player, _ := server.GetPlayerByNick(username)

	reply := make(chan game.Reply, 1)

	client := game.NewClient(c, &player, clientCh, reply)

	go game.Panel(client)

	log.Info(fmt.Sprintf("Player %q got connected", client.Player.Nickname))
	server.ClientLoggedIn(client.Player.Nickname, *client)
	client.ReadLinesInto(quit)
	log.Info(fmt.Sprintf("Connection from %v closed.", c.RemoteAddr()))
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
func broadcast(
	server game.Server,
	wg sync.WaitGroup,
	quit <-chan struct{},
	reqChan <-chan game.ClientRequest,
	roomsMap map[string]map[string][][]game.Cube,
) {

	defer wg.Done()

	go god(&server, roomsMap)

	for {
		select {
		case request := <-reqChan:
			server.HandleCommand(*request.Client, request.Cmd, roomsMap)
		case <-quit:
			return
		}
	}
}

func god(s *game.Server, map_array map[string]map[string][][]game.Cube) {

	for {
		select {
		case ev := <-s.Events:
			c := ev.Client
			switch ev.Etype {

			case "look":
				godPrint(s, c, map_array, "")
			case "move_east":
				msg := ""
				if !do_move(s, *c, map_array, 0) {
					msg = "You can't go that way"
				}
				godPrint(s, c, map_array, msg)

			case "move_west":
				msg := ""
				if !do_move(s, *c, map_array, 1) {
					msg = "You can't go that way"
				}
				godPrint(s, c, map_array, msg)
			case "move_north":
				msg := ""
				if !do_move(s, *c, map_array, 2) {
					msg = "You can't go that way"
				}
				godPrint(s, c, map_array, msg)
			case "move_south":
				msg := ""
				if !do_move(s, *c, map_array, 3) {
					msg = "You can't go that way"
				}
				godPrint(s, c, map_array, msg)

			case "quit":
				s.OnExit(*c)
				c.Conn.Close()
			}

		}
	}

}

func godPrint(s *game.Server, client *game.Client, roomsMap map[string]map[string][][]game.Cube, msg string) {

	room := client.Player.Room
	preroom := client.Player.PreviousRoom
	map_array := roomsMap[client.Player.Area][client.Player.Room]
	map_array_pre := roomsMap[client.Player.PreviousArea][client.Player.PreviousRoom]

	var onlineSameRoom []game.Client
	var previousSameRoom []game.Client
	online := s.OnlineClients()
	for i := range online {
		c := online[i]

		if c.Player.Room == room {
			onlineSameRoom = append(onlineSameRoom, c)
		} else if c.Player.Room == preroom {
			previousSameRoom = append(previousSameRoom, c)
		}

	}
	p := client.Player

	log.Info("Online players in the same room:")
	for i := range onlineSameRoom {

		c := onlineSameRoom[i]

		log.Info(fmt.Sprintf("%s", c.Player.Nickname))
		buffintro := game.PrintIntro(s, p.Area, p.Room)
		bufmap := game.UpdateMap(s, p, map_array)
		bufexits := game.PrintExits(game.FindExits(map_array, c.Player.Area, c.Player.Room, c.Player.Position))

		reply := game.Reply{
			World: bufmap.Bytes(),
			Intro: buffintro.Bytes(),
			Exits: bufexits.String(),
		}

		if c.Player.Nickname == p.Nickname {
			reply.Events = msg
		}

		c.Reply <- reply
	}

	for i := range previousSameRoom {
		c := previousSameRoom[i]

		buffexits := game.PrintExits(game.FindExits(map_array_pre, c.Player.Area, c.Player.Room, c.Player.Position))

		bufmap := game.UpdateMap(s, c.Player, map_array_pre)
		buffintro := game.PrintIntro(s, c.Player.Area, c.Player.Room)

		reply := game.Reply{
			World: bufmap.Bytes(),
			Intro: buffintro.Bytes(),
			Exits: buffexits.String(),
		}

		if c.Player.Nickname == p.Nickname {
			reply.Events = msg
		}

		c.Reply <- reply
	}

}

func do_move(s *game.Server, c game.Client, roomsMap map[string]map[string][][]game.Cube, direction int) bool {

	c.Player.PreviousArea = c.Player.Area
	c.Player.PreviousRoom = c.Player.Room

	map_array := roomsMap[c.Player.Area][c.Player.Room]
	newpos, _ := strconv.Atoi(game.FindExits(map_array, c.Player.Area, c.Player.Room, c.Player.Position)[direction][1])
	posarray := game.FindExits(map_array, c.Player.Area, c.Player.Room, c.Player.Position)

	if newpos > 0 {

		c.Player.Position = strconv.Itoa(newpos)
		delete(s.Players, c.Player.Nickname)
		c.Player.Area = posarray[direction][0]
		c.Player.Room = posarray[direction][2]
		s.Players[c.Player.Nickname] = *c.Player
		return true
	} else {
		return false
	}

}
