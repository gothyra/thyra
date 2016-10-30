package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"

	log "gopkg.in/inconshreveable/log15.v2"
	"gopkg.in/inconshreveable/log15.v2/stack"

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
	flag.Parse()
}

// Flags
var port = flag.Int64("port", 4000, "Port to listen on incoming connections")

func main() {
	// Setup and start the server
	s := server.NewServer()
	s.Start(*port)
}
