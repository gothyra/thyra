package game

type Level struct {
	Key   string `xml:"key,attr"`
	Tag   string `xml:"tag,attr"`
	Name  string `xml:"name"`
	Intro string `xml:"intro"`
	Cubes []Cube `xml:"cubes>cube"`
}

type Cube struct {
	ID   string `xml:"id,attr"`
	POSX string `xml:"posx,attr"`
	POSY string `xml:"posy,attr"`
}
