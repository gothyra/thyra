package game

import "strings"

// Player holds all variables for a character.
type Player struct {
	Nickname string `xml:"nickname"`
	PC

	Position string `xml:"position"`
	Area     string `xml:"area"`
	RoomID   string `xml:"roomid"`

	Ch        chan string `xml:"-"`
	ActionLog []string
}

func (p *Player) LogAction(action string) {
	if !p.HasAction(action) {
		p.ActionLog = append(p.ActionLog, strings.ToLower(action))
	}
}

func (p *Player) HasAction(action string) bool {
	for _, a := range p.ActionLog {
		if strings.ToLower(a) == strings.ToLower(action) {
			return true
		}
	}
	return false
}
