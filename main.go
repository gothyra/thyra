package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/gothyra/thyra/game"
)

func main() {
	// Environment variables
	staticDir := os.Getenv("THYRA_STATIC")
	if len(staticDir) == 0 {
		pwd, _ := os.Getwd()
		staticDir = filepath.Join(pwd, "static")
		log.Println("Set THYRA_STATIC if you wish to configure the directory for static content")
	}
	log.Printf("Using %s for static content\n", staticDir)

	// Flags
	port := flag.Int64("port", 4000, "Port to listen on incoming connections")
	flag.Parse()

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
		log.Println(err.Error())
		os.Exit(1)
	}

	log.Printf("Listen on: %s", ln.Addr())

	msgchan := make(chan string)
	addchan := make(chan game.Client)
	rmchan := make(chan game.Client)

	go handleMessages(msgchan, addchan, rmchan)

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}

		go handleConnection(conn, msgchan, addchan, rmchan, server, roomsMap)
	}
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

func handleConnection(
	c net.Conn,
	msgchan chan<- string,
	addchan chan<- game.Client,
	rmchan chan<- game.Client,
	server *game.Server,
	roomsMap map[string]map[string][][]game.Cube,
) {
	bufc := bufio.NewReader(c)
	defer c.Close()

	log.Println("New connection open:", c.RemoteAddr())

	io.WriteString(c, WelcomePage)

	var nickname string
	questions := 0
	for {
		if questions >= 3 {
			io.WriteString(c, "See you\n\r")
			return
		}

		nickname = promptMessage(c, bufc, "Whats your Nick?\n")
		isValidName := server.IsValidUsername(nickname)
		if !isValidName {
			questions++
			io.WriteString(c, fmt.Sprintf("Username %s is not valid (0-9a-z_-).\n", nickname))
			continue
		}

		exists, err := server.LoadPlayer(nickname)
		if err != nil {
			io.WriteString(c, fmt.Sprintf("%s\n\r", err.Error()))
			return
		}
		if exists {
			break
		}

		questions++
		io.WriteString(c, fmt.Sprintf("Username %s does not exists.\n", nickname))
		answer := promptMessage(c, bufc, "Do you want to create that user? [y|n] ")

		if answer == "y" || answer == "yes" {
			server.CreatePlayer(nickname)
			break
		}
	}

	player, playerLoaded := server.GetPlayerByNick(nickname)

	if !playerLoaded {
		log.Println("problem getting user object")
		io.WriteString(c, "Problem getting user object\n")
		return
	}

	cmdCh := make(chan string)
	client := game.NewClient(c, &player, cmdCh)

	if strings.TrimSpace(client.Nickname) == "" {
		log.Println("invalid username")
		io.WriteString(c, "Invalid Username\n")
		return
	}

	// Register user
	addchan <- client
	defer func() {
		msgchan <- fmt.Sprintf("User %s left the chat room.\n", client.Nickname)
		log.Printf("Connection from %v closed.\n", c.RemoteAddr())
		rmchan <- client
	}()
	io.WriteString(c, fmt.Sprintf("Welcome, %s!\n", client.Player.Nickname))
	server.ClientLoggedIn(client)

	// I/O
	go client.ReadLinesInto(msgchan, server)
	for {
		select {
		case cmd := <-cmdCh:
			server.HandleCommand(client, cmd, roomsMap)
		}
	}
}

func handleMessages(msgchan <-chan string, addchan <-chan game.Client, rmchan <-chan game.Client) {
	clients := make(map[net.Conn]chan<- string)

	for {
		select {
		case msg := <-msgchan:
			log.Printf("New message: %s", msg)
			for _, ch := range clients {
				go func(mch chan<- string) { mch <- "\033[1;33;40m" + msg + "\033[m\n\r" }(ch)
			}
		case client := <-addchan:
			log.Printf("New client: %v\n\r\n\r", client.Conn)
			clients[client.Conn] = client.Cmd
		case client := <-rmchan:
			log.Printf("Client disconnects: %v\n\r\n\r", client.Conn)
			delete(clients, client.Conn)
		}
	}
}
