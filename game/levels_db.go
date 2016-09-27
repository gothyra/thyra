package game

import (
	"bufio"
	"io"
	"os"
	"strings"
)

const game_data_file = "sokoban_levels.txt"

type ldb struct {
	maxLevel int
	data     [][]string
}

// convert strings to 2-d matrix with bytes
func (db *ldb) getLevel(l int) [][]Cell2 {
	board := make([][]Cell2, len(db.data[l-1]))
	for i, str := range db.data[l-1] {
		board[i] = make([]Cell2, len(str))
		for j := 0; j < len(str); j++ {
			var c Cell2
			switch str[j] {
			case WALL:
				c = Cell2{base: WALL, obj: WALL}
			case FLOOR:
				c = Cell2{base: FLOOR, obj: FLOOR}
			case GUY:
				c = Cell2{base: FLOOR, obj: GUY}
			case SLOT:
				c = Cell2{base: SLOT, obj: SLOT}
			case BOX:
				c = Cell2{base: FLOOR, obj: BOX}
			case BOXIN:
				c = Cell2{base: SLOT, obj: BOX}
			}
			board[i][j] = c
		}
	}
	return board
}

// Read game database from .txt
func (db *ldb) loadAll() {
	f, err := os.Open(game_data_file)
	if err != nil {
		panic(err)
	}

	rd := bufio.NewReader(f)

	var l = 0
	var matrix = make([]string, 0)
	for {
		line, err := rd.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				db.data = append(db.data, matrix)
				db.maxLevel = len(db.data)
				return
			}
			panic(err)
		}
		line = strings.TrimRight(line, "\t\n\f\r")
		if len(line) == 0 {
			db.data = append(db.data, matrix)
			l = l + 1
			matrix = make([]string, 0)
		} else {
			matrix = append(matrix, line)
		}
	}
}
