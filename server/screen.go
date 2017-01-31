package server

import (
	"bytes"
	"fmt"

	log "gopkg.in/inconshreveable/log15.v2"
)

type Screen struct {
	width        int
	height       int
	exitCanvas   [][]rune
	mapCanvas    [][]rune
	introCanvas  [][]rune
	screenRunes  [][]rune
	screenColors [][]ID // the player's view of the screen
}

func NewScreen(width, height int) *Screen {
	log.Info(fmt.Sprintf("SCREEN : Width :%d  Height:%d", width, height))

	screenRunes := make([][]rune, width)
	screenColors := make([][]ID, width)
	for w := 0; w < width; w++ {
		screenRunes[w] = make([]rune, height)
		screenColors[w] = make([]ID, height)

		for h := 0; h < height; h++ {
			screenRunes[w][h] = ' '
			screenColors[w][h] = ID(255)
		}
	}

	return &Screen{
		width:        width,
		height:       height,
		exitCanvas:   make([][]rune, 0),
		mapCanvas:    make([][]rune, 0),
		introCanvas:  make([][]rune, 0),
		screenRunes:  screenRunes,
		screenColors: screenColors,
	}

}

func (scr *Screen) updateScreen(frame string, bufToUpdate bytes.Buffer, height, width int) {

	switch frame {
	case "map":
		height = 1
		width = 1
		buf := bytes.NewBuffer(bufToUpdate.Bytes())
		counter := 0
		for {
			char, _, err := buf.ReadRune()
			if err != nil {
				// TODO: Log errors other than io.EOF
				// log.Info("world buffer read error: %v", err)
				break
			}
			if char == '\n' {
				height++
				width = 2
			} else {
				width++
			}

			scr.screenRunes[height][width] = char
			counter++
		}
	}

}
