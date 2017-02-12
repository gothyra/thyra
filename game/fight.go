package game

/*
Combat simulator based on SRD v3.5 rules
*/
import (
	"fmt"
	"math/rand"
	"strconv"
	"time"
)

// ------------Standard values-----------
type PC struct { //Character's attributes.
	STR        int    `toml:"str"`        //Strength of the character
	DEX        int    `toml:"dex"`        //Dexterity of the character
	CON        int    `toml:"con"`        //Constitution of the character
	INT        int    `toml:"int"`        //Intelligence of the character
	WIS        int    `toml:"wis"`        //Wisdomw of the character
	CHA        int    `toml:"cha"`        //Charisma of the character
	BAB        int    `toml:"bab"`        //Base attack Bonus of the character
	AC         int    `toml:"ac"`         //Armor Class of the character
	HP         int    `toml:"hp"`         //Hit points of the character
	HD         int    `toml:"hd"`         //Hit dice of the character
	Weapondie  int    `toml:"weapondie"`  //Type of multiside die of the weapon of the character
	Initiative int    `toml:"initiative"` //Indicates the initiative, who goes first in a turn-based battle
	Level      int    `toml:"level"`      //Level of the character
	Class      string `toml:"class"`      //Type of specialization of the character
	Armor      string `toml:"armor"`      //type of armor that the character wears
	Weapon     string `toml:"weapon"`     //type of weapon that the character weilds
}

/*
Executing the function generateAttrib(), a random value from 8 to 18 is assigned for each of the attribute of the
character and executing the function assignClass(), a random class from the three available is assigned.
*/
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
	/*
		Weapon and armor are assigned randomly to the characters, but Hit Points and BAB are based on algorithms
		according to the appropriate class
	*/
	player.Armor, player.AC = wearArmor(player.DEX)
	player.HP = calcHP(player.Class, player.Level)
	player.BAB = calcBAB(player.Class, player.Level)
	player.Weapon, player.Weapondie = weildWeapon()
	player.Initiative = random(1, 20) + attrModifier(player.DEX)

	return player
}

//------------Functions----------------
// Retused function that trully picks number from lowest to maximum
func random(min, max int) int {
	max = max + 1
	rand.Seed(time.Now().UTC().UnixNano()) // A sleep is vital for these calculations
	return rand.Intn(max-min) + min
}

// General attribute creation function
func generateAttrib() int {
	return random(8, 18)
}

/*
Basic function of calculating the attribute bonus. Negative values need tho shift by one lower, because the
negative bonus is one per two negative attribute points.
*/
func attrModifier(attribute int) int {
	return (attribute - 10) / 2
}

/*
This function picks an armor randomly. Then, it will calculate the total AC based on the armor's traits.
*/
func wearArmor(dexterity int) (string, int) {
	lottery := random(1, 5)
	var armorname string
	var armorBonus, dexBonus int
	dexBonus = attrModifier(dexterity)
	switch lottery {
	case 1:
		armorname = "Leather Armor"
		armorBonus = 2
		if dexBonus > 8 { // Every armor has a limit of how many dexterity bonus points can be added.
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

/*
This method provides a weapon to the character. The variable weapon is the name of the weapon and the variable weapondie
is the die that the weapon uses to calculate damage.
*/
func weildWeapon() (string, int) {
	lottery := random(1, 5)
	var weapon string
	var weapondie int
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
		weapondie = 12 // At this case, the Greataxe will deal random damage from 1 point to 12 points, a 12-side die.
	}
	return weapon, weapondie
}

/*
A function to assign a class randomly to the character. This is essential to calculate other factors, like HP etc.
*/
func assignClass() string { //To start with, three classes.
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

/*
Function for the Base Attack Bonus (BAB) calculation. This number is added with the 20-side die when a character
strike a blow to the opponent, to determine if he lands a hit or not.
*/
func calcBAB(class string, level int) int {
	// What BAB has every class at what level, depends of an algorithm.
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

/*
Function of Hit Points calculation. Depends of the class and the level of the character. Basically, every
class has a specific die, that rolls in every level up and adds the result to the sum of his maximum
hit points. In the first level, a character starts with the maximum number that this die can score.
*/
func calcHP(class string, level int) int {
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

/*
Battle function. First strikes the comb1 and then comb2. Initiative is determined in main function
Refactor of "for comb1.HP > 0 || comb2.HP > 0 {" gives fuzzy results. Don't know why.
*/
func fight(comb1, comb2 *PC) {
	for comb1.HP > 0 && comb2.HP > 0 {
		if (random(1, 20) + comb1.BAB + attrModifier(comb1.STR)) >= comb2.AC {
			hit := random(1, comb1.Weapondie)
			comb2.HP -= hit
			descrip := random(1, 4)

			strhit := strconv.Itoa(hit)

			switch descrip {
			case 1:
				fmt.Println("Player 2 was hit for " + strhit + " points of damage")
			case 2:
				fmt.Println("Player 2 was too slow, punished for " + strhit + " points of damage")
			case 3:
				fmt.Println("The evasion was worthless for Player 2, he suffered " + strhit + " points of damage")
			case 4:
				fmt.Println("If he brought a shield, Player 2 would avoid " + strhit + " points of damage")
			}

		} else {
			fmt.Println("Player 1 missed")
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
				fmt.Println("Player 1 was hit for " + strhit + " points of damage")
			case 2:
				fmt.Println("Bad news Player 1, you were hit for " + strhit + " points of damage")
			case 3:
				fmt.Println("Player 1 surelly didn't expect to suffer " + strhit + " points of damage")
			case 4:
				fmt.Println("Learn some parry next time Player 1, because you took " + strhit + " points of damage")
			}
		} else {
			fmt.Println("Player 2 missed")
		}
	}
	fmt.Println("--@@--@@--@@--@@--")

	strhp1 := strconv.Itoa(comb1.HP)
	strhp2 := strconv.Itoa(comb2.HP)
	if comb1.HP > 0 {
		fmt.Println("Player 1 won, he's at " + strhp1 + " Hit Points, leaving his opponent at " + strhp2)
	} else {
		fmt.Println("Player 2 won, he's at " + strhp2 + "Hit Points, leaving his opponent at " + strhp1)
	}
}

//------Main code------

func do_fight() {
	rand.Seed(time.Now().Unix())

	// Setting up player 1
	player1 := NewPC()
	// Setting up player 2
	player2 := NewPC()

	// final output
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

	// Initiative calculation, in case of a draw initiatives are rerolled, else they are assigned in accordance with the function fight()
	for player1.Initiative == player2.Initiative {
		player1.Initiative = random(1, 20) + attrModifier(player1.DEX)
		player2.Initiative = random(1, 20) + attrModifier(player2.DEX)
	}

	switch {
	case player1.Initiative > player2.Initiative:
		fight(player1, player2)
	case player1.Initiative < player2.Initiative:
		fight(player2, player1)
	default:
		fmt.Println("Problem!")
	}
}
