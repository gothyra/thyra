package game

import (
	"sort"
)

/* Απλό generator των attributes, όπου ρίχνουμε 4 εξάπλευρα ζάρια και επιλέγουμε τα 3 καλύτερα για να γίνει το αντίστοιχο attribute.
   Σε περίπτωση που το αποτέλεσμα είναι κάτω του 8, γίνεται 8 αυτόματα
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

		total = dice[1] + dice[2] + dice[3] // Εδώ απορρίπτουμε το πρώτο ζάρι, γιατί είναι πάντα το μικρότερο
		if total < 8 {
			for total < 8 {
				total += 1
			}
		}

		*attribute = total
		//fmt.Println(j, " is ", total)
	}
}
