package game

import (
	"fmt"
	"math/rand"
	"time"
)

/*
-Δημιουργία χαρακτήρων με βάση τον αλγόριθμο κουπονιών-

Η ιδέα είναι απλή. Έχουμε τα εξής κουπόνια:
4 που έχουν τον αριθμό 3
4 που έχουν τον αριθμό 4
4 που έχουν τον αριθμό 5
4 που έχουν τον αριθμό 6
Δύο άδεια κουπόνια

Ρίχνουμε ένα εξάπλευρο ζάρι και ότι φέρει, βάζουμε τους πόντους στο πρώτο άδειο κουπόνι και τους υπόλοιπους στον δεύτερο άδειο κουπόνι.
Δηλαδή, αν φέρω 2 στο εξάπλευρο, Θα έχει το πρώτο αξία 2 και ο δεύτερο 4. Κατόπιν, ανακατεύω τηα κουπόνια και τραβάω στη τύχη 6 φορές τρία μαζί
κουπόνια. Το σύνολο τους είναι και το χαρακτηριστικό του χαρακτήρα. Έτσι, έχουμε μέσο όρο 14 σε κάθε χαρακτηριστικό (STR, DEX, CON etc.)
και σύνολο 78 πόντων να μοιραστούν στα 6 χαρακτηριστικά.
*/
// ------------Standard values-----------
type PC struct {
	STR int `xml:"str"`
	DEX int `xml:"dex"`
	CON int `xml:"con"`
	INT int `xml:"int"`
	WIS int `xml:"wis"`
	CHA int `xml:"cha"`
}

func random(min, max int) int {
	max = max + 1
	rand.Seed(time.Now().UTC().UnixNano()) // Μετράει πολύ μια sleep τελικά
	return rand.Intn(max-min) + min
}

func create_character() {
	tokens := []int{3, 3, 3, 3, 4, 4, 4, 4, 5, 5, 5, 5, 6, 6, 6, 6, 0, 0} // Τα κουπόνια με τις τιμές τους
	tokens[16] = random(1, 6)                                             // Το πρώτο άδειο κουπόνι, παίρνει τιμή από 1 ως 6
	tokens[17] = 6 - tokens[16]                                           // Το δεύτερο άδειο κουπόνι, παίρνει ότι περισσέψει

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

		for i := 0; i < 3; i++ { // Αυτή η for, διαλέγει ένα κουπόνι τυχαία από το slice, το αποθηκεύει στο χαρακτηριστικο
			numb := rand.Intn(len(tokens))     // και μετά το σμπρώχνει στο τέλος του slice. Μετά, επαναπροσδιορίζουμε όλο το
			time.Sleep(100 * time.Millisecond) // slice χωρίς το τελικό στοιχείο.
			*attribute += tokens[numb]
			tokens[numb] = tokens[len(tokens)-1]
			tokens = tokens[:len(tokens)-1]

			for j := 0; j < len(tokens); j++ {
				fmt.Print(tokens[j], " ")
			}
			fmt.Println("\nSTR = ", *attribute)

		}
		sum += *attribute
	}

	fmt.Println("\nSum of attributes = ", sum, "\n")
	for i := 0; i < 20; i++ {
		fmt.Print("-")
	}
	// Από εδώ και κάτω είναι τι μας ενδιαφέρει να φαίνεται στον παίχτη
	fmt.Println("\nAnonymous character has the following attributes:")
	fmt.Println("STR = ", player.STR)
	fmt.Println("DEX = ", player.DEX)
	fmt.Println("CON = ", player.CON)
	fmt.Println("INT = ", player.INT)
	fmt.Println("WIS = ", player.WIS)
	fmt.Println("CHA = ", player.CHA)

}


