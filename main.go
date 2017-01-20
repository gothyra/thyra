package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/droslean/thyraNew/server"

	log "gopkg.in/inconshreveable/log15.v2"
	"gopkg.in/inconshreveable/log15.v2/stack"
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

func main() {
	db, _ := server.NewDatabase(filepath.Join(os.TempDir(), "thyra.db"), true)

	idPool := make(chan server.ID, 100)
	for id := 1; id <= 100; id++ {
		idPool <- server.ID(id)
	}
	port := 3030

	s, _ := server.NewServer(db, port, idPool)
	server.StartServer(s)
}
