package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"time"

	log "gopkg.in/inconshreveable/log15.v2"
	"gopkg.in/inconshreveable/log15.v2/stack"

	"github.com/gothyra/thyra/pkg/client"
	"github.com/gothyra/thyra/pkg/server"
)

func customFormat() log.Format {
	return log.FormatFunc(func(r *log.Record) []byte {
		var color = 0
		switch r.Lvl {
		case log.LvlCrit:
			color = 35
		case log.LvlError:
			color = 31
		case log.LvlWarn:
			color = 33
		case log.LvlInfo:
			color = 32
		case log.LvlDebug:
			color = 36
		}
		b := &bytes.Buffer{}
		call := stack.Call(r.CallPC[0])
		fmt.Fprintf(b, "\x1b[%dm%s\x1b[0m [%s %s:%d] %s\n", color, r.Lvl, r.Time.Format("2006-01-02|15:04:05.000"), call, call, r.Msg)
		return b.Bytes()
	})
}

func init() {
	h := log.StreamHandler(os.Stdout, customFormat())
	log.Root().SetHandler(h)
}

func main() {
	// Environment variables
	staticDir := os.Getenv("THYRA_STATIC")
	if len(staticDir) == 0 {
		pwd, _ := os.Getwd()
		staticDir = filepath.Join(pwd, "static")
		log.Warn("Set THYRA_STATIC if you wish to configure the directory for static content")
	}
	log.Info(fmt.Sprintf("Using %s for static content", staticDir))

	// Flags
	port := flag.Int64("port", 4000, "Port to listen on incoming connections")
	flag.Parse()

	// Setup and start the server
	s := server.NewServer(staticDir)

	if err := s.LoadConfig(); err != nil {
		os.Exit(1)
	}

	if err := s.LoadAreas(); err != nil {
		os.Exit(1)
	}

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
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
	go handleRegistrations(*s, wg, quit, regRequest)

	wg.Add(1)
	go acceptConnections(ln, s, wg, quit, clientRequest, regRequest)

	wg.Add(1)
	go broadcast(*s, wg, quit, clientRequest)

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
func handleRegistrations(s server.Server, wg *sync.WaitGroup, quit chan struct{}, regRequest chan client.LoginRequest) {
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
			exists, err = s.LoadPlayer(request.Username)
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
	s *server.Server,
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
	s *server.Server,
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
		isValidName := server.IsValidUsername(username)
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
	s.ClientLoggedIn(c.Player.Nickname, *c)

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
func broadcast(s server.Server, wg *sync.WaitGroup, quit <-chan struct{}, reqChan <-chan client.Request) {
	log.Info("broadcast started")
	defer wg.Done()

	wg.Add(1)
	go server.God(&s, wg, quit)

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
