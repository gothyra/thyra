package game

import (
	"bytes"
	"fmt"
)

var title = "-- Sokoban Level %d of %d --"

const debugConsoleColor = ColorBlack
const debugTextColor = ColorWhite
const textColor = ColorBlack
const backgroundColor = ColorBlue
const blockSize = 2
const viewStartX = 1
const viewStartY = 1
const titleStartX = viewStartX
const titleStartY = viewStartY
const boardStartX = viewStartX
const boardStartY = titleStartY + 2
const instructionStartY = boardStartY

var instructionStartX = 0

var tokenColor = map[byte]Attribute{
	'@': ColorWhite,
	'O': ColorYellow,
	'#': ColorRed,
	'X': ColorGreen,
	' ': backgroundColor,
}

const boxinTokenColor = ColorBlack

var instructions = []string{
	"Instructions:",
	"→ or l    :move right",
	"← or h    :move left",
	"↑ or k    :move up",
	"↓ or j    :move down",
	"     r    :reset",
	"     p    :previous level",
	"     n    :next level",
	"     d    :show debug console",
	"     esc  :quit",
	"",
	"The gola of this game to push all the boxes into the slot without been stuck somewhere.",
}

var colorInstructions = []struct {
	token byte
	text  string
}{
	{'@', "Player"},
	{'O', "Box"},
	{'#', "Wall"},
	{'X', "Slot"},
}

// this function renders debug console and debug messages
func renderDebugConsole(messages []string) {
	w, h := Size()

	for y := 0; y < h; y++ {
		for x := w / 2; x < w; x++ {
			SetCell(x, y, ' ', debugConsoleColor, debugConsoleColor)
		}
	}

	debugTextStartX := w/2 + 2
	for y, msg := range messages {
		printText(debugTextStartX, y+1, debugTextColor, debugConsoleColor, msg)
	}
}

func debugGameState(g *Game) {
	var text []string
	for i, cells := range g.board {
		var b bytes.Buffer
		for _, cell := range cells {
			b.WriteByte(cell.obj)
		}
		text = append(text, fmt.Sprintf("%-2d %s", i, b.String()))
	}
	text = append(text, " ")
	text = append(text, fmt.Sprintf("Where am I => X:%d, Y:%d", g.x, g.y))
	renderDebugConsole(text)
}

func render(g *Game, c Client) {
	Clear(backgroundColor, backgroundColor, c)

	printText(titleStartX, titleStartY, textColor, backgroundColor, fmt.Sprintf(title, g.level, g.db.maxLevel))
	if g.debug {
		debugGameState(g)
	}

	var maxWidth = 0
	for y, cells := range g.board {
		if maxWidth < len(cells) {
			maxWidth = len(cells)
		}
		for x, cel := range cells {
			for k := 0; k < blockSize; k++ {
				var cellColor = tokenColor[cel.obj]
				if cel.obj == BOX && cel.base == SLOT {
					cellColor = boxinTokenColor
				}
				SetCell(boardStartX+x*blockSize+k, boardStartY+y, ' ', cellColor, cellColor)
			}
		}
	}

	instructionStartX = maxWidth*blockSize + 10
	for y, msg := range instructions {
		printText(instructionStartX, instructionStartY+y, textColor, backgroundColor, msg)
	}

	for i, j := 0, 0; i < len(colorInstructions); i, j = i+1, j+2 {
		intr := colorInstructions[i]
		for k := 0; k < blockSize; k++ {
			SetCell(instructionStartX+k, instructionStartY+len(instructions)+j+1, ' ', tokenColor[intr.token], tokenColor[intr.token])
		}
		printText(instructionStartX+blockSize*2, instructionStartY+len(instructions)+j+1, textColor, backgroundColor, intr.text)
	}

	Flush(c)
}

func printText(x, y int, fg, bg Attribute, msg string) {
	for _, c := range msg {
		SetCell(x, y, c, fg, bg)
		x++
	}
}
