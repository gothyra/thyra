package game

import (
	"fmt"
	"math/rand"
	"time"
)

/*
-Character creation with use of token algorithm-

The idea is simple. We have these tokens with the values of:
4 tokens of value 3
4 tokens of value 4
4 tokens of value 5
4 tokens of value 6
two empty tokens

A six-side die is rolled. The result is stored in the first empty token, the remaining in the second. P.e. if the roll results
to 2, the first token will have the value of 2 and the second of 4. Then, the tokens are shuffled and 3 of them are drawed from
the stack for 6 times. Every sum of the drawn of the tokens determines the corresponding attribute, starting with Strength
and moving sequentially.
This algorithm ensures that there is an average of 14 points for every attribute and a total of 78 points to be disperced
to the 6 attributes.
*/
// ------------Standard values-----------

func create_character() {
	tokens := []int{3, 3, 3, 3, 4, 4, 4, 4, 5, 5, 5, 5, 6, 6, 6, 6, 0, 0} // Tokens and their values.
	tokens[16] = random(1, 6)                                             // The first empty token, taken a value from 1 to 6.
	tokens[17] = 6 - tokens[16]                                           // The second token, get's what is left from the first.

	player := &PC{
		STR: 0,
		DEX: 0,
		CON: 0,
		INT: 0,
		WIS: 0,
		CHA: 0,
	}

	sum := 0

	var attribute *int

	for h := 0; h < 6; h++ {

		switch {
		case h == 0:
			attribute = &player.STR
		case h == 1:
			attribute = &player.DEX
		case h == 2:
			attribute = &player.CON
		case h == 3:
			attribute = &player.INT
		case h == 4:
			attribute = &player.WIS
		case h == 5:
			attribute = &player.CHA

		}

		for i := 0; i < 3; i++ { // This loop picks a random token from the slice and stores it's value
			numb := rand.Intn(len(tokens))     // then it pushes it to the end of the slice. After that, it redefines the slice without
			time.Sleep(100 * time.Millisecond) // the final token.
			*attribute += tokens[numb]
			tokens[numb] = tokens[len(tokens)-1]
			tokens = tokens[:len(tokens)-1]

		}
		sum += *attribute
	}

	fmt.Println("\nSum of attributes = ", sum, "\n")
	for i := 0; i < 20; i++ {
		fmt.Print("-")
	}
	// Printing the results
	fmt.Println("\n\nAnonymous character has the following attributes:")
	fmt.Println("STR = ", player.STR)
	fmt.Println("DEX = ", player.DEX)
	fmt.Println("CON = ", player.CON)
	fmt.Println("INT = ", player.INT)
	fmt.Println("WIS = ", player.WIS)
	fmt.Println("CHA = ", player.CHA)

}
