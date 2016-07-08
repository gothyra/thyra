package game

import (

"fmt"
"io"
"os"
"encoding/xml"
"path/filepath"
)


type Level struct {
	Key         string      `xml:"key,attr"`
	Tag         string      `xml:"tag,attr"`
	Name        string      `xml:"name"`
	Cubes  []Cube `xml:"cubes>cube"`



}


type Cube struct {
	ID      string   `xml:"id,attr"`
	POSX    string   `xml:"posx,attr"`
	POSY   string   `xml:"posy,attr"`

}




type XMLCube struct {
    XMLName  xml.Name `xml:"cube"`
    ID      string   `xml:"id,attr"`
    POSX    string   `xml:"posx,attr"`
    POSY   string   `xml:"posy,attr"`

}

type XMLCubes struct {
    XMLName  xml.Name    `xml:"cubes"`
    Cubes   []XMLCube `xml:"cube"`
}

func ReadCubes(reader io.Reader) ([]XMLCube, error) { // Edw kane read ola ta cubes apo to XML. Auti i methodos einai LATHOS, alla to exw kanei proswrina
    var xmlCubes XMLCubes 								// Me ton Mixali eipame oti tha xrisimopoiisoume diaforetiko tropo.
    if err := xml.NewDecoder(reader).Decode(&xmlCubes); err != nil {
        return nil, err
    }

    return xmlCubes.Cubes, nil
}


func (l *Level) GetCubes(id string) ([]XMLCube, bool) {

  strapsFilePath, err := filepath.Abs("static/levels/City.area") //anoigw to arxeio. (lathos tropos) :)
    if err != nil {
        fmt.Println(err)
        os.Exit(1)
    }

    // Open the file
    file, err := os.Open(strapsFilePath)
    if err != nil {
        fmt.Println(err)
        os.Exit(1)
    }

    defer file.Close()

   xmlStraps, err := ReadCubes(file)
    if err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
    
return xmlStraps,true

}

	

