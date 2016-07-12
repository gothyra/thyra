package game

type Level struct {
	Key   string `xml:"key,attr"`
	Tag   string `xml:"tag,attr"`
	Name  string `xml:"name"`
	Intro string `xml:"intro"`
	Cubes []Cube `xml:"cubes>cube"`
}

//TODO : Add Rooms in area file , Cubes must belong to a specific room, and many rooms belong to area.
type Cube struct {
	ID       string `xml:"id,attr"`
	POSX     string `xml:"posx,attr"`
	POSY     string `xml:"posy,attr"`
	ToArea   string `xml:"toarea,attr"`
	ToId     string `xml:"toid,attr"`
	FromExit string `xml:"fromexit,attr"`
}
