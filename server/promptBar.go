package server

const (
	ARROW_UP = iota + 65
	ARROW_DOWN
	ARROW_RIGHT
	ARROW_LEFT
)

const (
	ENTER_KEY     = 13
	SPACE_KEY     = 32
	BACKSPACE_KEY = 127
	DELETE_KEY    = 27
)

const (
	NUM_0 = iota + 48
	NUM_1
	NUM_2
	NUM_3
	NUM_4
	NUM_5
	NUM_6
	NUM_7
	NUM_8
	NUM_9
)

const (
	LOW_ALPHA = 97
	LOW_OMEGA = 122
)

const (
	UPPER_ALPHA = 65
	UPPER_OMEGA = 90
)

var (
	alphabet = []string{
		"a",
		"b",
		"c",
		"d",
		"e",
		"f",
		"g",
		"h",
		"i",
		"j",
		"k",
		"l",
		"m",
		"n",
		"o",
		"p",
		"q",
		"r",
		"s",
		"t",
		"u",
		"v",
		"w",
		"x",
		"y",
		"z"}

	//from 33 to 47
	specialChars1 = []string{
		"!",
		"\"",
		"#",
		"$",
		"%",
		"&",
		"'",
		"(",
		")",
		"*",
		"+",
		",",
		"-",
		".",
		"/",
	}

	//from 58 to 64
	specialChars2 = []string{
		":",
		";",
		"<",
		"=",
		">",
		"?",
		"@",
	}

	//from 91 to 95
	specialChars3 = []string{
		"[",
		"\\",
		"]",
		"^",
		"_",
		"`",
	}

	//from 123 to 126
	specialChars4 = []string{
		"{",
		"|",
		"}",
		"~",
	}
)
