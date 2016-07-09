package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"

	"github.com/gothyra/thyra/game"
)

func main() {
	workingdir, _ := os.Getwd()

	log.Printf("Leveldir %s", workingdir+"/static/levels/")

	server := game.NewServer(workingdir)
	err := server.LoadLevels()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	ln, err := net.Listen("tcp", server.Config.Interface)
	if err != nil {
		fmt.Println(err)
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

		go handleConnection(conn, msgchan, addchan, rmchan, server)
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

func handleConnection(c net.Conn, msgchan chan<- string, addchan chan<- game.Client, rmchan chan<- game.Client, server *game.Server) {
	bufc := bufio.NewReader(c)
	defer c.Close()

	log.Println("New connection open:", c.RemoteAddr())
	io.WriteString(c, server.Config.Motd)

	var nickname string
	questions := 0
	for {
		if questions >= 3 {
			io.WriteString(c, "See you\n\r")
			return
		}

		nickname = promptMessage(c, bufc, "Whats your Nick?\n\r  ")
		isValidName := server.IsValidUsername(nickname)
		if !isValidName {
			questions++
			io.WriteString(c, fmt.Sprintf("Username %s is not valid (0-9a-z_-).\n\r", nickname))
			continue
		}

		ok := server.LoadPlayer(nickname)

		if ok == false {
			questions++
			io.WriteString(c, fmt.Sprintf("Username %s does not exists.\n\r", nickname))
			answer := promptMessage(c, bufc, "Do you want to create that user? [y|n] ")

			if answer == "y" {
				server.CreatePlayer(nickname)

				break
			}
		}

		if ok == true {
			break
		}
	}

	player, playerLoaded := server.GetPlayerByNick(nickname)

	if !playerLoaded {
		log.Println("problem getting user object")
		io.WriteString(c, "Problem getting user object\n")
		return
	}

	client := game.NewClient(c, player)

	if strings.TrimSpace(client.Nickname) == "" {
		log.Println("invalid username")
		io.WriteString(c, "Invalid Username\n")
		return
	}

	// Register user
	addchan <- client
	defer func() {
		msgchan <- fmt.Sprintf("User %s left the chat room.\n\r", client.Nickname)
		log.Printf("Connection from %v closed.\n", c.RemoteAddr())
		rmchan <- client
	}()
	io.WriteString(c, fmt.Sprintf("Welcome, %s!\n\n\r", client.Player.Nickname))
	server.PlayerLoggedIn(client.Nickname)

	// I/O
	go client.ReadLinesInto(msgchan, server)
	client.WriteLinesFrom(client.Ch)
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
			clients[client.Conn] = client.Ch
		case client := <-rmchan:
			log.Printf("Client disconnects: %v\n\r\n\r", client.Conn)
			delete(clients, client.Conn)
		}
	}
}
