package server

import "bytes"

type Screen struct {
	width          int
	height         int
	exitCanvas     []rune
	messagesCanvas []rune
	mapCanvas      [][]rune
	introCanvas    [][]rune
	screenRunes    [][]rune
	screenColors   [][]ID // the player's view of the screen
}

// Initialize new Screen
func NewScreen(width, height int) *Screen {
	screenRunes := make([][]rune, height)
	screenColors := make([][]ID, height)
	for h := 0; h < height-3; h++ {

		screenRunes[h] = make([]rune, width)
		screenColors[h] = make([]ID, width)

		for w := 0; w < width; w++ {

			screenRunes[h][w] = ' '
			screenColors[h][w] = ID(255)
		}
	}

	return &Screen{
		width:          width,
		height:         height,
		exitCanvas:     make([]rune, 0),
		messagesCanvas: make([]rune, 0),
		mapCanvas:      make([][]rune, 0),
		introCanvas:    make([][]rune, 0),
		screenRunes:    screenRunes,
		screenColors:   screenColors,
	}

}

// TODO : Check for offsets. Add limitation to all Canvas
func (scr *Screen) updateScreenRunes(frame string, bufToUpdate bytes.Buffer) {
	runes := make([]rune, 0)
	buf := bytes.NewBuffer(bufToUpdate.Bytes())

	switch frame {
	case "map":
		for {
			char, _, err := buf.ReadRune()
			if err != nil {
				break
			}
			if char == '\n' {
				scr.mapCanvas = append(scr.mapCanvas, runes)
				runes = []rune{}
			} else {
				runes = append(runes, char)
			}
		}

	case "exits":
		for {
			char, _, err := buf.ReadRune()
			if err != nil {
				break
			}
			if char == '\n' {
				scr.exitCanvas = runes
				runes = []rune{}
			} else {
				runes = append(runes, char)
			}
		}

	case "intro":
		for {
			char, _, err := buf.ReadRune()
			if err != nil {
				break
			}
			if char == '\n' {
				scr.introCanvas = append(scr.introCanvas, runes)
				runes = []rune{}
			} else {
				runes = append(runes, char)
			}
		}

	case "message":
		for {
			char, _, err := buf.ReadRune()
			if err != nil {
				break
			}
			if char == '\n' {
				scr.messagesCanvas = runes
				runes = []rune{}
			} else {
				runes = append(runes, char)
			}
		}
	}
}
