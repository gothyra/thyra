package game

import (
	"sort"
)

/*
Simple character attributes generator, rolling 4 six-sided dice, excluding the minor value of them and sum the rest three.
If the result is lower than 8, automatically is raised to this number.
*/

func create_character_dice() {

	player := &PC{
		STR: 0,
		DEX: 0,
		CON: 0,
		INT: 0,
		WIS: 0,
		CHA: 0,
	}

	dice := []int{0, 0, 0, 0}

	var attribute *int

	total := 0

	for j := 0; j < 6; j++ {

		switch {
		case j == 0:
			attribute = &player.STR
		case j == 1:
			attribute = &player.DEX
		case j == 2:
			attribute = &player.CON
		case j == 3:
			attribute = &player.INT
		case j == 4:
			attribute = &player.WIS
		case j == 5:
			attribute = &player.CHA
		}

		for i := 0; i < 4; i++ {
			dice[i] = random(1, 6)
		}

		sort.Ints(dice)
		//fmt.Println(dice)

		total = dice[1] + dice[2] + dice[3] // Here we discard the first "die", as is the lowest number.
		if total < 8 {
			for total < 8 {
				total += 1
			}
		}

		*attribute = total
		//fmt.Println(j, " is ", total)
	}
}
