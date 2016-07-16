package game

/*
Εξομοιωτης βασικης μαχης συμφωνα με τους κανονες της 3.5 εκδοσης
*/
import (
	"fmt"
	"math/rand"
	"strconv"
	"time"
)

// ------------Standard values-----------
type PC struct { //Τα στοιχεια του χαρακτήρα.
	STR        int    `xml:"str"`
	DEX        int    `xml:"dex"`
	CON        int    `xml:"con"`
	INT        int    `xml:"int"`
	WIS        int    `xml:"wis"`
	CHA        int    `xml:"cha"`
	BAB        int    `xml:"bab"`
	AC         int    `xml:"ac"`
	HP         int    `xml:"hp"`
	HD         int    `xml:"hd"`
	Weapondie  int    `xml:"weapondie"`
	Initiative int    `xml:"initiative"`
	Level      int    `xml:"level"`
	Class      string `xml:"class"`
	Armor      string `xml:"armor"`
	Weapon     string `xml:"weapon"`
}

func NewPC() *PC {
	player := &PC{
		STR:   generateAttrib(),
		DEX:   generateAttrib(),
		CON:   generateAttrib(),
		INT:   generateAttrib(),
		WIS:   generateAttrib(),
		CHA:   generateAttrib(),
		Level: 1,
		Class: assignClass(),
	}

	player.Armor, player.AC = wearArmor(player.DEX)
	player.HP = calcHP(player.Class, player.Level)
	player.BAB = calcBAB(player.Class, player.Level)
	player.Weapon, player.Weapondie = weildWeapon()
	player.Initiative = random(1, 20) + attrModifier(player.DEX)

	return player
}

//------------Functions----------------
// μια random οπως την ξερουμε
func random(min, max int) int {
	max = max + 1
	return rand.Intn(max-min) + min
}

// γενικη μεθοδος για να δημιουργουμε τα stats, δηλ. strength, constitution etc.
func generateAttrib() int {
	return random(8, 18)
}

// Βασικη μεθοδος υπολογισμου του attribute bonus. Θελει προβλεψη για τις αρνητικες τιμες, γιατι παει ανα δυο
// ποντους το αρνητικο bonus (9 και 8 attribute δινουν -1 κ.ο.κ.)
func attrModifier(attribute int) int {
	return (attribute - 10) / 2
}

// τωρα αυτη διαλεγει στην τυχη μια πανοπλια. Αργοτερα, απλα θα παιρνει το αναγνωριστικο της πανοπλιας απο την βαση δεδομενων
//και θα υπολογιζει το συνολο του AC
func wearArmor(dexterity int) (string, int) {
	lottery := random(1, 5)
	var armorname string
	var armorBonus, dexBonus int
	dexBonus = attrModifier(dexterity)
	switch lottery {
	case 1:
		armorname = "Leather Armor"
		armorBonus = 2
		if dexBonus > 8 { // καθε πανοπλια εχει κατωφλι στους ποσους ποντους dexterity modifier μπορουν να προστεθουν
			dexBonus = 8
		}
	case 2:
		armorname = "Chain Shirt"
		armorBonus = 4
		if dexBonus > 4 {
			dexBonus = 4
		}
	case 3:
		armorname = "Scale Mail"
		armorBonus = 4
		if dexBonus > 4 {
			dexBonus = 4
		}
	case 4:
		armorname = "Breastplate"
		armorBonus = 5
		if dexBonus > 3 {
			dexBonus = 3
		}
	case 5:
		armorname = "Full Plate Armor"
		armorBonus = 8
		if dexBonus > 1 {
			dexBonus = 1
		}
	}
	return armorname, 10 + armorBonus + dexBonus
}

func weildWeapon() (string, int) {
	lottery := random(1, 5)
	var weapon string
	var weapondie int // Στην ουσια ειναι σαν το damroll που ελεγες Νικο οτι εχει το MUD αλλα πιο ξεκαθαρα τα πραγματα
	switch lottery {
	case 1:
		weapon = "fist"
		weapondie = 3
	case 2:
		weapon = "dagger"
		weapondie = 4
	case 3:
		weapon = "short sword"
		weapondie = 6
	case 4:
		weapon = "longsword"
		weapondie = 8
	case 5:
		weapon = "greataxe"
		weapondie = 12 // δηλαδη, το ζαρι που ριχνεις για να κανεις damage με το πελεκυ ειναι δωδεκαπλευρο, 1d12
	}
	return weapon, weapondie
}

func assignClass() string { //Τρεις κλασσεις για αρχη και βλεπουμε
	lottery := random(1, 3)
	var class string
	switch lottery {
	case 1:
		class = "Commoner"
	case 2:
		class = "Fighter"
	case 3:
		class = "Rogue"
	}
	return class
}

func calcBAB(class string, level int) int {
	// Οι πινακες για το Base Attack Bonus που ειναι για καθε κλασση βγαινουν βαση αλγοριθμου
	//Εχει και προβλεψη για αν βαλουμε μεγαλυτερα level
	BAB := 0
	switch class {
	case "Commoner":
		BAB = level / 2
	case "Fighter":
		BAB = level
	case "Rogue":
		BAB = (3 * level) / 4
	}
	return BAB
}

func calcHP(class string, level int) int { // Εχει και προβλεψη για αν βαλουμε μεγαλυτερα level
	var HP int
	var HD int
	switch class {
	case "Commoner":
		HD = 4
		HP = HD
		level -= 1
		if level != 0 {
			for i := 0; i < level; i++ {
				HP += random(1, HD)
			}
		}
	case "Fighter":
		HD = 10
		HP = HD
		level = level - 1
		if level != 0 {
			for i := 0; i < level; i++ {
				HP += random(1, HD)
			}
		}
	case "Rogue":
		HD = 6
		HP = HD
		level = level - 1
		if level != 0 {
			for i := 0; i < level; i++ {
				HP += random(1, HD)
			}
		}
	}
	return HP
}

// Μεθοδος μαχης. Πρωτα βαραει ο comb1 και μετα ο comb2. Το initiative καθοριζεται στην main()
// δοκιμασα "for comb1.HP > 0 || comb2.HP > 0 {" και κανει οτι να'ναι. Γιατι; Για τωρα δουλευει
//  με αρχικο check των hit points σε ατερμονα βρογχο
func fight(c Client, comb1, comb2 *PC) {
	for comb1.HP > 0 && comb2.HP > 0 {
		if (random(1, 20) + comb1.BAB + attrModifier(comb1.STR)) >= comb2.AC {
			hit := random(1, comb1.Weapondie)
			comb2.HP -= hit
			descrip := random(1, 4)

			strhit := strconv.Itoa(hit)

			switch descrip {
			case 1:
				c.WriteLineToUser("Player 2 was hit for " + strhit + " points of damage")
			case 2:
				c.WriteLineToUser("Player 2 was too slow, punished for " + strhit + " points of damage")
			case 3:
				c.WriteLineToUser("The evasion was worthless for Player 2, he suffered " + strhit + " points of damage")
			case 4:
				c.WriteLineToUser("If he brought a shield, Player 2 would avoid " + strhit + " points of damage")
			}

		} else {
			c.WriteLineToUser("Player 1 missed")
		}
		if comb2.HP < 0 {
			break
		}
		if (random(1, 20) + comb2.BAB + attrModifier(comb2.STR)) >= comb1.AC {
			hit := random(1, comb2.Weapondie)
			comb1.HP -= hit
			descrip := random(1, 4)

			strhit := strconv.Itoa(hit)

			switch descrip {
			case 1:
				c.WriteLineToUser("Player 1 was hit for " + strhit + " points of damage")
			case 2:
				c.WriteLineToUser("Bad news Player 1, you were hit for " + strhit + " points of damage")
			case 3:
				c.WriteLineToUser("Player 1 surelly didn't expect to suffer " + strhit + " points of damage")
			case 4:
				c.WriteLineToUser("Learn some parry next time Player 1, because you took " + strhit + " points of damage")
			}
		} else {
			c.WriteLineToUser("Player 2 missed")
		}
	}
	fmt.Println("--@@--@@--@@--@@--")

	strhp1 := strconv.Itoa(comb1.HP)
	strhp2 := strconv.Itoa(comb2.HP)
	if comb1.HP > 0 {
		c.WriteLineToUser("Player 1 won, he's at " + strhp1 + " Hit Points, leaving his opponent at " + strhp2)
	} else {
		c.WriteLineToUser("Player 2 won, he's at " + strhp2 + "Hit Points, leaving his opponent at " + strhp1)
	}
}

//------Main code------

func do_fight(c Client) {
	rand.Seed(time.Now().Unix())

	// Setting up player 1
	player1 := NewPC()
	// Setting up player 2
	player2 := NewPC()

	// τελικο output
	fmt.Println("-----@@@@@@----@@@@@@@-----\nMy, what a characters you have here?\n-----@@@@@@----@@@@@@@-----")
	fmt.Println("Player 1, which is a ", player1.Class, ", with ", player1.HP, "HP",
		"his strength is", player1.STR,
		"his dexterity is", player1.DEX,
		"his constitution is", player1.CON,
		"his intelligence is ", player1.INT,
		"his wisdom is ", player1.WIS,
		"and his charisma is", player1.CHA,
		"He's wearing a ", player1.Armor, "providing him AC=", player1.AC, "and carries a ", player1.Weapon)
	fmt.Println("He rolled initiative", player1.Initiative)
	fmt.Println("-----------------------------------------------------------------------------------------------------")
	fmt.Println("Player 2, which is a ", player2.Class, ", with ", player2.HP, "HP",
		"his strength is", player2.STR,
		"his dexterity is", player2.DEX,
		"his constitution is", player2.CON,
		"his intelligence is ", player2.INT,
		"his wisdom is ", player2.WIS,
		"and his charisma is", player2.CHA,
		"He's wearing a ", player2.Armor, "providing him AC=", player2.AC, "and carries a ", player2.Weapon)
	fmt.Println("He rolled initiative", player2.Initiative)
	fmt.Println("----------------------\nLET THE FIGHT BEGIN!\n----------------------")

	//Υπολογισμός initiative, σε περιπτωση ισοπαλιας ξαναριχνουν ζαρια, αλλιως τοποθετουνται με αντιστοιχια στην μεθοδο fight()
	for player1.Initiative == player2.Initiative {
		player1.Initiative = random(1, 20) + attrModifier(player1.DEX)
		player2.Initiative = random(1, 20) + attrModifier(player2.DEX)
	}

	switch {
	case player1.Initiative > player2.Initiative:
		fight(c, player1, player2)
	case player1.Initiative < player2.Initiative:
		fight(c, player2, player1)
	default:
		fmt.Println("Problem!")
	}
}
